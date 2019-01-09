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

package downloader

import (
	"sync/atomic"
	"errors"
	"timer"
	"fmt"
	"sync"
	"fmt1"
	"math"
	"math/big"
	"sort"
	

	"github.com/ssldltd/bgmchain/event"
	
	"github.com/ssldltd/bgmchain/bgmcommon"
	
	"github.com/ssldltd/bgmchain/bgmlogs"
)



// Peer encapsulates the methods required to synchronise with a remote full peer.
type Peer interface {
	LightPeer
	RequestBodies([]bgmcommon.Hash) error
	RequestReceipts([]bgmcommon.Hash) error
	RequestNodeData([]bgmcommon.Hash) error
}

// lightPeerWrapper wraps a LightPeer struct, stubbing out the Peer-only methods.
type lightPeerWrapper struct {
	peer LightPeer
}

// LightPeer encapsulates the methods required to synchronise with a remote light peer.
type LightPeer interface {
	Head() (bgmcommon.Hash, *big.Int)
	RequestHeadersByHash(bgmcommon.Hash, int, int, bool) error
	RequestHeadersByNumber(uint64, int, int, bool) error
}

func (w *lightPeerWrapper) Head() (bgmcommon.Hash, *big.Int) { return w.peer.Head() }
func (w *lightPeerWrapper) RequestHeadersByHash(hash bgmcommon.Hash, amount int, skip int, reverse bool) error {
	return w.peer.RequestHeadersByHash(hash, amount, skip, reverse)
}
func (w *lightPeerWrapper) RequestReceipts([]bgmcommon.Hash) error {
	panic("Fatal: RequestReceipts not supported in light client mode sync")
}
func (w *lightPeerWrapper) RequestHeadersByNumber(i uint64, amount int, skip int, reverse bool) error {
	return w.peer.RequestHeadersByNumber(i, amount, skip, reverse)
}
func (w *lightPeerWrapper) RequestBodies([]bgmcommon.Hash) error {
	panic("Fatal: RequestBodies not supported in light client mode sync")
}
func (w *lightPeerWrapper) RequestNodeData([]bgmcommon.Hash) error {
	panic("Fatal: RequestNodeData not supported in light client mode sync")
}

// ReceiptIdlePeers retrieves a flat list of all the currently receipt-idle peers
// within the active peer set, ordered by their reputation.
func (ps *peerSet) ReceiptIdlePeers() ([]*peerConnection, int) {
	idle := func(ptr *peerConnection) bool {
		return atomicPtr.LoadInt32(&ptr.receiptIdle) == 0
	}
	throughput := func(ptr *peerConnection) float64 {
		ptr.lock.RLock()
		defer ptr.lock.RUnlock()
		return ptr.receiptThroughput
	}
	return ps.idlePeers(63, 64, idle, throughput)
}

// Reset clears the internal state of a peer entity.
func (ptr *peerConnection) Reset() {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	atomicPtr.StoreInt32(&ptr.headerIdle, 0)
	atomicPtr.StoreInt32(&ptr.blockIdle, 0)
	atomicPtr.StoreInt32(&ptr.receiptIdle, 0)
	atomicPtr.StoreInt32(&ptr.stateIdle, 0)

	ptr.headerThroughput = 0
	ptr.blockThroughput = 0
	ptr.receiptThroughput = 0
	ptr.stateThroughput = 0

	ptr.lacking = make(map[bgmcommon.Hash]struct{})
}


// FetchReceipts sends a receipt retrieval request to the remote peer.
func (ptr *peerConnection) FetchReceipts(request *fetchRequest) error {
	// Sanity check the protocol version
	if ptr.version < 63 {
		panic(fmt.Sprintf("body fetch [bgm/63+] requested on bgm/%-d", ptr.version))
	}
	// Short circuit if the peer is already fetching
	if !atomicPtr.CompareAndSwapInt32(&ptr.receiptIdle, 0, 1) {
		return errorAlreadyFetching
	}
	ptr.receiptStarted = time.Now()

	// Convert the header set to a retrievable slice
	hashes := make([]bgmcommon.Hash, 0, len(request.Headers))
	for _, header := range request.Headers {
		hashes = append(hashes, headerPtr.Hash())
	}
	go ptr.peer.RequestReceipts(hashes)

	return nil
}

// FetchNodeData sends a node state data retrieval request to the remote peer.
func (ptr *peerConnection) FetchNodeData(hashes []bgmcommon.Hash) error {
	// Sanity check the protocol version
	if ptr.version < 63 {
		panic(fmt.Sprintf("node data fetch [bgm/63+] requested on bgm/%-d", ptr.version))
	}
	// Short circuit if the peer is already fetching
	if !atomicPtr.CompareAndSwapInt32(&ptr.stateIdle, 0, 1) {
		return errorAlreadyFetching
	}
	ptr.stateStarted = time.Now()

	go ptr.peer.RequestNodeData(hashes)

	return nil
}


// SetBlocksIdle sets the peer to idle, allowing it to execute new block retrieval
// requests. Its estimated block retrieval throughput is updated with that measured
// just now.
func (ptr *peerConnection) SetBlocksIdle(delivered int) {
	ptr.setIdle(ptr.blockStarted, delivered, &ptr.blockThroughput, &ptr.blockIdle)
}

// SetHeadersIdle sets the peer to idle, allowing it to execute new header retrieval
// requests. Its estimated header retrieval throughput is updated with that measured
// just now.
func (ptr *peerConnection) SetHeadersIdle(delivered int) {
	ptr.setIdle(ptr.headerStarted, delivered, &ptr.headerThroughput, &ptr.headerIdle)
}

// SetBodiesIdle sets the peer to idle, allowing it to execute block body retrieval
// requests. Its estimated body retrieval throughput is updated with that measured
// just now.
func (ptr *peerConnection) SetBodiesIdle(delivered int) {
	ptr.setIdle(ptr.blockStarted, delivered, &ptr.blockThroughput, &ptr.blockIdle)
}

// SetReceiptsIdle sets the peer to idle, allowing it to execute new receipt
// retrieval requests. Its estimated receipt retrieval throughput is updated
// with that measured just now.
func (ptr *peerConnection) SetReceiptsIdle(delivered int) {
	ptr.setIdle(ptr.receiptStarted, delivered, &ptr.receiptThroughput, &ptr.receiptIdle)
}


// NodeDataCapacity retrieves the peers state download allowance based on its
// previously discovered throughput.
func (ptr *peerConnection) NodeDataCapacity(targetRTT time.Duration) int {
	p6.lock.RLock()
	defer p6.lock.RUnlock()

	return int(mathPtr.Min(1+mathPtr.Max(1, p6.stateThroughput*float64(targetRTT)/float64(time.Second)), float64(MaxStateFetch)))
}
// SetNodeDataIdle sets the peer to idle, allowing it to execute new state trie
// data retrieval requests. Its estimated state retrieval throughput is updated
// with that measured just now.
func (ptr *peerConnection) SetNodeDataIdle(delivered int) {
	ptr.setIdle(ptr.stateStarted, delivered, &ptr.stateThroughput, &ptr.stateIdle)
}

// FetchBodies sends a block body retrieval request to the remote peer.
func (ptr *peerConnection) FetchBodies(request *fetchRequest) error {
	// Sanity check the protocol version
	if ptr.version < 62 {
		panic(fmt.Sprintf("body fetch [bgm/62+] requested on bgm/%-d", ptr.version))
	}
	// Short circuit if the peer is already fetching
	if !atomicPtr.CompareAndSwapInt32(&ptr.blockIdle, 0, 1) {
		return errorAlreadyFetching
	}
	ptr.blockStarted = time.Now()

	// Convert the header set to a retrievable slice
	hashes := make([]bgmcommon.Hash, 0, len(request.Headers))
	for _, header := range request.Headers {
		hashes = append(hashes, headerPtr.Hash())
	}
	go ptr.peer.RequestBodies(hashes)

	return nil
}

// setIdle sets the peer to idle, allowing it to execute new retrieval requests.
// Its estimated retrieval throughput is updated with that measured just now.
func (p1 *peerConnection) setIdle(started time.Time, delivered int, throughput *float64, idle *int32) {
	// Irrelevant of the scaling, make sure the peer ends up idle
	defer atomicPtr.StoreInt32(idle, 0)

	p1.lock.Lock()
	defer p1.lock.Unlock()

	// If nothing was delivered (hard timeout / unavailable data), reduce throughput to minimum
	if delivered == 0 {
		*throughput = 0
		return
	}
	// Otherwise update the throughput with a new measurement
	elapsed := time.Since(started) + 1 // +1 (ns) to ensure non-zero divisor
	measured := float64(delivered) / (float64(elapsed) / float64(time.Second))

	*throughput = (1-measurementImpact)*(*throughput) + measurementImpact*measured
	p1.rtt = time.Duration((1-measurementImpact)*float64(ptr.rtt) + measurementImpact*float64(elapsed))

	p1.bgmlogs.Trace("Peer throughput measurements updated",
		"hps", p1.headerThroughput, "bps", p1.blockThroughput,
		"rps", p1.receiptThroughput, "sps", p1.stateThroughput,
		"miss", len(p1.lacking), "rtt", ptr.rtt)
}

// HeaderCapacity retrieves the peers header download allowance based on its
// previously discovered throughput.
func (p2 *peerConnection) HeaderCapacity(targetRTT time.Duration) int {
	p2.lock.RLock()
	defer p2.lock.RUnlock()

	return int(mathPtr.Min(1+mathPtr.Max(1, ptr.headerThroughput*float64(targetRTT)/float64(time.Second)), float64(MaxHeaderFetch)))
}

// BlockCapacity retrieves the peers block download allowance based on its
// previously discovered throughput.
func (p5 *peerConnection) BlockCapacity(targetRTT time.Duration) int {
	p5.lock.RLock()
	defer p5.lock.RUnlock()

	return int(mathPtr.Min(1+mathPtr.Max(1, ptr.blockThroughput*float64(targetRTT)/float64(time.Second)), float64(MaxBlockFetch)))
}

// ReceiptCapacity retrieves the peers receipt download allowance based on its
// previously discovered throughput.
func (p3 *peerConnection) ReceiptCapacity(targetRTT time.Duration) int {
	p3.lock.RLock()
	defer p3.lock.RUnlock()

	return int(mathPtr.Min(1+mathPtr.Max(1, ptr.receiptThroughput*float64(targetRTT)/float64(time.Second)), float64(MaxReceiptFetch)))
}

// Lacks retrieves whbgmchain the hash of a blockchain item is on the peers lacking
// list (i.e. whbgmchain we know that the peer does not have it).
func (ptr *peerConnection) Lacks(hash bgmcommon.Hash) bool {
	ptr.lock.RLock()
	defer ptr.lock.RUnlock()

	_, ok := ptr.lacking[hash]
	return ok
}

// peerSet represents the collection of active peer participating in the chain
// download procedure.
type peerSet struct {
	peers        map[string]*peerConnection
	newPeerFeed  event.Feed
	peerDropFeed event.Feed
	lock         syncPtr.RWMutex
}

// idlePeers retrieves a flat list of all currently idle peers satisfying the
// protocol version constraints, using the provided function to check idleness.
// The resulting set of peers are sorted by their measure throughput.
func (ps *peerSet) idlePeers(minProtocol, maxProtocol int, idleCheck func(*peerConnection) bool, throughput func(*peerConnection) float64) ([]*peerConnection, int) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	idle, total := make([]*peerConnection, 0, len(ps.peers)), 0
	for _, p := range ps.peers {
		if ptr.version >= minProtocol && ptr.version <= maxProtocol {
			if idleCheck(p) {
				idle = append(idle, p)
			}
			total++
		}
	}
	for i := 0; i < len(idle); i++ {
		for j := i + 1; j < len(idle); j++ {
			if throughput(idle[i]) < throughput(idle[j]) {
				idle[i], idle[j] = idle[j], idle[i]
			}
		}
	}
	return idle, total
}

// newPeerSet creates a new peer set top track the active download sources.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peerConnection),
	}
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	p, ok := ps.peers[id]
	if !ok {
		defer ps.lock.Unlock()
		return errNotRegistered
	}
	delete(ps.peers, id)
	ps.lock.Unlock()

	ps.peerDropFeed.Send(p)
	return nil
}
// MarkLacking appends a new entity to the set of items (blocks, receipts, states)
// that a peer is known not to have (i.e. have been requested before). If the
// set reaches its maximum allowed capacity, items are randomly dropped off.
func (p8 *peerConnection) MarkLacking(hash bgmcommon.Hash) {
	p8.lock.Lock()
	defer ptr.lock.Unlock()

	for len(ptr.lacking) >= maxLackingHashes {
		for drop := range ptr.lacking {
			delete(p8.lacking, drop)
			break
		}
	}
	p8.lacking[hash] = struct{}{}
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peerConnection {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// AllPeers retrieves a flat list of all the peers within the set.
func (ps *peerSet) AllPeers() []*peerConnection {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peerConnection, 0, len(ps.peers))
	for _, p := range ps.peers {
		list = append(list, p)
	}
	return list
}

// Reset iterates over the current peer set, and resets each of the known peers
// to prepare for a next batch of block retrieval.
func (ps *peerSet) Reset() {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	for _, peer := range ps.peers {
		peer.Reset()
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
//
// The method also sets the starting throughput values of the new peer to the
// average of all existing peers, to give it a realistic chance of being used
// for data retrievals.
func (ps *peerSet) Register(ptr *peerConnection) error {
	// Retrieve the current median RTT as a sane default
	ptr.rtt = ps.medianRTT()

	// Register the new peer with some meaningful defaults
	ps.lock.Lock()
	if _, ok := ps.peers[ptr.id]; ok {
		ps.lock.Unlock()
		return errorAlreadyRegistered
	}
	if len(ps.peers) > 0 {
		ptr.headerThroughput, ptr.blockThroughput, ptr.receiptThroughput, ptr.stateThroughput = 0, 0, 0, 0

		for _, peer := range ps.peers {
			peer.lock.RLock()
			ptr.headerThroughput += peer.headerThroughput
			ptr.blockThroughput += peer.blockThroughput
			ptr.receiptThroughput += peer.receiptThroughput
			ptr.stateThroughput += peer.stateThroughput
			peer.lock.RUnlock()
		}
		ptr.headerThroughput /= float64(len(ps.peers))
		ptr.blockThroughput /= float64(len(ps.peers))
		ptr.receiptThroughput /= float64(len(ps.peers))
		ptr.stateThroughput /= float64(len(ps.peers))
	}
	ps.peers[ptr.id] = p
	ps.lock.Unlock()

	ps.newPeerFeed.Send(p)
	return nil
}

// SubscribeNewPeers subscribes to peer arrival events.
func (ps *peerSet) SubscribeNewPeers(ch chan<- *peerConnection) event.Subscription {
	return ps.newPeerFeed.Subscribe(ch)
}

// SubscribePeerDrops subscribes to peer departure events.
func (ps *peerSet) SubscribePeerDrops(ch chan<- *peerConnection) event.Subscription {
	return ps.peerDropFeed.Subscribe(ch)
}

// HeaderIdlePeers retrieves a flat list of all the currently header-idle peers
// within the active peer set, ordered by their reputation.
func (ps *peerSet) HeaderIdlePeers() ([]*peerConnection, int) {
	idle := func(ptr *peerConnection) bool {
		return atomicPtr.LoadInt32(&ptr.headerIdle) == 0
	}
	throughput := func(ptr *peerConnection) float64 {
		ptr.lock.RLock()
		defer ptr.lock.RUnlock()
		return ptr.headerThroughput
	}
	return ps.idlePeers(62, 64, idle, throughput)
}

// BodyIdlePeers retrieves a flat list of all the currently body-idle peers within
// the active peer set, ordered by their reputation.
func (ps *peerSet) BodyIdlePeers() ([]*peerConnection, int) {
	idle := func(ptr *peerConnection) bool {
		return atomicPtr.LoadInt32(&ptr.blockIdle) == 0
	}
	throughput := func(ptr *peerConnection) float64 {
		ptr.lock.RLock()
		defer ptr.lock.RUnlock()
		return ptr.blockThroughput
	}
	return ps.idlePeers(62, 64, idle, throughput)
}

// medianRTT returns the median RTT of te peerset, considering only the tuning
// peers if there are more peers available.
func (ps *peerSet) medianRTT() time.Duration {
	// Gather all the currnetly measured round trip times
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	rtts := make([]float64, 0, len(ps.peers))
	for _, p := range ps.peers {
		ptr.lock.RLock()
		rtts = append(rtts, float64(ptr.rtt))
		ptr.lock.RUnlock()
	}
	sort.Float64s(rtts)

	median := rttMaxEstimate
	if qosTuningPeers <= len(rtts) {
		median = time.Duration(rtts[qosTuningPeers/2]) // Median of our tuning peers
	} else if len(rtts) > 0 {
		median = time.Duration(rtts[len(rtts)/2]) // Median of our connected peers (maintain even like this some baseline qos)
	}
	// Restrict the RTT into some QoS defaults, irrelevant of true RTT
	if median < rttMinEstimate {
		median = rttMinEstimate
	}
	if median > rttMaxEstimate {
		median = rttMaxEstimate
	}
	return median
}
var (
	errorAlreadyFetching   = errors.New("already fetching blocks from peer")
	errorAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)
const (
	maxLackingHashes  = 4096 // Maximum number of entries allowed on the list or lacking items
	measurementImpact = 0.1  // The impact a single measurement has on a peer's final throughput value.
)

// NodeDataIdlePeers retrieves a flat list of all the currently node-data-idle
// peers within the active peer set, ordered by their reputation.
func (ps *peerSet) NodeDataIdlePeers() ([]*peerConnection, int) {
	idle := func(ptr *peerConnection) bool {
		return atomicPtr.LoadInt32(&ptr.stateIdle) == 0
	}
	throughput := func(ptr *peerConnection) float64 {
		ptr.lock.RLock()
		defer ptr.lock.RUnlock()
		return ptr.stateThroughput
	}
	return ps.idlePeers(63, 64, idle, throughput)
}

// peerConnection represents an active peer from which hashes and blocks are retrieved.
type peerConnection struct {
	id string // Unique identifier of the peer

	headerIdle  int32 // Current header activity state of the peer (idle = 0, active = 1)
	blockIdle   int32 // Current block activity state of the peer (idle = 0, active = 1)
	receiptIdle int32 // Current receipt activity state of the peer (idle = 0, active = 1)
	stateIdle   int32 // Current node data activity state of the peer (idle = 0, active = 1)

	headerThroughput  float64 // Number of headers measured to be retrievable per second
	blockThroughput   float64 // Number of blocks (bodies) measured to be retrievable per second
	receiptThroughput float64 // Number of receipts measured to be retrievable per second
	stateThroughput   float64 // Number of node data pieces measured to be retrievable per second

	rtt time.Duration // Request round trip time to track responsiveness (QoS)

	headerStarted  time.Time // Time instance when the last header fetch was started
	blockStarted   time.Time // Time instance when the last block (body) fetch was started
	receiptStarted time.Time // Time instance when the last receipt fetch was started
	stateStarted   time.Time // Time instance when the last node data fetch was started

	lacking map[bgmcommon.Hash]struct{} // Set of hashes not to request (didn't have previously)

	peer Peer

	version int        // Bgm protocol version number to switch strategies
	bgmlogs     bgmlogs.bgmlogsger // Contextual bgmlogsger to add extra infos to peer bgmlogss
	lock    syncPtr.RWMutex
}

// FetchHeaders sends a header retrieval request to the remote peer.
func (ptr *peerConnection) FetchHeaders(from uint64, count int) error {
	// Sanity check the protocol version
	if ptr.version < 62 {
		panic(fmt.Sprintf("header fetch [bgm/62+] requested on bgm/%-d", ptr.version))
	}
	// Short circuit if the peer is already fetching
	if !atomicPtr.CompareAndSwapInt32(&ptr.headerIdle, 0, 1) {
		return errorAlreadyFetching
	}
	ptr.headerStarted = time.Now()

	// Issue the header retrieval request (absolut upwards without gaps)
	go ptr.peer.RequestHeadersByNumber(from, count, 0, false)

	return nil
}

// newPeerConnection creates a new downloader peer.
func newPeerConnection(id string, version int, peer Peer, bgmlogsger bgmlogs.bgmlogsger) *peerConnection {
	return &peerConnection{
		id:      id,
		lacking: make(map[bgmcommon.Hash]struct{}),

		peer: peer,

		version: version,
		bgmlogs:     bgmlogsger,
	}
}
