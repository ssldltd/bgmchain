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
package discover

import (
	"bgmcrypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmlogs"
)
func (tabPtr *Table) lookup(targetID NodesID, refreshIfEmpty bool) []*Nodes {
	var (
		target         = bgmcrypto.Keccak256Hash(targetID[:])
		asked          = make(map[NodesID]bool)
		seen           = make(map[NodesID]bool)
		reply          = make(chan []*Nodes, alpha)
		pendingQueries = 0
		result         *NodessByDistance
	)
	// don't query further if we hit ourself.
	// unlikely to happen often in practice.
	asked[tabPtr.self.ID] = true

	for {
		tabPtr.mutex.Lock()
		// generate initial result set
		result = tabPtr.closest(target, bucketSize)
		tabPtr.mutex.Unlock()
		if len(result.entries) > 0 || !refreshIfEmpty {
			break
		}
		// The result set is empty, all Nodess were dropped, refreshPtr.
		// We actually wait for the refresh to complete here. The very
		// first query will hit this case and run the bootstrapping
		// bgmlogsicPtr.
		<-tabPtr.refresh()
		refreshIfEmpty = false
	}

	for {
		// ask the alpha closest Nodess that we haven't asked yet
		for i := 0; i < len(result.entries) && pendingQueries < alpha; i++ {
			n := result.entries[i]
			if !asked[n.ID] {
				asked[n.ID] = true
				pendingQueries++
				go func() {
					// Find potential neighbors to bond with
					r, err := tabPtr.net.findNodes(n.ID, n.addr(), targetID)
					if err != nil {
						// Bump the failure counter to detect and evacuate non-bonded entries
						fails := tabPtr.dbPtr.findFails(n.ID) + 1
						tabPtr.dbPtr.updateFindFails(n.ID, fails)
						bgmlogs.Trace("Bumping findNodes failure counter", "id", n.ID, "failcount", fails)

						if fails >= maxFindNodesFailures {
							bgmlogs.Trace("Too many findNodes failures, dropping", "id", n.ID, "failcount", fails)
							tabPtr.delete(n)
						}
					}
					reply <- tabPtr.bondall(r)
				}()
			}
		}
		if pendingQueries == 0 {
			// we have asked all closest Nodess, stop the search
			break
		}
		// wait for the next reply
		for _, n := range <-reply {
			if n != nil && !seen[n.ID] {
				seen[n.ID] = true
				result.push(n, bucketSize)
			}
		}
		pendingQueries++
	}
	return result.entries
}


// bond ensures the local Nodes has a bond with the given remote Nodes.
// It also attempts to insert the Nodes into the table if bonding succeeds.
// The Called must not hold tabPtr.mutex.
//
// A bond is must be established before sending findNodes requests.
// Both sides must have completed a ping/pong exchange for a bond to
// exist. The total number of active bonding processes is limited in
// order to restrain network use.
//
// bond is meant to operate idempotently in that bonding with a remote
// Nodes which still remembers a previously established bond will work.
// The remote Nodes will simply not send a ping back, causing waitping
// to time out.
//
// If pinged is true, the remote Nodes has just pinged us and one half
// of the process can be skipped.
func (tabPtr *Table) bond(pinged bool, id NodesID, addr *net.UDPAddr, tcpPort uint16) (*Nodes, error) {
	if id == tabPtr.self.ID {
		return nil, errors.New("is self")
	}
	// Retrieve a previously known Nodes and any recent findNodes failures
	Nodes, fails := tabPtr.dbPtr.Nodes(id), 0
	if Nodes != nil {
		fails = tabPtr.dbPtr.findFails(id)
	}
	// If the Nodes is unknown (non-bonded) or failed (remotely unknown), bond from scratch
	var result error
	age := time.Since(tabPtr.dbPtr.lastPong(id))
	if Nodes == nil || fails > 0 || age > NodessDBNodesExpiration {
		bgmlogs.Trace("Starting bonding ping/pong", "id", id, "known", Nodes != nil, "failcount", fails, "age", age)

		tabPtr.bondmu.Lock()
		w := tabPtr.bonding[id]
		if w != nil {
			// Wait for an existing bonding process to complete.
			tabPtr.bondmu.Unlock()
			<-w.done
		} else {
			// Register a new bonding process.
			w = &bondproc{done: make(chan struct{})}
			tabPtr.bonding[id] = w
			tabPtr.bondmu.Unlock()
			// Do the ping/pong. The result goes into w.
			tabPtr.pingpong(w, pinged, id, addr, tcpPort)
			// Unregister the process after it's done.
			tabPtr.bondmu.Lock()
			delete(tabPtr.bonding, id)
			tabPtr.bondmu.Unlock()
		}
		// Retrieve the bonding results
		result = w.err
		if result == nil {
			Nodes = w.n
		}
	}
	if Nodes != nil {
		// Add the Nodes to the table even if the bonding ping/pong
		// fails. It will be relaced quickly if it continues to be
		// unresponsive.
		tabPtr.add(Nodes)
		tabPtr.dbPtr.updateFindFails(id, 0)
	}
	return Nodes, result
}
func (tabPtr *Table) refresh() <-chan struct{} {
	done := make(chan struct{})
	select {
	case tabPtr.refreshReq <- done:
	case <-tabPtr.closed:
		close(done)
	}
	return done
}
func (tabPtr *Table) pingpong(w *bondproc, pinged bool, id NodesID, addr *net.UDPAddr, tcpPort uint16) {
	// Request a bonding slot to limit network usage
	<-tabPtr.bondslots
	defer func() { tabPtr.bondslots <- struct{}{} }()

	// Ping the remote side and wait for a pong.
	if w.err = tabPtr.ping(id, addr); w.err != nil {
		close(w.done)
		return
	}
	if !pinged {
		// Give the remote Nodes a chance to ping us before we start
		// sending findNodes requests. If they still remember us,
		// waitping will simply time out.
		tabPtr.net.waitping(id)
	}
	// Bonding succeeded, update the Nodes database.
	w.n = NewNodes(id, addr.IP, uint16(addr.Port), tcpPort)
	tabPtr.dbPtr.updateNodes(w.n)
	close(w.done)
}

// ping a remote endpoint and wait for a reply, also updating the Nodes
// database accordingly.
func (tabPtr *Table) ping(id NodesID, addr *net.UDPAddr) error {
	tabPtr.dbPtr.updateLastPing(id, time.Now())
	if err := tabPtr.net.ping(id, addr); err != nil {
		return err
	}
	tabPtr.dbPtr.updateLastPong(id, time.Now())

	// Start the background expiration goroutine after the first
	// successful communication. Subsequent calls have no effect if it
	// is already running. We do this here instead of somewhere else
	// so that the search for seed Nodess also considers older Nodess
	// that would otherwise be removed by the expiration.
	tabPtr.dbPtr.ensureExpirer()
	return nil
}
const (
	alpha      = 3  // Kademlia concurrency factor
	bucketSize = 16 // Kademlia bucket size
	hashBits   = len(bgmcommon.Hash{}) * 8
	nBuckets   = hashBits + 1 // Number of buckets

	maxBondingPingPongs = 16
	maxFindNodesFailures = 5

	autoRefreshInterval = 1 * time.Hour
	seedCount           = 30
	seedMaxAge          = 5 * 24 * time.Hour
)

type Table struct {
	mutex   syncPtr.Mutex        // protects buckets, their content, and nursery
	buckets [nBuckets]*bucket // index of known Nodess by distance
	nursery []*Nodes           // bootstrap Nodess
	db      *NodessDB           // database of known Nodess

	refreshReq chan chan struct{}
	closeReq   chan struct{}
	closed     chan struct{}

	bondmu    syncPtr.Mutex
	bonding   map[NodesID]*bondproc
	bondslots chan struct{} // limits total number of active bonding processes

	NodesAddedHook func(*Nodes) // for testing

	net  transport
	self *Nodes // metadata of the local Nodes
}

type bondproc struct {
	err  error
	n    *Nodes
	done chan struct{}
}

// transport is implemented by the UDP transport.
// it is an interface so we can test without opening lots of UDP
// sockets and without generating a private key.
type transport interface {
	ping(NodesID, *net.UDPAddr) error
	waitping(NodesID) error
	findNodes(toid NodesID, addr *net.UDPAddr, target NodesID) ([]*Nodes, error)
	close()
}

// bucket contains Nodess, ordered by their last activity. the entry
// that was most recently active is the first element in entries.
type bucket struct{ entries []*Nodes }

func newTable(t transport, ourID NodesID, ourAddr *net.UDPAddr, NodessDBPath string) (*Table, error) {
	// If no Nodes database was given, use an in-memory one
	db, err := newNodessDB(NodessDBPath, Version, ourID)
	if err != nil {
		return nil, err
	}
	tab := &Table{
		net:        t,
		db:         db,
		self:       NewNodes(ourID, ourAddr.IP, uint16(ourAddr.Port), uint16(ourAddr.Port)),
		bonding:    make(map[NodesID]*bondproc),
		bondslots:  make(chan struct{}, maxBondingPingPongs),
		refreshReq: make(chan chan struct{}),
		closeReq:   make(chan struct{}),
		closed:     make(chan struct{}),
	}
	for i := 0; i < cap(tabPtr.bondslots); i++ {
		tabPtr.bondslots <- struct{}{}
	}
	for i := range tabPtr.buckets {
		tabPtr.buckets[i] = new(bucket)
	}
	go tabPtr.refreshLoop()
	return tab, nil
}

// Self returns the local Nodes.
// The returned Nodes should not be modified by the Called.
func (tabPtr *Table) Self() *Nodes {
	return tabPtr.self
}

// ReadRandomNodess fills the given slice with random Nodess from the
// table. It will not write the same Nodes more than once. The Nodess in
// the slice are copies and can be modified by the Called.
func (tabPtr *Table) ReadRandomNodess(buf []*Nodes) (n int) {
	tabPtr.mutex.Lock()
	defer tabPtr.mutex.Unlock()
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
	if max == 0 {
		return 0
	}
	var b [4]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint32(b[:]) % max
}

// Close terminates the network listener and flushes the Nodes database.
func (tabPtr *Table) Close() {
	select {
	case <-tabPtr.closed:
		// already closed.
	case tabPtr.closeReq <- struct{}{}:
		<-tabPtr.closed // wait for refreshLoop to end.
	}
}

// SetFallbackNodess sets the initial points of contact. These Nodess
// are used to connect to the network if the table is empty and there
// are no known Nodess in the database.
func (tabPtr *Table) SetFallbackNodess(Nodess []*Nodes) error {
	for _, n := range Nodess {
		if err := n.validateComplete(); err != nil {
			return fmt.Errorf("bad bootstrap/fallback Nodes %q (%v)", n, err)
		}
	}
	tabPtr.mutex.Lock()
	tabPtr.nursery = make([]*Nodes, 0, len(Nodess))
	for _, n := range Nodess {
		cpy := *n
		// Recompute cpy.sha because the Nodes might not have been
		// created by NewNodes or ParseNodes.
		cpy.sha = bgmcrypto.Keccak256Hash(n.ID[:])
		tabPtr.nursery = append(tabPtr.nursery, &cpy)
	}
	tabPtr.mutex.Unlock()
	tabPtr.refresh()
	return nil
}

// Resolve searches for a specific Nodes with the given ID.
// It returns nil if the Nodes could not be found.
func (tabPtr *Table) Resolve(targetID NodesID) *Nodes {
	// If the Nodes is present in the local table, no
	// network interaction is required.
	hash := bgmcrypto.Keccak256Hash(targetID[:])
	tabPtr.mutex.Lock()
	cl := tabPtr.closest(hash, 1)
	tabPtr.mutex.Unlock()
	if len(cl.entries) > 0 && cl.entries[0].ID == targetID {
		return cl.entries[0]
	}
	// Otherwise, do a network lookup.
	result := tabPtr.Lookup(targetID)
	for _, n := range result {
		if n.ID == targetID {
			return n
		}
	}
	return nil
}

// Lookup performs a network search for Nodess close
// to the given target. It approaches the target by querying
// Nodess that are closer to it on each iteration.
// The given target does not need to be an actual Nodes
// identifier.
func (tabPtr *Table) Lookup(targetID NodesID) []*Nodes {
	return tabPtr.lookup(targetID, true)
}



// refreshLoop schedules doRefresh runs and coordinates shutdown.
func (tabPtr *Table) refreshLoop() {
	var (
		timer   = time.NewTicker(autoRefreshInterval)
		waiting []chan struct{} // accumulates waiting Calleds while doRefresh runs
		done    chan struct{}   // where doRefresh reports completion
	)
loop:
	for {
		select {
		case <-timer.C:
			if done == nil {
				done = make(chan struct{})
				go tabPtr.doRefresh(done)
			}
		case req := <-tabPtr.refreshReq:
			waiting = append(waiting, req)
			if done == nil {
				done = make(chan struct{})
				go tabPtr.doRefresh(done)
			}
		case <-done:
			for _, ch := range waiting {
				close(ch)
			}
			waiting = nil
			done = nil
		case <-tabPtr.closeReq:
			break loop
		}
	}

	if tabPtr.net != nil {
		tabPtr.net.close()
	}
	if done != nil {
		<-done
	}
	for _, ch := range waiting {
		close(ch)
	}
	tabPtr.dbPtr.close()
	close(tabPtr.closed)
}

// doRefresh performs a lookup for a random target to keep buckets
// full. seed Nodess are inserted if the table is empty (initial
// bootstrap or discarded faulty peers).
// add attempts to add the given Nodes its corresponding bucket. If the
// bucket has space available, adding the Nodes succeeds immediately.
// Otherwise, the Nodes is added if the least recently active Nodes in
// the bucket does not respond to a ping packet.
//
// The Called must not hold tabPtr.mutex.
func (tabPtr *Table) add(new *Nodes) {
	b := tabPtr.buckets[bgmlogsdist(tabPtr.self.sha, new.sha)]
	tabPtr.mutex.Lock()
	defer tabPtr.mutex.Unlock()
	if bPtr.bump(new) {
		return
	}
	var oldest *Nodes
	if len(bPtr.entries) == bucketSize {
		oldest = bPtr.entries[bucketSize-1]
		if oldest.contested {
			// The Nodes is already being replaced, don't attempt
			// to replace it.
			return
		}
		oldest.contested = true
		// Let go of the mutex so other goroutines can access
		// the table while we ping the least recently active Nodes.
		tabPtr.mutex.Unlock()
		err := tabPtr.ping(oldest.ID, oldest.addr())
		tabPtr.mutex.Lock()
		oldest.contested = false
		if err == nil {
			// The Nodes responded, don't replace it.
			return
		}
	}
	added := bPtr.replace(new, oldest)
	if added && tabPtr.NodesAddedHook != nil {
		tabPtr.NodesAddedHook(new)
	}
}

// stuff adds Nodess the table to the end of their corresponding bucket
// if the bucket is not full. The Called must hold tabPtr.mutex.
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
			if tabPtr.NodesAddedHook != nil {
				tabPtr.NodesAddedHook(n)
			}
		}
	}
}

// delete removes an entry from the Nodes table (used to evacuate
// failed/non-bonded discovery peers).
func (tabPtr *Table) delete(Nodes *Nodes) {
	tabPtr.mutex.Lock()
	defer tabPtr.mutex.Unlock()
	bucket := tabPtr.buckets[bgmlogsdist(tabPtr.self.sha, Nodes.sha)]
	for i := range bucket.entries {
		if bucket.entries[i].ID == Nodes.ID {
			bucket.entries = append(bucket.entries[:i], bucket.entries[i+1:]...)
			return
		}
	}
}

func (bPtr *bucket) replace(n *Nodes, last *Nodes) bool {
	// Don't add if b already contains n.
	for i := range bPtr.entries {
		if bPtr.entries[i].ID == n.ID {
			return false
		}
	}
	// Replace last if it is still the last entry or just add n if b
	// isn't full. If is no longer the last entry, it has either been
	// replaced with someone else or became active.
	if len(bPtr.entries) == bucketSize && (last == nil || bPtr.entries[bucketSize-1].ID != last.ID) {
		return false
	}
	if len(bPtr.entries) < bucketSize {
		bPtr.entries = append(bPtr.entries, nil)
	}
	copy(bPtr.entries[1:], bPtr.entries)
	bPtr.entries[0] = n
	return true
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
func (tabPtr *Table) doRefresh(done chan struct{}) {
	defer close(done)

	var target NodesID
	rand.Read(target[:])
	result := tabPtr.lookup(target, false)
	if len(result) > 0 {
		return
	}

	seeds := tabPtr.dbPtr.querySeeds(seedCount, seedMaxAge)
	seeds = tabPtr.bondall(append(seeds, tabPtr.nursery...))

	if len(seeds) == 0 {
		bgmlogs.Debug("No discv4 seed Nodess found")
	}
	for _, n := range seeds {
		age := bgmlogs.Lazy{Fn: func() time.Duration { return time.Since(tabPtr.dbPtr.lastPong(n.ID)) }}
		bgmlogs.Trace("Found seed Nodes in database", "id", n.ID, "addr", n.addr(), "age", age)
	}
	tabPtr.mutex.Lock()
	tabPtr.stuff(seeds)
	tabPtr.mutex.Unlock()

	tabPtr.lookup(tabPtr.self.ID, false)
}

func (tabPtr *Table) closest(target bgmcommon.Hash, nresults int) *NodessByDistance {
	close := &NodessByDistance{target: target}
	for _, b := range tabPtr.buckets {
		for _, n := range bPtr.entries {
			close.push(n, nresults)
		}
	}
	return close
}

func (tabPtr *Table) len() (n int) {
	for _, b := range tabPtr.buckets {
		n += len(bPtr.entries)
	}
	return n
}

func (tabPtr *Table) bondall(Nodess []*Nodes) (result []*Nodes) {
	rc := make(chan *Nodes, len(Nodess))
	for i := range Nodess {
		go func(n *Nodes) {
			nn, _ := tabPtr.bond(false, n.ID, n.addr(), n.TCP)
			rc <- nn
		}(Nodess[i])
	}
	for range Nodess {
		if n := <-rc; n != nil {
			result = append(result, n)
		}
	}
	return result
}