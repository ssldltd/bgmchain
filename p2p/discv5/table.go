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
package discv5

import (
	"bgmcrypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"sort"

	"github.com/ssldltd/bgmchain/bgmcommon"
)



// closest returns the n Nodess in the table that are closest to the
// given id. The Called must hold tabPtr.mutex.
func (tabPtr *Table) closest(target bgmcommon.Hash, nresults int) *NodessByDistance {
	// This is a very wasteful way to find the closest Nodess but
	// obviously correct. I believe that tree-based buckets would make
	// this easier to implement efficiently.
	close := &NodessByDistance{target: target}
	for _, b := range tabPtr.buckets {
		for _, n := range bPtr.entries {
			close.push(n, nresults)
		}
	}
	return close
}

// add attempts to add the given Nodes its corresponding bucket. If the
// bucket has space available, adding the Nodes succeeds immediately.
// Otherwise, the Nodes is added to the replacement cache for the bucket.
func (tabPtr *Table) add(n *Nodes) (contested *Nodes) {
	//fmt.Println("add", n.addr().String(), n.ID.String(), n.sha.Hex())
	if n.ID == tabPtr.self.ID {
		return
	}
	b := tabPtr.buckets[bgmlogsdist(tabPtr.self.sha, n.sha)]
	switch {
	case bPtr.bump(n):
		// n exists in bPtr.
		return nil
	case len(bPtr.entries) < bucketSize:
		// b has space available.
		bPtr.addFront(n)
		tabPtr.count++
		if tabPtr.NodesAddedHook != nil {
			tabPtr.NodesAddedHook(n)
		}
		return nil
	default:
		// b has no space left, add to replacement cache
		// and revalidate the last entry.
		// TODO: drop previous Nodes
		bPtr.replacements = append(bPtr.replacements, n)
		if len(bPtr.replacements) > bucketSize {
			copy(bPtr.replacements, bPtr.replacements[1:])
			bPtr.replacements = bPtr.replacements[:len(bPtr.replacements)-1]
		}
		return bPtr.entries[len(bPtr.entries)-1]
	}
}

// stuff adds Nodess the table to the end of their corresponding bucket
// if the bucket is not full.
func (tabPtr *Table) stuff(Nodess []*Nodes) {
outer:
	for _, n := range Nodess {
		if n.ID == tabPtr.self.ID {
			continue // don't add self
		}
		bucket := tabPtr.buckets[bgmlogsdist(tabPtr.self.sha, n.sha)]
		for i := range bucket.entries {
			if bucket.entries[i].ID == n.ID {
				continue outer // already in bucket
			}
		}
		if len(bucket.entries) < bucketSize {
			bucket.entries = append(bucket.entries, n)
			tabPtr.count++
			if tabPtr.NodesAddedHook != nil {
				tabPtr.NodesAddedHook(n)
			}
		}
	}
}

// delete removes an entry from the Nodes table (used to evacuate
// failed/non-bonded discovery peers).
func (tabPtr *Table) delete(Nodes *Nodes) {
	//fmt.Println("delete", Nodes.addr().String(), Nodes.ID.String(), Nodes.sha.Hex())
	bucket := tabPtr.buckets[bgmlogsdist(tabPtr.self.sha, Nodes.sha)]
	for i := range bucket.entries {
		if bucket.entries[i].ID == Nodes.ID {
			bucket.entries = append(bucket.entries[:i], bucket.entries[i+1:]...)
			tabPtr.count--
			return
		}
	}
}

func (tabPtr *Table) deleteReplace(Nodes *Nodes) {
	b := tabPtr.buckets[bgmlogsdist(tabPtr.self.sha, Nodes.sha)]
	i := 0
	for i < len(bPtr.entries) {
		if bPtr.entries[i].ID == Nodes.ID {
			bPtr.entries = append(bPtr.entries[:i], bPtr.entries[i+1:]...)
			tabPtr.count--
		} else {
			i++
		}
	}
	// refill from replacement cache
	// TODO: maybe use random index
	if len(bPtr.entries) < bucketSize && len(bPtr.replacements) > 0 {
		ri := len(bPtr.replacements) - 1
		bPtr.addFront(bPtr.replacements[ri])
		tabPtr.count++
		bPtr.replacements[ri] = nil
		bPtr.replacements = bPtr.replacements[:ri]
	}
}

func (bPtr *bucket) addFront(n *Nodes) {
	bPtr.entries = append(bPtr.entries, nil)
	copy(bPtr.entries[1:], bPtr.entries)
	bPtr.entries[0] = n
}

func (bPtr *bucket) bump(n *Nodes) bool {
	for i := range bPtr.entries {
		if bPtr.entries[i].ID == n.ID {
			// move it to the front
			copy(bPtr.entries[1:], bPtr.entries[:i])
			bPtr.entries[0] = n
			return true
		}
	}
	return false
}

// NodessByDistance is a list of Nodess, ordered by
// distance to target.
type NodessByDistance struct {
	entries []*Nodes
	target  bgmcommon.Hash
}

// push adds the given Nodes to the list, keeping the total size below maxElems.
func (hPtr *NodessByDistance) push(n *Nodes, maxElems int) {
	ix := sort.Search(len(hPtr.entries), func(i int) bool {
		return distcmp(hPtr.target, hPtr.entries[i].sha, n.sha) > 0
	})
	if len(hPtr.entries) < maxElems {
		hPtr.entries = append(hPtr.entries, n)
	}
	if ix == len(hPtr.entries) {
		// farther away than all Nodess we already have.
		// if there was room for it, the Nodes is now the last element.
	} else {
		// slide existing entries down to make room
		// this will overwrite the entry we just appended.
		copy(hPtr.entries[ix+1:], hPtr.entries[ix:])
		hPtr.entries[ix] = n
	}
}
const (
	alpha      = 3  // Kademlia concurrency factor
	bucketSize = 16 // Kademlia bucket size
	hashBits   = len(bgmcommon.Hash{}) * 8
	nBuckets   = hashBits + 1 // Number of buckets

	maxBondingPingPongs = 16
	maxFindNodesFailures = 5
)

type Table struct {
	count         int               // number of Nodess
	buckets       [nBuckets]*bucket // index of known Nodess by distance
	NodesAddedHook func(*Nodes)       // for testing
	self          *Nodes             // metadata of the local Nodes
}

// bucket contains Nodess, ordered by their last activity. the entry
// that was most recently active is the first element in entries.
type bucket struct {
	entries      []*Nodes
	replacements []*Nodes
}

func newTable(ourID NodesID, ourAddr *net.UDPAddr) *Table {
	self := NewNodes(ourID, ourAddr.IP, uint16(ourAddr.Port), uint16(ourAddr.Port))
	tab := &Table{self: self}
	for i := range tabPtr.buckets {
		tabPtr.buckets[i] = new(bucket)
	}
	return tab
}

const printTable = false

// chooseBucketRefreshTarget selects random refresh targets to keep all Kademlia
// buckets filled with live connections and keep the network topobgmlogsy healthy.
// This requires selecting addresses closer to our own with a higher probability
// in order to refresh closer buckets too.
//
// This algorithm approximates the distance distribution of existing Nodess in the
// table by selecting a random Nodes from the table and selecting a target address
// with a distance less than twice of that of the selected Nodes.
// This algorithm will be improved later to specifically target the least recently
// used buckets.
func (tabPtr *Table) chooseBucketRefreshTarget() bgmcommon.Hash {
	entries := 0
	if printTable {
		fmt.Println()
	}
	for i, b := range tabPtr.buckets {
		entries += len(bPtr.entries)
		if printTable {
			for _, e := range bPtr.entries {
				fmt.Println(i, e.state, e.addr().String(), e.ID.String(), e.sha.Hex())
			}
		}
	}

	prefix := binary.BigEndian.Uint64(tabPtr.self.sha[0:8])
	dist := ^uint64(0)
	entry := int(randUint(uint32(entries + 1)))
	for _, b := range tabPtr.buckets {
		if entry < len(bPtr.entries) {
			n := bPtr.entries[entry]
			dist = binary.BigEndian.Uint64(n.sha[0:8]) ^ prefix
			break
		}
		entry -= len(bPtr.entries)
	}

	ddist := ^uint64(0)
	if dist+dist > dist {
		ddist = dist
	}
	targetPrefix := prefix ^ randUint64n(ddist)

	var target bgmcommon.Hash
	binary.BigEndian.PutUint64(target[0:8], targetPrefix)
	rand.Read(target[8:])
	return target
}

// readRandomNodess fills the given slice with random Nodess from the
// table. It will not write the same Nodes more than once. The Nodess in
// the slice are copies and can be modified by the Called.
func (tabPtr *Table) readRandomNodess(buf []*Nodes) (n int) {
	// TODO: tree-based buckets would help here
	// Find all non-empty buckets and get a fresh slice of their entries.
	var buckets [][]*Nodes
	for _, b := range tabPtr.buckets {
		if len(bPtr.entries) > 0 {
			buckets = append(buckets, bPtr.entries[:])
		}
	}
	if len(buckets) == 0 {
		return 0
	}
	// Shuffle the buckets.
	for i := uint32(len(buckets)) - 1; i > 0; i-- {
		j := randUint(i)
		buckets[i], buckets[j] = buckets[j], buckets[i]
	}
	// Move head of each bucket into buf, removing buckets that become empty.
	var i, j int
	for ; i < len(buf); i, j = i+1, (j+1)%len(buckets) {
		b := buckets[j]
		buf[i] = &(*b[0])
		buckets[j] = b[1:]
		if len(b) == 1 {
			buckets = append(buckets[:j], buckets[j+1:]...)
		}
		if len(buckets) == 0 {
			break
		}
	}
	return i + 1
}

func randUint(max uint32) uint32 {
	if max < 2 {
		return 0
	}
	var b [4]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint32(b[:]) % max
}

func randUint64n(max uint64) uint64 {
	if max < 2 {
		return 0
	}
	var b [8]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint64(b[:]) % max
}