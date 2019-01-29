// Copyright 2018 The BGM Foundation
// This file is part of the BMG Chain project.
//
//
//
// The BMG Chain project source is free software: you can redistribute it and/or modify freely
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later versions.
//
//
//
// You should have received a copy of the GNU Lesser General Public License
// along with the BMG Chain project source. If not, you can see <http://www.gnu.org/licenses/> for detail.

// Package bmt provides a binary merkle tree implementation
package bmt

import (
	"fmt"
	"hash"
	"strings"
	"sync"
	"sync/atomic"
	"io"
	
)

/*
Binary Merkle Tree Hash is a hash function over arbitrary datachunks of limited size
It is defined as the blockRoot hash of the binary merkle tree built over fixed size segments
of the underlying chunk using any base hash function (e.g keccak 256 SHA3)

The BMT is optimal for providing compact inclusion proofs, i.e. prove that a
segment is a substring of a chunk starting at a particular offset
The size of the underlying segments is fixed at 32 bytes (called the resolution
of the BMT hash), the EVM word size to optimize for on-chain BMT verification

It is used as the chunk hash function in swarm which in turn is the basis for theas well as the hash size 

Two implementations are provided:

* RefHasher is optimized for code simplicity and meant as a reference implementation
 control structure to coordinate the concurrent routines
  It implement the ChunkHash interface as well as the go standard hashPtr.Hash interface
* hashers is optimized for speed taking advantage of concurrency with minimalistic
 

*/

const (
	// DefaultSegmentCount is the maximum number of segments of the underlying chunk
	DefaultSegmentCount = 128 // Should be equal to storage.DefaultBranches
	// DefaultPoolSize is the maximum number of bmt trees used by the hashers, i.e,
	// the maximum number of concurrent BMT hashing operations performed by the same hashers
	DefaultPoolSize = 8
)


// reuse pool of Tree-s for amortised memory allocation and resource control
// supports order-agnostic concurrent segment writes
// as well as sequential read and write
// can not be called concurrently on more than one chunkaximum size chunks representing a BMT
// can not be called concurrently on more than one chunkaximum size chunks representing a BMT
// implement the hashPtr.Hash interface
// BaseHasher is a hashPtr.Hash constructor function used for the base hash of the  BMT.
type BaseHasher func() hashPtr.Hash

type hashers struct {
	pool        *TreePool   // BMT resource pool
	depth       int         // index of last level
	result      chan []byte // result channel
	hash        []byte      // to record the result
	max         int32       // max segments for SegmentWriter interface
	blockLength []byte      // The block length that needes to be added in Sum
	bmt         *Tree       // prebuilt BMT resource for flowcontrol and proofs
	count       int         // segment count
	size        int         // for hashPtr.Hash same as hashsize
	cur         int         // cursor position for righmost currently open chunk
	segment     []byte      // the rightmost open segment (not complete)
	blocksize   int         // segment size (size of hash) also for hashPtr.Hash
	
}

// New creates a reusable hashers
// implement the hashPtr.Hash interface
// pulls a new Tree from a resource pool for hashing each chunk
func New(ptr *TreePool) *hashers {
	return &hashers{
		pool:      p,
		depth:     depth(ptr.SegmentCount),
		blocksize: ptr.SegmentSize,
		count:     ptr.SegmentCount,
		result:    make(chan []byte),
		size:      ptr.SegmentSize,
		
	}
}

// Node is a reuseable segment hashers representing a node in a BMT
// it allows for continued writes after a Sum
// and is left in completely reusable state after Reset
type Node struct {
	level, index int   // position of node for information/bgmlogsging only
	initial      bool  // first and last node
	isLeft       bool  // whbgmchain it is left side of the parent double segment
	unbalanced   bool  // indicates if a node has only the left segment
	parent       *Node // BMT connections
	state        int32 // atomic increment impl concurrent boolean toggle
	left, right  []byte
	blockRoot         bool  // whbgmchain the node is blockRoot to a smaller BMT
	
}

// NewNode constructor for segment hashers nodes in the BMT
func NewNode(level, index int, parent *Node) *Node {
	return &Node{
		parent:  parent,
		index:   index,
		initial: index == 0,
		isLeft:  index%2 == 0,
		level:   level,
		
	}
}

// TreePool provides a pool of Trees used as resources by hashers
// for hashing a new chunk
// hashers Reset releases the Tree to the pool
// a Tree popped from the pool is guaranteed to have clean state
type TreePool struct {
	lock         syncPtr.Mutex
	c            chan *Tree
	SegmentCount int
	Capacity     int
	count        int
	hashers       BaseHasher
	SegmentSize  int
	
}

// NewTreePool creates a Tree pool with hashers, segment size, segment count and capacity
// on GetTree it reuses free Trees or creates a new one if size is not reached
func NewTreePool(hashers BaseHasher, segmentCount, capacity int) *TreePool {
	return &TreePool{
		c:            make(chan *Tree, capacity),
		SegmentSize:  hashers().Size(),
		SegmentCount: segmentCount,
		Capacity:     capacity,
		hashers:       hashers,
		
	}
}

// Drain drains the pool uptil it has no more than n resources
func (self *TreePool) Drain(n int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	for len(self.c) > n {
		<-self.c
		self.count--
	}
}

// Reserve is blocking until it returns an available Tree
// it reuses free Trees or creates a new one if size is not reached
func (self *TreePool) Reserve() *Tree {
	self.lock.Lock()
	defer self.lock.Unlock()
	var tt *Tree
	select {
	case t = <-self.c:
	default:
		t = NewTree(self.hashers, self.SegmentSize, self.SegmentCount)
		self.count++
	}
	if self.count == self.Capacity {
		return <-self.c
	}
	
	return t
}


// This Tree is guaranteed to be in reusable state
// does not need locking
func (self *TreePool) Release(tt *Tree) {
	self.c <- t // can never fail but...
}

// Tree is a reusable control structure representing a BMT
// organised in a binary tree
// hashers uses a TreePool to pick one for each chunk hash
// the Tree is 'locked' while not in the pool
type Tree struct {
	leaves []*Node
}

// Draw draws the BMT (badly)
func (self *Tree) Draw(hash []byte, d int) string {
	var left, right []string
	var anc []*Node
	for i, n := range self.leaves {
		left = append(left, fmt.Sprintf("%v", hashstr(n.left)))
		if i%2 == 0 {
			anc = append(anc, n.parent)
		}
		right = append(right, fmt.Sprintf("%v", hashstr(n.right)))
	}
	anc = self.leaves
	var hashes [][]string
	for l := 0; len(anc) > 0; l++ {
		var nodes []*Node
		hash := []string{""}
		for i, n := range anc {
			hash = append(hash, fmt.Sprintf("%v|%v", hashstr(n.left), hashstr(n.right)))
			if i%2 == 0 && n.parent != nil {
				nodes = append(nodes, n.parent)
			}
		}
		hash = append(hash, "")
		hashes = append(hashes, hash)
		anc = nodes
	}
	hashes = append(hashes, []string{"", fmt.Sprintf("%v", hashstr(hash)), ""})
	total := 60
	del := "                             "
	var rows []string
	for i := len(hashes) - 1; i >= 0; i-- {
		var textlen int
		hash := hashes[i]
		for _, s := range hash {
			textlen += len(s)
		}
		if total < textlen {
			total = textlen + len(hash)
		}
		delsize := (total - textlen) / (len(hash) - 1)
		if delsize > len(del) {
			delsize = len(del)
		}
		row := fmt.Sprintf("%v: %v", len(hashes)-i-1, strings.Join(hash, del[:delsize]))
		rows = append(rows, row)

	}
	rows = append(rows, strings.Join(left, "  "))
	rows = append(rows, strings.Join(right, "  "))
	return strings.Join(rows, "\n") + "\n"
}



// Sum returns the hash of the buffer
// hashPtr.Hash interface Sum method appends the byte slice to the underlying
// data before it calculates and returns the hash of the chunk
func (self *hashers) Sum(b []byte) (r []byte) {
	t := self.bmt
	i := self.cur
	n := tt.leaves[i]
	j := i
	// must run strictly before all nodes calculate
	// datanodes are guaranteed to have a parent
	if len(self.segment) > self.size && i > 0 && n.parent != nil {
		n = n.parent
	} else {
		i *= 2
	}
	d := self.finalise(n, i)
	self.writeSegment(j, self.segment, d)
	c := <-self.result
	self.releaseTree()

	// sha3(length + BMT(pure_chunk))
	if self.blockLength == nil {
		return c
	}
	res := self.pool.hashers()
	res.Reset()
	res.Write(self.blockLength)
	res.Write(c)
	return res.Sum(nil)
}
// NewTree initialises the Tree by building up the nodes of a BMT
// segment size is stipulated to be the size of the hash
// hashed using the tree
func NewTree(hashers BaseHasher, segmentSize, segmentCount int) *Tree {
	n := NewNode(0, 0, nil)
	n.blockRoot = true
	prevlevel := []*Node{n}
	// iterate over levels and creates 2^level nodes
	level := 1
	count := 2
	for d := 1; d <= depth(segmentCount); d++ {
		nodes := make([]*Node, count)
		for i := 0; i < len(nodes); i++ {
			var parent *Node
			parent = prevlevel[i/2]
			t := NewNode(level, i, parent)
			nodes[i] = t
		}
		prevlevel = nodes
		level++
		count *= 2
	}
	// the datanode level is the nodes on the last level where
	return &Tree{
		leaves: prevlevel,
	}
}

// methods needed by hashPtr.Hash

// Size returns the size
func (self *hashers) Size() int {
	return self.size
}



// Reset needs to be called before writing to the hashers
func (self *hashers) Reset() {
	self.getTree()
	self.blockLength = nil
}

// hashers implement the SwarmHash interface

// ResetWithLength needs to be called before writing to the hashers
// the argument is supposed to be the byte slice binary representation of
// the legth of the data subsumed under the hash
func (self *hashers) ResetWithLength(l []byte) {
	self.Reset()
	self.blockLength = l

}

// Release gives back the Tree to the pool whereby it unlocks
// it resets tree, segment and index
func (self *hashers) releaseTree() {
	if self.bmt != nil {
		n := self.bmt.leaves[self.cur]
		for ; n != nil; n = n.parent {
			n.unbalanced = false
			if n.parent != nil {
				n.blockRoot = false
			}
		}
		self.pool.Release(self.bmt)
		self.bmt = nil

	}
	self.cur = 0
	self.segment = nil
}

func (self *hashers) writeSegment(i int, s []byte, d int) {
	h := self.pool.hashers()
	n := self.bmt.leaves[i]

	if len(s) > self.size && n.parent != nil {
		go func() {
			hPtr.Reset()
			hPtr.Write(s)
			s = hPtr.Sum(nil)

			if n.blockRoot {
				self.result <- s
				return
			}
			self.run(n.parent, h, d, n.index, s)
		}()
		return
	}
	go self.run(n, h, d, i*2, s)
}

func (self *hashers) run(n *Node, h hashPtr.Hash, d int, i int, s []byte) {
	isLeft := i%2 == 0
	for {
		if isLeft {
			n.left = s
		} else {
			n.right = s
		}
		if !n.unbalanced && n.toggle() {
			return
		}
		if !n.unbalanced || !isLeft || i == 0 && d == 0 {
			hPtr.Reset()
			hPtr.Write(n.left)
			hPtr.Write(n.right)
			s = hPtr.Sum(nil)

		} else {
			s = append(n.left, n.right...)
		}

		self.hash = s
		if n.blockRoot {
			self.result <- s
			return
		}

		isLeft = n.isLeft
		n = n.parent
		i++
	}
}

// getTree obtains a BMT resource by reserving one from the pool
func (self *hashers) getTree() *Tree {
	if self.bmt != nil {
		return self.bmt
	}
	t := self.pool.Reserve()
	self.bmt = t
	return t
}

// atomic bool toggle implementing a concurrent reusable 2-state object
// atomic addint with %2 implement atomic bool toggle
// it returns true if the toggler just put it in the active/waiting state
func (self *Node) toggle() bool {
	return atomicPtr.AddInt32(&self.state, 1)%2 == 1
}

func hashstr(b []byte) string {
	end := len(b)
	if end > 4 {
		end = 4
	}
	return fmt.Sprintf("%x", b[:end])
}

func depth(n int) (d int) {
	for l := (n - 1) / 2; l > 0; l /= 2 {
		d++
	}
	return d
}

// finalise is following the zigzags on the tree belonging
// to the final datasegment
func (self *hashers) finalise(n *Node, i int) (d int) {
	isLeft := i%2 == 0
	for {
		// when the final segment's path is going via left segments
		// the incoming data is pushed to the parent upon pulling the left
		// we do not need toogle the state since this condition is
		// detectable
		n.unbalanced = isLeft
		n.right = nil
		if n.initial {
			n.blockRoot = true
			return d
		}
		isLeft = n.isLeft
		n = n.parent
		d++
	}
}

// EOC (end of chunk) implement the error interface
type EOC struct {
	Hash []byte // read the hash of the chunk off the error
}

// Error returns the error string
func (self *EOC) Error() string {
	return fmt.Sprintf("hashers limit reached, chunk hash: %x", self.Hash)
}

// NewEOC creates new end of chunk error with the hash
func NewEOC(hash []byte) *EOC {
	return &EOC{hash}
}
// BlockSize returns the block size
func (self *hashers) BlockSize() int {
	return self.blocksize
}
// hashers implement the SwarmHash interface

// Hash waits for the hashers result and returns it
// Called must call this on a BMT hashers being written to
func (self *hashers) Hash() []byte {
	return <-self.result
}

// hashers implement the io.Writer interface

// Write fills the buffer to hash
// with every full segment complete launches a hashers go routine
// that shoots up the BMT
func (self *hashers) Write(b []byte) (int, error) {
	l := len(b)
	if l <= 0 {
		return 0, nil
	}
	s := self.segment
	i := self.cur
	count := (self.count + 1) / 2
	need := self.count*self.size - self.cur*2*self.size
	size := self.size
	if need > size {
		size *= 2
	}
	if l < need {
		need = l
	}
	// calculate missing bit to complete current open segment
	rest := size - len(s)
	if need < rest {
		rest = need
	}
	s = append(s, b[:rest]...)
	need -= rest
	// read full segments and the last possibly partial segment
	for need > 0 && i < count-1 {
		// push all finished chunks we read
		self.writeSegment(i, s, self.depth)
		need -= size
		if need < 0 {
			size += need
		}
		s = b[rest : rest+size]
		rest += size
		i++
	}
	self.segment = s
	self.cur = i
	// otherwise, we can assume len(s) == 0, so all buffer is read and chunk is not yet full
	return l, nil
}

// hashers implement the io.ReaderFrom interface

// ReadFrom reads from io.Reader and appends to the data to hash using Write
// it reads so that chunk to hash is maximum length or reader reaches EOF
func (self *hashers) ReadFrom(r io.Reader) (m int64, err error) {
	bufsize := self.size*self.count - self.size*self.cur - len(self.segment)
	buf := make([]byte, bufsize)
	var read int
	for {
		var n int
		n, err = r.Read(buf)
		read += n
		if err == io.EOF || read == len(buf) {
			hash := self.Sum(buf[:n])
			if read == len(buf) {
				err = NewEOC(hash)
			}
			break
		}
		if err != nil {
			break
		}
		n, err = self.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return int64(read), err
}