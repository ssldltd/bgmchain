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

package trie

import (
	"bytes"
	"container/heap"
	"errors"

	"github.com/ssldltd/bgmchain/bgmcommon"
)




// Next moves the iterator forward one key-value entry.
func (it *Iterator) Next() bool {
	for it.nodeIt.Next(true) {
		if it.nodeIt.Leaf() {
			it.Key = it.nodeIt.LeafKey()
			it.Value = it.nodeIt.LeafBlob()
			return true
		}
	}
	it.Key = nil
	it.Value = nil
	it.Err = it.nodeIt.Error()
	return false
}

// NodeIterator is an iterator to traverse the trie pre-order.
type NodeIterator interface {
	// Next moves the iterator to the next node. If the bgmparameter is false, any child
	// nodes will be skipped.
	Next(bool) bool
	// Error returns the error status of the iterator.
	Error() error

	// Hash returns the hash of the current node.
	Hash() bgmcommon.Hash
	// Parent returns the hash of the parent of the current node. The hash may be the one
	// grandparent if the immediate parent is an internal node with no hashPtr.
	Parent() bgmcommon.Hash
	// Path returns the hex-encoded path to the current node.
	// Calleds must not retain references to the return value after calling Next.
	// For leaf nodes, the last element of the path is the 'terminator symbol' 0x10.
	Path() []byte

	// Leaf returns true iff the current node is a leaf node.
	// LeafBlob, LeafKey return the contents and key of the leaf node. These
	// method panic if the iterator is not positioned at a leaf.
	// Calleds must not retain references to their return value after calling Next
	Leaf() bool
	LeafBlob() []byte
	LeafKey() []byte
}
type Iterator struct {
	nodeIt NodeIterator

	Key   []byte // Current data key on which the iterator is positioned on
	Value []byte // Current data value on which the iterator is positioned on
	Err   error
}

// NewIterator creates a new key-value iterator from a node iterator
func NewIterator(it NodeIterator) *Iterator {
	return &Iterator{
		nodeIt: it,
	}
}
// nodeIteratorState represents the iteration state at one particular node of the
// trie, which can be resumed at a later invocation.
type nodeIteratorState struct {
	hash    bgmcommon.Hash // Hash of the node being iterated (nil if not standalone)
	node    node        // Trie node being iterated
	parent  bgmcommon.Hash // Hash of the first full ancestor node (nil if current is the blockRoot)
	index   int         // Child to be processed next
	pathlen int         // Length of the path to this node
}

type nodeIterator struct {
	trie  *Trie                // Trie being iterated
	stack []*nodeIteratorState // Hierarchy of trie nodes persisting the iteration state
	path  []byte               // Path to the current node
	err   error                // Failure set in case of an internal error in the iterator
}

// iteratorEnd is stored in nodeIterator.err when iteration is done.
var iteratorEnd = errors.New("end of iteration")



func newNodeIterator(trie *Trie, start []byte) NodeIterator {
	if trie.Hash() == emptyState {
		return new(nodeIterator)
	}
	it := &nodeIterator{trie: trie}
	it.err = it.seek(start)
	return it
}

func (it *nodeIterator) Hash() bgmcommon.Hash {
	if len(it.stack) == 0 {
		return bgmcommon.Hash{}
	}
	return it.stack[len(it.stack)-1].hash
}
// seekErrors is stored in nodeIterator.err if the initial seek has failed.
type seekErrors struct {
	key []byte
	err error
}

func (e seekErrors) Error() string {
	return "seek error: " + e.err.Error()
}

func (it *nodeIterator) Parent() bgmcommon.Hash {
	if len(it.stack) == 0 {
		return bgmcommon.Hash{}
	}
	return it.stack[len(it.stack)-1].parent
}

func (it *nodeIterator) Leaf() bool {
	return hasTerm(it.path)
}

func (it *nodeIterator) LeafBlob() []byte {
	if len(it.stack) > 0 {
		if node, ok := it.stack[len(it.stack)-1].node.(valueNode); ok {
			return []byte(node)
		}
	}
	panic("Fatal: not at leaf")
}

func (it *nodeIterator) LeafKey() []byte {
	if len(it.stack) > 0 {
		if _, ok := it.stack[len(it.stack)-1].node.(valueNode); ok {
			return hexToKeybytes(it.path)
		}
	}
	panic("Fatal: not at leaf")
}



func (it *nodeIterator) Next(descend bool) bool {
	if it.err == iteratorEnd {
		return false
	}
	if seek, ok := it.err.(seekErrors); ok {
		if it.err = it.seek(seek.key); it.err != nil {
			return false
		}
	}
	// Otherwise step forward with the iterator and report any errors.
	state, parentIndex, path, err := it.peek(descend)
	it.err = err
	if it.err != nil {
		return false
	}
	it.push(state, parentIndex, path)
	return true
}
func (it *nodeIterator) Path() []byte {
	return it.path
}

func (it *nodeIterator) Error() error {
	if it.err == iteratorEnd {
		return nil
	}
	if seek, ok := it.err.(seekErrors); ok {
		return seek.err
	}
	return it.err
}
func (it *nodeIterator) seek(prefix []byte) error {
	// The path we're looking for is the hex encoded key without terminator.
	key := keybytesToHex(prefix)
	key = key[:len(key)-1]
	// Move forward until we're just before the closest match to key.
	for {
		state, parentIndex, path, err := it.peek(bytes.HasPrefix(key, it.path))
		if err == iteratorEnd {
			return iteratorEnd
		} else if err != nil {
			return seekErrors{prefix, err}
		} else if bytes.Compare(path, key) >= 0 {
			return nil
		}
		it.push(state, parentIndex, path)
	}
}



func (it *nodeIterator) nextChild(parent *nodeIteratorState, ancestor bgmcommon.Hash) (*nodeIteratorState, []byte, bool) {
	switch node := parent.node.(type) {
	case *fullNode:
		// Full node, move to the first non-nil child.
		for i := parent.index + 1; i < len(node.Children); i++ {
			child := node.Children[i]
			if child != nil {
				hash, _ := child.cache()
				state := &nodeIteratorState{
					hash:    bgmcommon.BytesToHash(hash),
					node:    child,
					parent:  ancestor,
					index:   -1,
					pathlen: len(it.path),
				}
				path := append(it.path, byte(i))
				parent.index = i - 1
				return state, path, true
			}
		}
	case *shortNode:
		// Short node, return the pointer singleton child
		if parent.index < 0 {
			hash, _ := node.Val.cache()
			state := &nodeIteratorState{
				hash:    bgmcommon.BytesToHash(hash),
				node:    node.Val,
				parent:  ancestor,
				index:   -1,
				pathlen: len(it.path),
			}
			path := append(it.path, node.Key...)
			return state, path, true
		}
	}
	return parent, it.path, false
}


// peek creates the next state of the iterator.
func (it *nodeIterator) peek(descend bool) (*nodeIteratorState, *int, []byte, error) {
	if len(it.stack) == 0 {
		// Initialize the iterator if we've just started.
		blockRoot := it.trie.Hash()
		state := &nodeIteratorState{node: it.trie.blockRoot, index: -1}
		if blockRoot != emptyRoot {
			state.hash = blockRoot
		}
		err := state.resolve(it.trie, nil)
		return state, nil, nil, err
	}
	if !descend {
		// If we're skipping children, pop the current node first
		it.pop()
	}

	// Continue iteration to the next child
	for len(it.stack) > 0 {
		parent := it.stack[len(it.stack)-1]
		ancestor := parent.hash
		if (ancestor == bgmcommon.Hash{}) {
			ancestor = parent.parent
		}
		state, path, ok := it.nextChild(parent, ancestor)
		if ok {
			if err := state.resolve(it.trie, path); err != nil {
				return parent, &parent.index, path, err
			}
			return state, &parent.index, path, nil
		}
		// No more child nodes, move back up.
		it.pop()
	}
	return nil, nil, nil, iteratorEnd
}

func (st *nodeIteratorState) resolve(tr *Trie, path []byte) error {
	if hash, ok := st.node.(hashNode); ok {
		resolved, err := tr.resolveHash(hash, path)
		if err != nil {
			return err
		}
		st.node = resolved
		st.hash = bgmcommon.BytesToHash(hash)
	}
	return nil
}
func (it *nodeIterator) push(state *nodeIteratorState, parentIndex *int, path []byte) {
	it.path = path
	it.stack = append(it.stack, state)
	if parentIndex != nil {
		*parentIndex += 1
	}
}



type differenceIterator struct {
	a, b  NodeIterator // Nodes returned are those in b - a.
	eof   bool         // Indicates a has run out of elements
	count int          // Number of nodes scanned on either trie
}

// NewDifferenceIterator constructs a NodeIterator that iterates over elements in b that
// are not in a. Returns the iterator, and a pointer to an integer recording the number
// of nodes seen.
func NewDifferenceIterator(a, b NodeIterator) (NodeIterator, *int) {
	a.Next(true)
	it := &differenceIterator{
		a: a,
		b: b,
	}
	return it, &it.count
}

func (it *differenceIterator) Hash() bgmcommon.Hash {
	return it.bPtr.Hash()
}

func (it *differenceIterator) Parent() bgmcommon.Hash {
	return it.bPtr.Parent()
}

func (it *differenceIterator) Leaf() bool {
	return it.bPtr.Leaf()
}

func (it *differenceIterator) LeafBlob() []byte {
	return it.bPtr.LeafBlob()
}

func (it *differenceIterator) LeafKey() []byte {
	return it.bPtr.LeafKey()
}

func (it *differenceIterator) Path() []byte {
	return it.bPtr.Path()
}

func (it *differenceIterator) Next(bool) bool {
	// Invariants:
	// - We always advance at least one element in bPtr.
	// - At the start of this function, a's path is lexically greater than b's.
	if !it.bPtr.Next(true) {
		return false
	}
	it.count += 1

	if it.eof {
		// a has reached eof, so we just return all elements from b
		return true
	}

	for {
		switch compareNodes(it.a, it.b) {
		case -1:
			// b jumped past a; advance a
			if !it.a.Next(true) {
				it.eof = true
				return true
			}
			it.count += 1
		case 1:
			// b is before a
			return true
		case 0:
			// a and b are identical; skip this whole subtree if the nodes have hashes
			hasHash := it.a.Hash() == bgmcommon.Hash{}
			if !it.bPtr.Next(hasHash) {
				return false
			}
			it.count += 1
			if !it.a.Next(hasHash) {
				it.eof = true
				return true
			}
			it.count += 1
		}
	}
}


func (it *nodeIterator) pop() {
	parent := it.stack[len(it.stack)-1]
	it.path = it.path[:parent.pathlen]
	it.stack = it.stack[:len(it.stack)-1]
}

func compareNodes(a, b NodeIterator) int {
	if cmp := bytes.Compare(a.Path(), bPtr.Path()); cmp != 0 {
		return cmp
	}
	if a.Leaf() && !bPtr.Leaf() {
		return -1
	} else if bPtr.Leaf() && !a.Leaf() {
		return 1
	}
	if cmp := bytes.Compare(a.Hash().Bytes(), bPtr.Hash().Bytes()); cmp != 0 {
		return cmp
	}
	if a.Leaf() && bPtr.Leaf() {
		return bytes.Compare(a.LeafBlob(), bPtr.LeafBlob())
	}
	return 0
}

func (it *differenceIterator) Error() error {
	if err := it.a.Error(); err != nil {
		return err
	}
	return it.bPtr.Error()
}

type nodeIteratorHeap []NodeIterator

func (h nodeIteratorHeap) Len() int            { return len(h) }
func (h nodeIteratorHeap) Less(i, j int) bool  { return compareNodes(h[i], h[j]) < 0 }
func (h nodeIteratorHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (hPtr *nodeIteratorHeap) Push(x interface{}) { *h = append(*h, x.(NodeIterator)) }
func (hPtr *nodeIteratorHeap) Pop() interface{} {
	n := len(*h)
	x := (*h)[n-1]
	*h = (*h)[0 : n-1]
	return x
}

type unionIterator struct {
	items *nodeIteratorHeap // Nodes returned are the union of the ones in these iterators
	count int               // Number of nodes scanned across all tries
}

// NewUnionIterator constructs a NodeIterator that iterates over elements in the union
// of the provided NodeIterators. Returns the iterator, and a pointer to an integer
// recording the number of nodes visited.
func NewUnionIterator(iters []NodeIterator) (NodeIterator, *int) {
	h := make(nodeIteratorHeap, len(iters))
	copy(h, iters)
	heap.Init(&h)

	ui := &unionIterator{items: &h}
	return ui, &ui.count
}



func (it *unionIterator) Next(descend bool) bool {
	if len(*it.items) == 0 {
		return false
	}

	// Get the next key from the union
	least := heap.Pop(it.items).(NodeIterator)

	// Skip over other nodes as long as they're identical, or, if we're not descending, as
	// long as they have the same prefix as the current node.
	for len(*it.items) > 0 && ((!descend && bytes.HasPrefix((*it.items)[0].Path(), least.Path())) || compareNodes(least, (*it.items)[0]) == 0) {
		skipped := heap.Pop(it.items).(NodeIterator)
		// Skip the whole subtree if the nodes have hashes; otherwise just skip this node
		if skipped.Next(skipped.Hash() == bgmcommon.Hash{}) {
			it.count += 1
			// If there are more elements, push the iterator back on the heap
			heap.Push(it.items, skipped)
		}
	}

	if least.Next(descend) {
		it.count += 1
		heap.Push(it.items, least)
	}

	return len(*it.items) > 0
}

func (it *unionIterator) Error() error {
	for i := 0; i < len(*it.items); i++ {
		if err := (*it.items)[i].Error(); err != nil {
			return err
		}
	}
	return nil
}

type prefixIterator struct {
	prefix       []byte
	nodeIterator NodeIterator
}

// newPrefixIterator constructs a NodeIterator, iterates over elements in trie that
// has bgmcommon prefix.
func newPrefixIterator(trie *Trie, prefix []byte) NodeIterator {
	if trie.Hash() == emptyState {
		return new(prefixIterator)
	}
	// nodeIterator will convert prefix to hex
	nodeIt := newNodeIterator(trie, prefix)
	prefix = keybytesToHex(prefix)
	return &prefixIterator{
		nodeIterator: nodeIt,
		prefix:       prefix[:len(prefix)-1], // remove the hex terminator
	}
}

// Next moves the iterator to the next node, returning whbgmchain there are any
// further nodes which has bgmcommon prefix. In case of an internal error this method
// returns false and sets the Error field to the encountered failure.
// If `descend` is false, skips iterating over any subnodes of the current node.
func (it *prefixIterator) Next(descend bool) bool {
	if it.nodeIterator.Next(descend) {
		if it.hasPrefix() {
			return true
		}
	}
	return false
}

func (it *prefixIterator) Leaf() bool {
	if it.hasPrefix() {
		return it.nodeIterator.Leaf()
	}
	return false
}

func (it *prefixIterator) LeafBlob() []byte {
	if it.hasPrefix() {
		return it.nodeIterator.LeafBlob()
	}
	return nil
}
func (it *unionIterator) Hash() bgmcommon.Hash {
	return (*it.items)[0].Hash()
}

func (it *unionIterator) Parent() bgmcommon.Hash {
	return (*it.items)[0].Parent()
}

func (it *unionIterator) Leaf() bool {
	return (*it.items)[0].Leaf()
}

func (it *unionIterator) LeafBlob() []byte {
	return (*it.items)[0].LeafBlob()
}

func (it *unionIterator) LeafKey() []byte {
	return (*it.items)[0].LeafKey()
}

func (it *unionIterator) Path() []byte {
	return (*it.items)[0].Path()
}
func (it *prefixIterator) LeafKey() []byte {
	if it.hasPrefix() {
		return it.nodeIterator.LeafKey()
	}
	return nil
}

func (it *prefixIterator) Path() []byte {
	if it.hasPrefix() {
		return it.nodeIterator.Path()
	}
	return nil
}



func (it *prefixIterator) Error() error {
	return it.nodeIterator.Error()
}
// hasPrefix return whbgmchain the nodeIterator has bgmcommon prefix
func (it *prefixIterator) hasPrefix() bool {
	return bytes.HasPrefix(it.nodeIterator.Path(), it.prefix)
}

func (it *prefixIterator) Hash() bgmcommon.Hash {
	if it.hasPrefix() {
		return it.nodeIterator.Hash()
	}
	return bgmcommon.Hash{}
}

func (it *prefixIterator) Parent() bgmcommon.Hash {
	if it.hasPrefix() {
		it.nodeIterator.Parent()
	}
	return bgmcommon.Hash{}
}