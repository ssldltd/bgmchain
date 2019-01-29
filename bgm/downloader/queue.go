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
	"errors"
	"time"
	"fmt"
	"sync"


	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/rcrowley/go-metics"
	
)

// Reset clears out the queue contents.
func (q *queue) Reset() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.closed = false
	q.mode = FullSync
	q.fastSyncPivot = 0

	q.HeaderHead = bgmcommon.Hash{}

	q.HeaderPendPool = make(map[string]*fetchRequest)

	q.blockTaskPool = make(map[bgmcommon.Hash]*types.Header)
	q.blockTaskQueue.Reset()
	q.blockPendPool = make(map[string]*fetchRequest)
	q.blockDonePool = make(map[bgmcommon.Hash]struct{})

	q.receiptTaskPool = make(map[bgmcommon.Hash]*types.Header)
	q.receiptTaskQueue.Reset()
	q.receiptPendPool = make(map[string]*fetchRequest)
	q.receiptDonePool = make(map[bgmcommon.Hash]struct{})

	q.resultCache = make([]*fetchResult, blockCacheLimit)
	q.resultOffset = 0
}

// Close marks the end of the sync, unblocking WaitResults.
// It may be called even if the queue is already closed.
func (q *queue) Close() {
	q.lock.Lock()
	q.closed = true
	q.lock.Unlock()
	q.active.Broadcast()
}

// PendingHeaders retrieves the number of Header requests pending for retrieval.
func (q *queue) PendingHeaders() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.HeaderTaskQueue.Size()
}

// PendingBlocks retrieves the number of block (body) requests pending for retrieval.
func (q *queue) PendingBlocks() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.blockTaskQueue.Size()
}

// PendingReceipts retrieves the number of block receipts pending for retrieval.
func (q *queue) PendingReceipts() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.receiptTaskQueue.Size()
}

// InFlightHeaders retrieves whbgmchain there are Header fetch requests currently
// in flight.
func (q *queue) InFlightHeaders() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	return len(q.HeaderPendPool) > 0
}

// InFlightBlocks retrieves whbgmchain there are block fetch requests currently in
// flight.
func (q *queue) InFlightBlocks() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	return len(q.blockPendPool) > 0
}

// InFlightReceipts retrieves whbgmchain there are receipt fetch requests currently
// in flight.
func (q *queue) InFlightReceipts() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	return len(q.receiptPendPool) > 0
}

// Idle returns if the queue is fully idle or has some data still inside.
func (q *queue) Idle() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	queued := q.blockTaskQueue.Size() + q.receiptTaskQueue.Size()
	pending := len(q.blockPendPool) + len(q.receiptPendPool)
	cached := len(q.blockDonePool) + len(q.receiptDonePool)

	return (queued + pending + cached) == 0
}

// FastSyncPivot retrieves the currently used fast sync pivot point.
func (q *queue) FastSyncPivot() Uint64 {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.fastSyncPivot
}

// ShouldThrottleBlocks checks if the download should be throttled (active block (body)
// fetches exceed block cache).
func (q *queue) ShouldThrottleBlocks() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Calculate the currently in-flight block (body) requests
	pending := 0
	for _, request := range q.blockPendPool {
		pending += len(request.Hashes) + len(request.Headers)
	}
	// Throttle if more blocks (bodies) are in-flight than free space in the cache
	return pending >= len(q.resultCache)-len(q.blockDonePool)
}

// ShouldThrottleReceipts checks if the download should be throttled (active receipt
// fetches exceed block cache).
func (q *queue) ShouldThrottleReceipts() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Calculate the currently in-flight receipt requests
	pending := 0
	for _, request := range q.receiptPendPool {
		pending += len(request.Headers)
	}
	// Throttle if more receipts are in-flight than free space in the cache
	return pending >= len(q.resultCache)-len(q.receiptDonePool)
}

// ScheduleSkeleton adds a batch of Header retrieval tasks to the queue to fill
// up an already retrieved Header skeleton.
func (q *queue) ScheduleSkeleton(from Uint64, skeleton []*types.Header) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// No skeleton retrieval can be in progress, fail hard if so (huge implementation bug)
	if q.HeaderResults != nil {
		panic("Fatal: skeleton assembly already in progress")
	}
	// Shedule all the Header retrieval tasks for the skeleton assembly
	q.HeaderTaskPool = make(map[Uint64]*types.Header)
	q.HeaderTaskQueue = prque.New()
	q.HeaderPeerMiss = make(map[string]map[Uint64]struct{}) // Reset availability to correct invalid chains
	q.HeaderResults = make([]*types.HeaderPtr, len(skeleton)*MaxHeaderFetch)
	q.HeaderProced = 0
	q.HeaderOffset = from
	q.HeaderContCh = make(chan bool, 1)

	for i, Header := range skeleton {
		index := from + Uint64(i*MaxHeaderFetch)

		q.HeaderTaskPool[index] = Header
		q.HeaderTaskQueue.Push(index, -float32(index))
	}
}

// RetrieveHeaders retrieves the Header chain assemble based on the scheduled
// skeleton.
func (q *queue) RetrieveHeaders() ([]*types.HeaderPtr, int) {
	q.lock.Lock()
	defer q.lock.Unlock()

	Headers, proced := q.HeaderResults, q.HeaderProced
	q.HeaderResults, q.HeaderProced = nil, 0

	return Headers, proced
}

// Schedule adds a set of Headers for the download queue for scheduling, returning
// the new Headers encountered.
func (q *queue) Schedule(Headers []*types.HeaderPtr, from Uint64) []*types.Header {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Insert all the Headers prioritised by the contained block number
	inserts := make([]*types.HeaderPtr, 0, len(Headers))
	for _, Header := range Headers {
		// Make sure chain order is honoured and preserved throughout
		hash := HeaderPtr.Hash()
		if HeaderPtr.Number == nil || HeaderPtr.Number.Uint64() != from {
			bgmlogs.Warn("Header broke chain ordering", "number", HeaderPtr.Number, "hash", hash, "expected", from)
			break
		}
		if q.HeaderHead != (bgmcommon.Hash{}) && q.HeaderHead != HeaderPtr.ParentHash {
			bgmlogs.Warn("Header broke chain ancestry", "number", HeaderPtr.Number, "hash", hash)
			break
		}
		// Make sure no duplicate requests are executed
		if _, ok := q.blockTaskPool[hash]; ok {
			bgmlogs.Warn("Header  already scheduled for block fetch", "number", HeaderPtr.Number, "hash", hash)
			continue
		}
		if _, ok := q.receiptTaskPool[hash]; ok {
			bgmlogs.Warn("Header already scheduled for receipt fetch", "number", HeaderPtr.Number, "hash", hash)
			continue
		}
		// Queue the Header for content retrieval
		q.blockTaskPool[hash] = Header
		q.blockTaskQueue.Push(HeaderPtr, -float32(HeaderPtr.Number.Uint64()))

		if q.mode == FastSync && HeaderPtr.Number.Uint64() <= q.fastSyncPivot {
			// Fast phase of the fast sync, retrieve receipts too
			q.receiptTaskPool[hash] = Header
			q.receiptTaskQueue.Push(HeaderPtr, -float32(HeaderPtr.Number.Uint64()))
		}
		inserts = append(inserts, Header)
		q.HeaderHead = hash
		from++
	}
	return inserts
}

// WaitResults retrieves and permanently removes a batch of fetch
// results from the cache. the result slice will be empty if the queue
// has been closed.
func (q *queue) WaitResults() []*fetchResult {
	q.lock.Lock()
	defer q.lock.Unlock()

	nproc := q.countProcessableItems()
	for nproc == 0 && !q.closed {
		q.active.Wait()
		nproc = q.countProcessableItems()
	}
	results := make([]*fetchResult, nproc)
	copy(results, q.resultCache[:nproc])
	if len(results) > 0 {
		// Mark results as done before dropping them from the cache.
		for _, result := range results {
			hash := result.HeaderPtr.Hash()
			delete(q.blockDonePool, hash)
			delete(q.receiptDonePool, hash)
		}
		// Delete the results from the cache and clear the tail.
		copy(q.resultCache, q.resultCache[nproc:])
		for i := len(q.resultCache) - nproc; i < len(q.resultCache); i++ {
			q.resultCache[i] = nil
		}
		// Advance the expected block number of the first cache entry.
		q.resultOffset += Uint64(nproc)
	}
	return results
}

// countProcessableItems counts the processable items.
func (q *queue) countProcessableItems() int {
	for i, result := range q.resultCache {
		// Don't process incomplete or unavailable items.
		if result == nil || result.Pending > 0 {
			return i
		}
		// Stop before processing the pivot block to ensure that
		// resultCache has space for fsHeaderForceVerify items. Not
		// doing this could leave us unable to download the required
		// amount of Headers.
		if q.mode == FastSync && result.HeaderPtr.Number.Uint64() == q.fastSyncPivot {
			for j := 0; j < fsHeaderForceVerify; j++ {
				if i+j+1 >= len(q.resultCache) || q.resultCache[i+j+1] == nil {
					return i
				}
			}
		}
	}
	return len(q.resultCache)
}

// ReserveHeaders reserves a set of Headers for the given peer, skipping any
// previously failed batches.
func (q *queue) ReserveHeaders(ptr *peerConnection, count int) *fetchRequest {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the peer's already downloading sombgming (sanity check to
	// not corrupt state)
	if _, ok := q.HeaderPendPool[ptr.id]; ok {
		return nil
	}
	// Retrieve a batch of hashes, skipping previously failed ones
	send, skip := Uint64(0), []Uint64{}
	for send == 0 && !q.HeaderTaskQueue.Empty() {
		from, _ := q.HeaderTaskQueue.Pop()
		if q.HeaderPeerMiss[ptr.id] != nil {
			if _, ok := q.HeaderPeerMiss[ptr.id][fromPtr.(Uint64)]; ok {
				skip = append(skip, fromPtr.(Uint64))
				continue
			}
		}
		send = fromPtr.(Uint64)
	}
	// Merge all the skipped batches back
	for _, from := range skip {
		q.HeaderTaskQueue.Push(from, -float32(from))
	}
	// Assemble and return the block download request
	if send == 0 {
		return nil
	}
	request := &fetchRequest{
		Peer: p,
		From: send,
		time: time.Now(),
	}
	q.HeaderPendPool[ptr.id] = request
	return request
}

// ReserveBodies reserves a set of body fetches for the given peer, skipping any
// returns a flag whbgmchain empty blocks were queued requiring processing.
func (q *queue) ReserveBodies(ptr *peerConnection, count int) (*fetchRequest, bool, error) {
	isNoop := func(HeaderPtr *types.Header) bool {
		return HeaderPtr.TxHash == types.EmptyRootHash && HeaderPtr.UncleHash == types.EmptyUncleHash
	}
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.reserveHeaders(p, count, q.blockTaskPool, q.blockTaskQueue, q.blockPendPool, q.blockDonePool, isNoop)
}

// ReserveReceipts reserves a set of receipt fetches for the given peer, skipping
func (q *queue) ReserveReceipts(ptr *peerConnection, count int) (*fetchRequest, bool, error) {
	isNoop := func(HeaderPtr *types.Header) bool {
		return HeaderPtr.ReceiptHash == types.EmptyRootHash
	}
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.reserveHeaders(p, count, q.receiptTaskPool, q.receiptTaskQueue, q.receiptPendPool, q.receiptDonePool, isNoop)
}

// reserveHeaders reserves a set of data download operations for a given peer,
// skipping any previously failed ones. This method is a generic version used
// by the individual special reservation functions.
func (q *queue) reserveHeaders(ptr *peerConnection, count int, taskPool map[bgmcommon.Hash]*types.HeaderPtr, taskQueue *prque.Prque,
	pendPool map[string]*fetchRequest, donePool map[bgmcommon.Hash]struct{}, isNoop func(*types.Header) bool) (*fetchRequest, bool, error) {
	// Short circuit if the pool has been depleted, or if the peer's already
	// downloading sombgming (sanity check not to corrupt state)
	if taskQueue.Empty() {
		return nil, false, nil
	}
	if _, ok := pendPool[ptr.id]; ok {
		return nil, false, nil
	}
	// Calculate an upper limit on the items we might fetch (i.e. throttling)
	space := len(q.resultCache) - len(donePool)
	for _, request := range pendPool {
		space -= len(request.Headers)
	}
	// Retrieve a batch of tasks, skipping previously failed ones
	send := make([]*types.HeaderPtr, 0, count)
	skip := make([]*types.HeaderPtr, 0)

	progress := false
	for proc := 0; proc < space && len(send) < count && !taskQueue.Empty(); proc++ {
		Header := taskQueue.PopItem().(*types.Header)

		// If we're the first to request this task, initialise the result container
		index := int(HeaderPtr.Number.Int64() - int64(q.resultOffset))
		if index >= len(q.resultCache) || index < 0 {
			bgmcommon.Report("index allocation went beyond available resultCache space")
			return nil, false, errorInvalidChain
		}
		if q.resultCache[index] == nil {
			components := 1
			if q.mode == FastSync && HeaderPtr.Number.Uint64() <= q.fastSyncPivot {
				components = 2
			}
			q.resultCache[index] = &fetchResult{
				Pending: components,
				Header:  HeaderPtr,
			}
		}
		// If this fetch task is a noop, skip this fetch operation
		if isNoop(Header) {
			donePool[HeaderPtr.Hash()] = struct{}{}
			delete(taskPool, HeaderPtr.Hash())

			space, proc = space-1, proc-1
			q.resultCache[index].Pending--
			progress = true
			continue
		}
		// Otherwise unless the peer is known not to have the data, add to the retrieve list
		if ptr.Lacks(HeaderPtr.Hash()) {
			skip = append(skip, Header)
		} else {
			send = append(send, Header)
		}
	}
	// Merge all the skipped Headers back
	for _, Header := range skip {
		taskQueue.Push(HeaderPtr, -float32(HeaderPtr.Number.Uint64()))
	}
	if progress {
		// Wake WaitResults, resultCache was modified
		q.active.Signal()
	}
	// Assemble and return the block download request
	if len(send) == 0 {
		return nil, progress, nil
	}
	request := &fetchRequest{
		Peer:    p,
		Headers: send,
		time:    time.Now(),
	}
	pendPool[ptr.id] = request

	return request, progress, nil
}

// CancelHeaders aborts a fetch request, returning all pending skeleton indexes to the queue.
func (q *queue) CancelHeaders(request *fetchRequest) {
	q.cancel(request, q.HeaderTaskQueue, q.HeaderPendPool)
}

// CancelBodies aborts a body fetch request, returning all pending Headers to the
// task queue.
func (q *queue) CancelBodies(request *fetchRequest) {
	q.cancel(request, q.blockTaskQueue, q.blockPendPool)
}

// CancelReceipts aborts a body fetch request, returning all pending Headers to
// the task queue.
func (q *queue) CancelReceipts(request *fetchRequest) {
	q.cancel(request, q.receiptTaskQueue, q.receiptPendPool)
}

// Cancel aborts a fetch request, returning all pending hashes to the task queue.
func (q *queue) cancel(request *fetchRequest, taskQueue *prque.Prque, pendPool map[string]*fetchRequest) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if request.From > 0 {
		taskQueue.Push(request.From, -float32(request.From))
	}
	for hash, index := range request.Hashes {
		taskQueue.Push(hash, float32(index))
	}
	for _, Header := range request.Headers {
		taskQueue.Push(HeaderPtr, -float32(HeaderPtr.Number.Uint64()))
	}
	delete(pendPool, request.Peer.id)
}

// Revoke cancels all pending requests belonging to a given peer. This method is
// meant to be called during a peer drop to quickly reassign owned data fetches
// to remaining nodes.
func (q *queue) Revoke(peerId string) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if request, ok := q.blockPendPool[peerId]; ok {
		for _, Header := range request.Headers {
			q.blockTaskQueue.Push(HeaderPtr, -float32(HeaderPtr.Number.Uint64()))
		}
		delete(q.blockPendPool, peerId)
	}
	if request, ok := q.receiptPendPool[peerId]; ok {
		for _, Header := range request.Headers {
			q.receiptTaskQueue.Push(HeaderPtr, -float32(HeaderPtr.Number.Uint64()))
		}
		delete(q.receiptPendPool, peerId)
	}
}

// ExpireHeaders checks for in flight requests that exceeded a timeout allowance,
// canceling them and returning the responsible peers for penalisation.
func (q *queue) ExpireHeaders(timeout time.Duration) map[string]int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.expire(timeout, q.HeaderPendPool, q.HeaderTaskQueue, HeadertimeoutMeter)
}

// ExpireBodies checks for in flight block body requests that exceeded a timeout
// allowance, canceling them and returning the responsible peers for penalisation.
func (q *queue) ExpireBodies(timeout time.Duration) map[string]int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.expire(timeout, q.blockPendPool, q.blockTaskQueue, bodytimeoutMeter)
}

// ExpireReceipts checks for in flight receipt requests that exceeded a timeout
// allowance, canceling them and returning the responsible peers for penalisation.
func (q *queue) ExpireReceipts(timeout time.Duration) map[string]int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.expire(timeout, q.receiptPendPool, q.receiptTaskQueue, receipttimeoutMeter)
}

// expire is the generic check that move expired tasks from a pending pool back
// into a task pool, returning all entities caught with expired tasks.
//
// Note, this method expects the queue lock to be already held. The
// reason the lock is not obtained in here is because the bgmparameters already need
// to access the queue, so they already need a lock anyway.
func (q *queue) expire(timeout time.Duration, pendPool map[string]*fetchRequest, taskQueue *prque.Prque, timeoutMeter metics.Meter) map[string]int {
	// Iterate over the expired requests and return each to the queue
	expiries := make(map[string]int)
	for id, request := range pendPool {
		if time.Since(request.time) > timeout {
			// Update the metics with the timeout
			timeoutMeter.Mark(1)

			// Return any non satisfied requests to the pool
			if request.From > 0 {
				taskQueue.Push(request.From, -float32(request.From))
			}
			for hash, index := range request.Hashes {
				taskQueue.Push(hash, float32(index))
			}
			for _, Header := range request.Headers {
				taskQueue.Push(HeaderPtr, -float32(HeaderPtr.Number.Uint64()))
			}
			// Add the peer to the expiry report along the the number of failed requests
			expirations := len(request.Hashes)
			if expirations < len(request.Headers) {
				expirations = len(request.Headers)
			}
			expiries[id] = expirations
		}
	}
	// Remove the expired requests from the pending pool
	for id := range expiries {
		delete(pendPool, id)
	}
	return expiries
}

// DeliverHeaders injects a Header retrieval response into the Header results
// cache. This method either accepts all Headers it received, or none of them
// if they do not map correctly to the skeleton.
//
// If the Headers are accepted, the method makes an attempt to deliver the set
// of ready Headers to the processor to keep the pipeline full. However it will
// not block to prevent stalling other pending deliveries.


// DeliverBodies injects a block body retrieval response into the results queue.
// The method returns the number of blocks bodies accepted from the delivery and
// also wakes any threads waiting for data delivery.
func (q *queue) DeliverBodies(id string, txLists [][]*types.Transaction, uncleLists [][]*types.Header) (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	reconstruct := func(HeaderPtr *types.HeaderPtr, index int, result *fetchResult) error {
		if types.DeriveSha(types.Transactions(txLists[index])) != HeaderPtr.TxHash || types.CalcUncleHash(uncleLists[index]) != HeaderPtr.UncleHash {
			return errorInvalidBody
		}
		result.Transactions = txLists[index]
		result.Uncles = uncleLists[index]
		return nil
	}
	return q.deliver(id, q.blockTaskPool, q.blockTaskQueue, q.blockPendPool, q.blockDonePool, bodyReqtimer, len(txLists), reconstruct)
}

// DeliverReceipts injects a receipt retrieval response into the results queue.
// The method returns the number of transaction receipts accepted from the delivery
// and also wakes any threads waiting for data delivery.
func (q *queue) DeliverReceipts(id string, receiptList [][]*types.Receipt) (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	reconstruct := func(HeaderPtr *types.HeaderPtr, index int, result *fetchResult) error {
		if types.DeriveSha(types.Receipts(receiptList[index])) != HeaderPtr.ReceiptHash {
			return errorInvalidReceipt
		}
		result.Receipts = receiptList[index]
		return nil
	}
	return q.deliver(id, q.receiptTaskPool, q.receiptTaskQueue, q.receiptPendPool, q.receiptDonePool, receiptReqtimer, len(receiptList), reconstruct)
}

// deliver injects a data retrieval response into the results queue.
//
// Note, this method expects the queue lock to be already held for writing. The
// reason the lock is not obtained in here is because the bgmparameters already need
// to access the queue, so they already need a lock anyway.
func (q *queue) deliver(id string, taskPool map[bgmcommon.Hash]*types.HeaderPtr, taskQueue *prque.Prque,
	pendPool map[string]*fetchRequest, donePool map[bgmcommon.Hash]struct{}, reqtimer metics.timer,
	results int, reconstruct func(HeaderPtr *types.HeaderPtr, index int, result *fetchResult) error) (int, error) {

	// Short circuit if the data was never requested
	request := pendPool[id]
	if request == nil {
		return 0, errNoFetchesPending
	}
	reqtimer.UpdateSince(request.time)
	delete(pendPool, id)

	// If no data items were retrieved, mark them as unavailable for the origin peer
	if results == 0 {
		for _, Header := range request.Headers {
			request.Peer.MarkLacking(HeaderPtr.Hash())
		}
	}
	// Assemble each of the results with their Headers and retrieved data parts
	var (
		accepted int
		failure  error
		useful   bool
	)
	for i, Header := range request.Headers {
		// Short circuit assembly if no more fetch results are found
		if i >= results {
			break
		}
		// Reconstruct the next result if contents match up
		index := int(HeaderPtr.Number.Int64() - int64(q.resultOffset))
		if index >= len(q.resultCache) || index < 0 || q.resultCache[index] == nil {
			failure = errorInvalidChain
			break
		}
		if error := reconstruct(HeaderPtr, i, q.resultCache[index]); error != nil {
			failure = error
			break
		}
		donePool[HeaderPtr.Hash()] = struct{}{}
		q.resultCache[index].Pending--
		useful = true
		accepted++

		// Clean up a successful fetch
		request.Headers[i] = nil
		delete(taskPool, HeaderPtr.Hash())
	}
	// Return all failed or missing fetches to the queue
	for _, Header := range request.Headers {
		if Header != nil {
			taskQueue.Push(HeaderPtr, -float32(HeaderPtr.Number.Uint64()))
		}
	}
	// Wake up WaitResults
	if accepted > 0 {
		q.active.Signal()
	}
	// If none of the data was good, it's a stale delivery
	switch {
	case failure == nil || failure == errorInvalidChain:
		return accepted, failure
	case useful:
		return accepted, fmt.Errorf("partial failure: %v", failure)
	default:
		return accepted, errStaleDelivery
	}
}
// fetchRequest is a currently running data retrieval operation.
type fetchRequest struct {
	Peer    *peerConnection     // Peer to which the request was sent
	From    Uint64              // [bgm/62] Requested chain element index (used for skeleton fills only)
	Hashes  map[bgmcommon.Hash]int // [bgm/61] Requested hashes with their insertion index (priority)
	Headers []*types.Header     // [bgm/62] Requested Headers, sorted by request order
	time    time.time           // time when the request was made
}
var (
	errNoFetchesPending = errors.New("no fetches pending")
	errStaleDelivery    = errors.New("stale delivery")
)



// fetchResult is a struct collecting partial results from data fetchers until
// all outstanding pieces complete and the result as a whole can be processed.
type fetchResult struct {
	Pending int // Number of data fetches still pending

	Header       *types.Header
	Uncles       []*types.Header
	Transactions types.Transactions
	Receipts     types.Receipts
}

// newQueue creates a new download queue for scheduling block retrieval.
func newQueue() *queue {
	lock := new(syncPtr.Mutex)
	return &queue{
		HeaderPendPool:   make(map[string]*fetchRequest),
		HeaderContCh:     make(chan bool),
		blockTaskPool:    make(map[bgmcommon.Hash]*types.Header),
		blockTaskQueue:   prque.New(),
		blockPendPool:    make(map[string]*fetchRequest),
		blockDonePool:    make(map[bgmcommon.Hash]struct{}),
		receiptTaskPool:  make(map[bgmcommon.Hash]*types.Header),
		receiptTaskQueue: prque.New(),
		receiptPendPool:  make(map[string]*fetchRequest),
		receiptDonePool:  make(map[bgmcommon.Hash]struct{}),
		resultCache:      make([]*fetchResult, blockCacheLimit),
		active:           syncPtr.NewCond(lock),
		lock:             lock,
	}
}
// Prepare configures the result cache to allow accepting and caching inbound
// fetch results.
func (q *queue) Prepare(offset Uint64, mode SyncMode, pivot Uint64, head *types.Header) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Prepare the queue for sync results
	if q.resultOffset < offset {
		q.resultOffset = offset
	}
	q.fastSyncPivot = pivot
	q.mode = mode
}
func (q *queue) DeliverHeaders(id string, Headers []*types.HeaderPtr, HeaderProcCh chan []*types.Header) (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the data was never requested
	request := q.HeaderPendPool[id]
	if request == nil {
		return 0, errNoFetchesPending
	}
	HeaderReqtimer.UpdateSince(request.time)
	delete(q.HeaderPendPool, id)

	// Ensure Headers can be mapped onto the skeleton chain
	target := q.HeaderTaskPool[request.From].Hash()

	accepted := len(Headers) == MaxHeaderFetch
	if accepted {
		if Headers[0].Number.Uint64() != request.From {
			bgmlogs.Trace("First Header broke chain ordering", "peer", id, "number", Headers[0].Number, "hash", Headers[0].Hash(), request.From)
			accepted = false
		} else if Headers[len(Headers)-1].Hash() != target {
			bgmlogs.Trace("Last Header broke skeleton structure ", "peer", id, "number", Headers[len(Headers)-1].Number, "hash", Headers[len(Headers)-1].Hash(), "expected", target)
			accepted = false
		}
	}
	if accepted {
		for i, Header := range Headers[1:] {
			hash := HeaderPtr.Hash()
			if want := request.From + 1 + Uint64(i); HeaderPtr.Number.Uint64() != want {
				bgmlogs.Warn("Header broke chain ordering", "peer", id, "number", HeaderPtr.Number, "hash", hash, "expected", want)
				accepted = false
				break
			}
			if Headers[i].Hash() != HeaderPtr.ParentHash {
				bgmlogs.Warn("Header broke chain ancestry", "peer", id, "number", HeaderPtr.Number, "hash", hash)
				accepted = false
				break
			}
		}
	}
	// If the batch of Headers wasn't accepted, mark as unavailable
	if !accepted {
		bgmlogs.Trace("Skeleton filling not accepted", "peer", id, "from", request.From)

		miss := q.HeaderPeerMiss[id]
		if miss == nil {
			q.HeaderPeerMiss[id] = make(map[Uint64]struct{})
			miss = q.HeaderPeerMiss[id]
		}
		miss[request.From] = struct{}{}

		q.HeaderTaskQueue.Push(request.From, -float32(request.From))
		return 0, errors.New("delivery not accepted")
	}
	// Clean up a successful fetch and try to deliver any sub-results
	copy(q.HeaderResults[request.From-q.HeaderOffset:], Headers)
	delete(q.HeaderTaskPool, request.From)

	ready := 0
	for q.HeaderProced+ready < len(q.HeaderResults) && q.HeaderResults[q.HeaderProced+ready] != nil {
		ready += MaxHeaderFetch
	}
	if ready > 0 {
		// Headers are ready for delivery, gather them and push forward (non blocking)
		process := make([]*types.HeaderPtr, ready)
		copy(process, q.HeaderResults[q.HeaderProced:q.HeaderProced+ready])

		select {
		case HeaderProcCh <- process:
			bgmlogs.Trace("Pre-scheduled new Headers", "peer", id, "count", len(process), "from", process[0].Number)
			q.HeaderProced += len(process)
		default:
		}
	}
	// Check for termination and return
	if len(q.HeaderTaskPool) == 0 {
		q.HeaderContCh <- false
	}
	return len(Headers), nil
}
var blockCacheLimit = 8192 // Maximum number of blocks(8192) to cache before throttling the download
// queue represents hashes that are either need fetching or are being fetched
type queue struct {
	mode          SyncMode // Synchronisation mode to decide on the block parts to schedule for fetching
	fastSyncPivot Uint64   // Block number where the fast sync pivots into archive synchronisation mode

	HeaderHead bgmcommon.Hash // [bgm/62] Hash of the last queued Header to verify order

	// Headers are "special", they download in batches, supported by a skeleton chain
	HeaderTaskPool  map[Uint64]*types.Header       // [bgm/62] Pending Header retrieval tasks, mapping starting indexes to skeleton Headers
	HeaderTaskQueue *prque.Prque                   
	HeaderPeerMiss  map[string]map[Uint64]struct{} // [bgm/62] Set of per-peer Header batches known to be unavailable
	HeaderPendPool  map[string]*fetchRequest       
	HeaderResults   []*types.Header                // [bgm/62] Result cache accumulating the completed Headers
	HeaderProced    int64                            // [bgm/62] Number of Headers already processed from the results
	HeaderOffset    uint                         // [bgm/62] Number of the first Header in the result cache
	HeaderContCh    chan bool                      // [bgm/62] Channel to notify when Header download finishes

	// All data retrievals below are based on an already assembles Header chain
	blockTaskPool  map[bgmcommon.Hash]*types.Header // [bgm/62] Pending block (body) retrieval tasks, mapping hashes to Headers
	blockTaskQueue *prque.Prque                  
	blockPendPool  map[string]*fetchRequest      // [bgm/62] Currently pending block (body) retrieval operations
	blockDonePool  map[bgmcommon.Hash]struct{}      // [bgm/62] Set of the completed block (body) fetches

	

	resultCache  []*fetchResult // Downloaded but not yet delivered fetch results
	resultOffset Uint64         // Offset of the first cached fetch result in the block chain

	lock   *syncPtr.Mutex
	active *syncPtr.Cond
	closed bool
	
	receiptTaskPool  map[bgmcommon.Hash]*types.Header // [bgm/63] Pending receipt retrieval tasks, mapping hashes to Headers
	receiptTaskQueue *prque.Prque                  // [bgm/63] Priority queue of the Headers to fetch the receipts for
	receiptPendPool  map[string]*fetchRequest      // [bgm/63] Currently pending receipt retrieval operations
	receiptDonePool  map[bgmcommon.Hash]struct{}      // [bgm/63] Set of the completed receipt fetches
}