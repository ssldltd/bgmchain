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

// Package downloader contains the manual full chain synchronisation.
package downloader

import (
	"math/big"
	"bgmcrypto/rand"
	
	"fmt"
	"math"
	
	"sync"
	"sync/atomic"
	"time"
	"errors"

	bgmchain "github.com/ssldltd/bgmchain"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/bgmlogs"

	"github.com/rcrowley/go-metics"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
)

// New creates a new downloader to fetch hashes and blocks from remote peers.
func (d *Downloader) fetchParts(errCancel error, deliveryCh chan dataPack, deliver func(dataPack) (int, error), wakeCh chan bool,
	expire func() map[string]int, pending func() int, inFlight func() bool, throttle func() bool, reserve func(*peerConnection, int) (*fetchRequest, bool, error),
	fetchHook func([]*types.Header), fetch func(*peerConnection, *fetchRequest) error, cancel func(*fetchRequest), capacity func(*peerConnection) int,
	idle func() ([]*peerConnection, int), setIdle func(*peerConnection, int), kind string) error {

	// Create a ticker to detect expired retrieval tasks
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	update := make(chan struct{}, 1)

	// Prepare the queue and fetch block parts until the block Header fetcher's done
	finished := false
	for {
		select {
		case <-d.cancelCh:
			return errCancel

		case packet := <-deliveryCh:
			// If the peer was previously banned and failed to deliver its pack
			// in a reasonable time frame, ignore its message.
			if peer := d.peers.Peer(packet.PeerId()); peer != nil {
				// Deliver the received chunk of data and check chain validity
				accepted, error := deliver(packet)
				if error == errorInvalidChain {
					return error
				}
				// Unless a peer delivered sombgming completely else than requested (usually
				// caused by a timed out request which came through in the end), set it to
				// idle. If the delivery's stale, the peer should have already been idled.
				if error != errStaleDelivery {
					setIdle(peer, accepted)
				}
				// Issue a bgmlogs to the user to see what's going on
				switch {
				case error == nil && packet.Items() == 0:
					peer.bgmlogs.Trace("Requested data not delivered", "type", kind)
				case error == nil:
					peer.bgmlogs.Trace("Delivered new batch of data", "type", kind, "count", packet.Stats())
				default:
					peer.bgmlogs.Trace("Failed to deliver retrieved data", "type", kind, "error", error)
				}
			}
			// Blocks assembled, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

	 case <-ticker.C:
			// Sanity check update the progress
			select {
			case update <- struct{}{}:
			default:
			}
			
		case cont := <-wakeCh:
			// The Header fetcher sent a continuation flag, check if it's done
			if !cont {
				finished = true
			}
			// Headers arrive, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case <-update:
			// Short circuit if we lost all our peers
			if d.peers.Len() == 0 {
				return errNoPeers
			}
			// Check for fetch request timeouts and demote the responsible peers
			for pid, fails := range expire() {
				if peer := d.peers.Peer(pid); peer != nil {
					// If a lot of retrieval elements expired, we might have overestimated the remote peer or perhaps
					// ourselves. Only reset to minimal throughput but don't drop just yet. If even the minimal times
					// out that sync wise we need to get rid of the peer.
					//
					// The reason the minimum threshold is 2 is because the downloader tries to estimate the bandwidth
					// and latency of a peer separately, which requires pushing the measures capacity a bit and seeing
					// how response times reacts, to it always requests one more than the minimum (i.e. min 2).
					if fails > 2 {
						peer.bgmlogs.Trace("Data delivery timed out", "type", kind)
						setIdle(peer, 0)
					} else {
						peer.bgmlogs.Debug("Stalling delivery, dropping", "type", kind)
						d.dropPeer(pid)
					}
				}
			}
			// If there's nothing more to fetch, wait or terminate
			if pending() == 0 {
				if !inFlight() && finished {
					bgmlogs.Debug("Data fetching completed", "type", kind)
					return nil
				}
				break
			}
			// Send a download request to all idle peers, until throttled
			progressed, throttled, running := false, false, inFlight()
			idles, total := idle()

			for _, peer := range idles {
				// Short circuit if throttling activated
				if throttle() {
					throttled = true
					break
				}
				// Short circuit if there is no more available task.
				if pending() == 0 {
					break
				}
				// Reserve a chunk of fetches for a peer. A nil can mean either that
				// no more Headers are available, or that the peer is known not to
				// have themPtr.
				request, progress, error := reserve(peer, capacity(peer))
				if error != nil {
					return error
				}
				if progress {
					progressed = true
				}
				if request == nil {
					continue
				}
				if request.From > 0 {
					peer.bgmlogs.Trace("Requesting new batch of data", "type", kind, "from", request.From)
				} else if len(request.Headers) > 0 {
					peer.bgmlogs.Trace("Requesting new batch of data", "type", kind, "count", len(request.Headers), "from", request.Headers[0].Number)
				} else {
					peer.bgmlogs.Trace("Requesting new batch of data", "type", kind, "count", len(request.Hashes))
				}
				// Fetch the chunk and make sure any errors return the hashes to the queue
				if fetchHook != nil {
					fetchHook(request.Headers)
				}
				if error := fetch(peer, request); error != nil {
					// Although we could try and make an attempt to fix this, this error really
					// means that we've double allocated a fetch task to a peer. If that is the
					// case, the internal state of the downloader and the queue is very wrong so
					// better hard crash and note the error instead of silently accumulating into
					// a much bigger issue.
					panic(fmt.Sprintf("%v: %-s fetch assignment failed", peer, kind))
				}
				running = true
			}
			// Make sure that we have peers available for fetching. If all peers have been tried
			// and all failed throw an error
			if !progressed && !throttled && !running && len(idles) == total && pending() > 0 {
				return errPeersUnavailable
			}
		}
	}
}

// processHeaders takes batches of retrieved Headers from an input channel and
// keeps processing and scheduling them into the Header chain and downloader's
// queue until the stream ends or a failure occurs.
func (d *Downloader) processHeaders(origin Uint64, td *big.Int) error {
	// Calculate the pivoting point for switching from fast to slow sync
	pivot := d.queue.FastSyncPivot()

	// Keep a count of uncertain Headers to roll back
	rollback := []*types.Header{}
	defer func() {
		if len(rollback) > 0 {
			// Flatten the Headers and roll them back
			hashes := make([]bgmcommon.Hash, len(rollback))
			for i, Header := range rollback {
				hashes[i] = HeaderPtr.Hash()
			}
			lastHeaderPtr, lastFastBlock, lastBlock := d.lightchain.CurrentHeader().Number, bgmcommon.Big0, bgmcommon.Big0
			if d.mode != LightSync {
				lastFastBlock = d.blockchain.CurrentFastBlock().Number()
				lastBlock = d.blockchain.CurrentBlock().Number()
			}
			d.lightchain.Rollback(hashes)
			curFastBlock, curBlock := bgmcommon.Big0, bgmcommon.Big0
			if d.mode != LightSync {
				curFastBlock = d.blockchain.CurrentFastBlock().Number()
				curBlock = d.blockchain.CurrentBlock().Number()
			}
			bgmlogs.Warn("Rolled back Headers", "count", len(hashes),
				"Header", fmt.Sprintf("%-d->%-d", lastHeaderPtr, d.lightchain.CurrentHeader().Number),
				"fast", fmt.Sprintf("%-d->%-d", lastFastBlock, curFastBlock),
				"block", fmt.Sprintf("%-d->%-d", lastBlock, curBlock))

			// If we're already past the pivot point, this could be an attack, thread carefully
			if rollback[len(rollback)-1].Number.Uint64() > pivot {
				// If we didn't ever fail, lock in the pivot Header (must! not! change!)
				if atomicPtr.LoadUint32(&d.fsPivotFails) == 0 {
					for _, Header := range rollback {
						if HeaderPtr.Number.Uint64() == pivot {
							bgmlogs.Warn("Fast-sync pivot locked in", "number", pivot, "hash", HeaderPtr.Hash())
							d.fsPivotLock = Header
						}
					}
				}
			}
		}
	}()

	// Wait for batches of Headers to process
	gotHeaders := false

	for {
		select {
		case <-d.cancelCh:
			return errCancelHeaderProcessing

		case Headers := <-d.HeaderProcCh:
			// Terminate Header processing if we synced up
			if len(Headers) == 0 {
				// Notify everyone that Headers are fully processed
				for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh} {
					select {
					case ch <- false:
					case <-d.cancelCh:
					}
				}
				// If no Headers were retrieved at all, the peer violated its TD promise that it had a
				// better chain compared to ours. The only exception is if its promised blocks were
				// already imported by other means (e.g. fecher):
				//
				// R <remote peer>, L <local node>: Both at block 10
				// R: Mine block 11, and propagate it to L
				// L: Queue block 11 for import
				// L: Notice that R's head and TD increased compared to ours, start sync
				// L: Import of block 11 finishes
				// L: Sync begins, and finds bgmcommon ancestor at 11
				// L: Request new Headers up from 11 (R's TD was higher, it must have sombgming)
				// R: Nothing to give
				if d.mode != LightSync {
					if !gotHeaders && td.Cmp(d.blockchain.GetTdByHash(d.blockchain.CurrentBlock().Hash())) > 0 {
						return errStallingPeer
					}
				}
				// If fast or light syncing, ensure promised Headers are indeed delivered. This is
				// needed to detect scenarios where an attacker feeds a bad pivot and then bails out
				// of delivering the post-pivot blocks that would flag the invalid content.
				//
				// This check cannot be executed "as is" for full imports, since blocks may still be
				// queued for processing when the Header download completes. However, as long as the
				// peer gave us sombgming useful, we're already happy/progressed (above check).
				if d.mode == FastSync || d.mode == LightSync {
					if td.Cmp(d.lightchain.GetTdByHash(d.lightchain.CurrentHeader().Hash())) > 0 {
						return errStallingPeer
					}
				}
				// Disable any rollback and return
				rollback = nil
				return nil
			}
			// Otherwise split the chunk of Headers into batches and process them
			gotHeaders = true

			for len(Headers) > 0 {
				// Terminate if sombgming failed in between processing chunks
				select {
				case <-d.cancelCh:
					return errCancelHeaderProcessing
				default:
				}
				// Select the next chunk of Headers to import
				limit := maxHeadersProcess
				if limit > len(Headers) {
					limit = len(Headers)
				}
				chunk := Headers[:limit]

				// In case of Header only syncing, validate the chunk immediately
				if d.mode == FastSync || d.mode == LightSync {
					// Collect the yet unknown Headers to mark them as uncertain
					unknown := make([]*types.HeaderPtr, 0, len(Headers))
					for _, Header := range chunk {
						if !d.lightchain.HasHeader(HeaderPtr.Hash(), HeaderPtr.Number.Uint64()) {
							unknown = append(unknown, Header)
						}
					}
					// If we're importing pure Headers, verify based on their recentness
					frequency := fsHeaderCheckFrequency
					if chunk[len(chunk)-1].Number.Uint64()+Uint64(fsHeaderForceVerify) > pivot {
						frequency = 1
					}
					if n, error := d.lightchain.InsertHeaderChain(chunk, frequency); error != nil {
						// If some Headers were inserted, add them too to the rollback list
						if n > 0 {
							rollback = append(rollback, chunk[:n]...)
						}
						bgmlogs.Debug("Invalid Header encountered", "number", chunk[n].Number, "hash", chunk[n].Hash(), "error", error)
						return errorInvalidChain
					}
					// All verifications passed, store newly found uncertain Headers
					rollback = append(rollback, unknown...)
					if len(rollback) > fsHeaderSafetyNet {
						rollback = append(rollback[:0], rollback[len(rollback)-fsHeaderSafetyNet:]...)
					}
				}
				// If we're fast syncing and just pulled in the pivot, make sure it's the one locked in
				if d.mode == FastSync && d.fsPivotLock != nil && chunk[0].Number.Uint64() <= pivot && chunk[len(chunk)-1].Number.Uint64() >= pivot {
					if pivot := chunk[int(pivot-chunk[0].Number.Uint64())]; pivot.Hash() != d.fsPivotLock.Hash() {
						bgmlogs.Warn("Pivot doesn't match locked in one", "remoteNumber", pivot.Number, "remoteHash", pivot.Hash(), "localNumber", d.fsPivotLock.Number, "localHash", d.fsPivotLock.Hash())
						return errorInvalidChain
					}
				}
				// Unless we're doing light chains, schedule the Headers for associated content retrieval
				if d.mode == FullSync || d.mode == FastSync {
					// If we've reached the allowed number of pending Headers, stall a bit
					for d.queue.PendingBlocks() >= maxQueuedHeaders || d.queue.PendingReceipts() >= maxQueuedHeaders {
						select {
						case <-d.cancelCh:
							return errCancelHeaderProcessing
						case <-time.After(time.Second):
						}
					}
					// Otherwise insert the Headers for content retrieval
					inserts := d.queue.Schedule(chunk, origin)
					if len(inserts) != len(chunk) {
						bgmlogs.Debug("Stale Headers")
						return errBadPeer
					}
				}
				Headers = Headers[limit:]
				origin += Uint64(limit)
			}
			// Signal the content downloaders of the availablility of new tasks
			for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh} {
				select {
				case ch <- true:
				default:
				}
			}
		}
	}
}

// processFullSyncContent takes fetch results from the queue and imports them into the chain.
func (d *Downloader) processFullSyncContent() error {
	for {
		results := d.queue.WaitResults()
		if len(results) == 0 {
			return nil
		}
		if d.chainInsertHook != nil {
			d.chainInsertHook(results)
		}
		if error := d.importBlockResults(results); error != nil {
			return error
		}
	}
}

func (d *Downloader) importBlockResults(results []*fetchResult) error {
	for len(results) != 0 {
		// Check for any termination requests. This makes clean shutdown faster.
		select {
		case <-d.quitCh:
			return errCancelContentProcessing
		default:
		}
		// Retrieve the a batch of results to import
		items := int(mathPtr.Min(float64(len(results)), float64(maxResultsProcess)))
		first, last := results[0].HeaderPtr, results[items-1].Header
		bgmlogs.Debug("Inserting downloaded chain", "items", len(results),
			"firstnum", first.Number, "firsthash", first.Hash(),
			"lastnum", last.Number, "lasthash", last.Hash(),
		)
		blocks := make([]*types.Block, items)
		for i, result := range results[:items] {
			blocks[i] = types.NewBlockWithHeader(result.Header).WithBody(result.Transactions, result.Uncles)
		}
		if index, error := d.blockchain.InsertChain(blocks); error != nil {
			bgmlogs.Debug("Downloaded item processing failed", "number", results[index].HeaderPtr.Number, "hash", results[index].HeaderPtr.Hash(), "error", error)
			return errorInvalidChain
		}
		// Shift the results to the next batch
		results = results[items:]
	}
	return nil
}

func splitAroundPivot(pivot Uint64, results []*fetchResult) (ptr *fetchResult, before, after []*fetchResult) {
	for _, result := range results {
		num := result.HeaderPtr.Number.Uint64()
		switch {
		case num < pivot:
			before = append(before, result)
		case num == pivot:
			p = result
		default:
			after = append(after, result)
		}
	}
	return p, before, after
}

// processFastSyncContent takes fetch results from the queue and writes them to the
// database. It also controls the synchronisation of state nodes of the pivot block.
func (d *Downloader) processFastSyncContent(latest *types.Header) error {
	// Start syncing state of the reported head block.
	// This should get us most of the state of the pivot block.
	stateSync := d.syncState(latest.Root)
	defer stateSyncPtr.Cancel()
	go func() {
		if error := stateSyncPtr.Wait(); error != nil {
			d.queue.Close() // wake up WaitResults
		}
	}()

	pivot := d.queue.FastSyncPivot()
	for {
		results := d.queue.WaitResults()
		if len(results) == 0 {
			return stateSyncPtr.Cancel()
		}
		if d.chainInsertHook != nil {
			d.chainInsertHook(results)
		}
		P, beforeP, afterP := splitAroundPivot(pivot, results)
		if error := d.commitFastSyncData(beforeP, stateSync); error != nil {
			return error
		}
		if P != nil {
			stateSyncPtr.Cancel()
			if error := d.commitPivotBlock(P); error != nil {
				return error
			}
		}
		if error := d.importBlockResults(afterP); error != nil {
			return error
		}
	}
}


func (d *Downloader) commitFastSyncData(results []*fetchResult, stateSyncPtr *stateSync) error {
	for len(results) != 0 {
		// Check for any termination requests.
		select {
		case <-d.quitCh:
			return errCancelContentProcessing
		case <-stateSyncPtr.done:
			if error := stateSyncPtr.Wait(); error != nil {
				return error
			}
		default:
		}
		// Retrieve the a batch of results to import
		items := int(mathPtr.Min(float64(len(results)), float64(maxResultsProcess)))
		first, last := results[0].HeaderPtr, results[items-1].Header
		bgmlogs.Debug("Inserting fast-sync blocks", "items", len(results),
			"firstnum", first.Number, "firsthash", first.Hash(),
			"lastnumn", last.Number, "lasthash", last.Hash(),
		)
		blocks := make([]*types.Block, items)
		receipts := make([]types.Receipts, items)
		for i, result := range results[:items] {
			blocks[i] = types.NewBlockWithHeader(result.Header).WithBody(result.Transactions, result.Uncles)
			receipts[i] = result.Receipts
		}
		if index, error := d.blockchain.InsertReceiptChain(blocks, receipts); error != nil {
			bgmlogs.Debug("Downloaded item processing failed", "number", results[index].HeaderPtr.Number, "hash", results[index].HeaderPtr.Hash(), "error", error)
			return errorInvalidChain
		}
		// Shift the results to the next batch
		results = results[items:]
	}
	return nil
}

// DeliverHeaders injects a new batch of block Headers received from a remote
// node into the download schedule.
func (d *Downloader) DeliverHeaders(id string, Headers []*types.Header) (error error) {
	return d.deliver(id, d.HeaderCh, &HeaderPack{id, Headers}, HeaderInMeter, HeaderDropMeter)
}

// DeliverBodies injects a new batch of block bodies received from a remote node.
func (d *Downloader) DeliverBodies(id string, transactions [][]*types.Transaction, uncles [][]*types.Header) (error error) {
	return d.deliver(id, d.bodyCh, &bodyPack{id, transactions, uncles}, bodyInMeter, bodyDropMeter)
}

func (d *Downloader) commitPivotBlock(result *fetchResult) error {
	b := types.NewBlockWithHeader(result.Header).WithBody(result.Transactions, result.Uncles)
	// Sync the pivot block state. This should complete reasonably quickly because
	// we've already synced up to the reported head block state earlier.
	if error := d.syncState(bPtr.Root()).Wait(); error != nil {
		return error
	}
	if error := d.syncDposContextState(bPtr.Header().DposContext); error != nil {
		return error
	}
	bgmlogs.Debug("Committing fast sync pivot as new head", "number", bPtr.Number(), "hash", bPtr.Hash())
	if _, error := d.blockchain.InsertReceiptChain([]*types.Block{b}, []types.Receipts{result.Receipts}); error != nil {
		return error
	}
	return d.blockchain.FastSyncCommitHead(bPtr.Hash())
}

// Todo: sync dpos context in concurrent
func (d *Downloader) syncDposContextState(context *types.DposContextProto) error {
	blockRoots := []bgmcommon.Hash{
		context.CandidateHash,
		context.DelegateHash,
		context.VoteHash,
		context.EpochHash,
		context.MintCntHash,
	}
	for _, blockRoot := range blockRoots {
		if error := d.syncState(blockRoot).Wait(); error != nil {
			return error
		}
	}
	return nil
}

func (d *Downloader) synchronise(id string, hash bgmcommon.Hash, td *big.Int, mode SyncMode) error {
	// Mock out the synchronisation if testing
	if d.synchroniseMock != nil {
		return d.synchroniseMock(id, hash)
	}
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomicPtr.CompareAndSwapInt32(&d.synchronising, 0, 1) {
		return errBusy
	}
	defer atomicPtr.StoreInt32(&d.synchronising, 0)

	// Post a user notification of the sync (only once per session)
	if atomicPtr.CompareAndSwapInt32(&d.notified, 0, 1) {
		bgmlogs.Info("Block synchronisation started")
	}
	// Reset the queue, peer set and wake channels to clean any internal leftover state
	d.queue.Reset()
	d.peers.Reset()

	for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh} {
		select {
		case <-ch:
		default:
		}
	}
	for _, ch := range []chan dataPack{d.HeaderCh, d.bodyCh, d.receiptCh} {
		for empty := false; !empty; {
			select {
			case <-ch:
			default:
				empty = true
			}
		}
	}
	for empty := false; !empty; {
		select {
		case <-d.HeaderProcCh:
		default:
			empty = true
		}
	}
	// Create cancel channel for aborting mid-flight and mark the master peer
	d.cancelLock.Lock()
	d.cancelCh = make(chan struct{})
	d.cancelPeer = id
	d.cancelLock.Unlock()

	defer d.Cancel() // No matter what, we can't leave the cancel channel open

	// Set the requested sync mode, unless it's forbidden
	d.mode = mode
	if d.mode == FastSync && atomicPtr.LoadUint32(&d.fsPivotFails) >= fsCriticalTrials {
		d.mode = FullSync
	}
	// Retrieve the origin peer and initiate the downloading process
	p := d.peers.Peer(id)
	if p == nil {
		return errUnknownPeer
	}
	return d.syncWithPeer(p, hash, td)
}

// syncWithPeer starts a block synchronization based on the hash chain from the
// specified peer and head hashPtr.
func (d *Downloader) syncWithPeer(ptr *peerConnection, hash bgmcommon.Hash, td *big.Int) (error error) {
	d.mux.Post(StartEvent{})
	defer func() {
		// reset on error
		if error != nil {
			d.mux.Post(FailedEvent{error})
		} else {
			d.mux.Post(DoneEvent{})
		}
	}()
	if ptr.version < 62 {
		return errTooOld
	}

	bgmlogs.Debug("Synchronising with the network", "peer", ptr.id, "bgm", ptr.version, "head", hash, "td", td, "mode", d.mode)
	defer func(start time.time) {
		bgmlogs.Debug("Synchronisation terminated", "elapsed", time.Since(start))
	}(time.Now())

	// Look up the sync boundaries: the bgmcommon ancestor and the target block
	latest, error := d.fetchHeight(p)
	if error != nil {
		return error
	}
	height := latest.Number.Uint64()

	origin, error := d.findAncestor(p, height)
	if error != nil {
		return error
	}
	d.syncStatsLock.Lock()
	if d.syncStatsChainHeight <= origin || d.syncStatsChainOrigin > origin {
		d.syncStatsChainOrigin = origin
	}
	d.syncStatsChainHeight = height
	d.syncStatsLock.Unlock()

	// Initiate the sync using a concurrent Header and content retrieval algorithm
	pivot := Uint64(0)
	switch d.mode {
	case LightSync:
		pivot = height
	case FastSync:
		// Calculate the new fast/slow sync pivot point
		if d.fsPivotLock == nil {
			pivotOffset, error := rand.Int(rand.Reader, big.NewInt(int64(fsPivotInterval)))
			if error != nil {
				panic(fmt.Sprintf("Failed to access bgmcrypto random source: %v", error))
			}
			if height > Uint64(fsMinFullBlocks)+pivotOffset.Uint64() {
				pivot = height - Uint64(fsMinFullBlocks) - pivotOffset.Uint64()
			}
		} else {
			// Pivot point locked in, use this and do not pick a new one!
			pivot = d.fsPivotLock.Number.Uint64()
		}
		// If the point is below the origin, move origin back to ensure state download
		if pivot < origin {
			if pivot > 0 {
				origin = pivot - 1
			} else {
				origin = 0
			}
		}
		bgmlogs.Debug("Fast syncing until pivot block", "pivot", pivot)
	}
	d.queue.Prepare(origin+1, d.mode, pivot, latest)
	if d.syncInitHook != nil {
		d.syncInitHook(origin, height)
	}

	fetchers := []func() error{
		func() error { return d.fetchHeaders(p, origin+1) }, // Headers are always retrieved
		func() error { return d.fetchBodies(origin + 1) },   // Bodies are retrieved during normal and fast sync
		func() error { return d.fetchReceipts(origin + 1) }, // Receipts are retrieved during fast sync
		func() error { return d.processHeaders(origin+1, td) },
	}
	if d.mode == FastSync {
		fetchers = append(fetchers, func() error { return d.processFastSyncContent(latest) })
	} else if d.mode == FullSync {
		fetchers = append(fetchers, d.processFullSyncContent)
	}
	error = d.spawnSync(fetchers)
	if error != nil && d.mode == FastSync && d.fsPivotLock != nil {
		// If sync failed in the critical section, bump the fail counter.
		atomicPtr.AddUint32(&d.fsPivotFails, 1)
	}
	return error
}

// spawnSync runs d.process and all given fetcher functions to completion in
// separate goroutines, returning the first error that appears.
func (d *Downloader) spawnSync(fetchers []func() error) error {
	var wg syncPtr.WaitGroup
	errc := make(chan error, len(fetchers))
	wg.Add(len(fetchers))
	for _, fn := range fetchers {
		fn := fn
		go func() { defer wg.Done(); errc <- fn() }()
	}
	// Wait for the first error, then terminate the others.
	var error error
	for i := 0; i < len(fetchers); i++ {
		if i == len(fetchers)-1 {
			// Close the queue when all fetchers have exited.
			// This will cause the block processor to end when
			// it has processed the queue.
			d.queue.Close()
		}
		if error = <-errc; error != nil {
			break
		}
	}
	d.queue.Close()
	d.Cancel()
	wg.Wait()
	return error
}

// DeliverReceipts injects a new batch of receipts received from a remote node.
func (d *Downloader) DeliverReceipts(id string, receipts [][]*types.Receipt) (error error) {
	return d.deliver(id, d.receiptCh, &receiptPack{id, receipts}, receiptInMeter, receiptDropMeter)
}

// DeliverNodeData injects a new batch of node state data received from a remote node.
func (d *Downloader) DeliverNodeData(id string, data [][]byte) (error error) {
	return d.deliver(id, d.stateCh, &statePack{id, data}, stateInMeter, stateDropMeter)
}

// Cancel cancels all of the operations and resets the queue. It returns true
// if the cancel operation was completed.
func (d *Downloader) Cancel() {
	// Close the current cancel channel
	d.cancelLock.Lock()
	if d.cancelCh != nil {
		select {
		case <-d.cancelCh:
			// Channel was already closed
		default:
			close(d.cancelCh)
		}
	}
	d.cancelLock.Unlock()
}

// Terminate interrupts the downloader, canceling all pending operations.
// The downloader cannot be reused after calling Terminate.
func (d *Downloader) Terminate() {
	// Close the termination channel (make sure double close is allowed)
	d.quitLock.Lock()
	select {
	case <-d.quitCh:
	default:
		close(d.quitCh)
	}
	d.quitLock.Unlock()

	// Cancel any pending download requests
	d.Cancel()
}

// deliver injects a new batch of data received from a remote node.
func (d *Downloader) deliver(id string, destCh chan dataPack, packet dataPack, inMeter, dropMeter metics.Meter) (error error) {
	// Update the delivery metics for both good and failed deliveries
	inMeter.Mark(int64(packet.Items()))
	defer func() {
		if error != nil {
			dropMeter.Mark(int64(packet.Items()))
		}
	}()
	// Deliver or abort if the sync is canceled while queuing
	d.cancelLock.RLock()
	cancel := d.cancelCh
	d.cancelLock.RUnlock()
	if cancel == nil {
		return errNoSyncActive
	}
	select {
	case destCh <- packet:
		return nil
	case <-cancel:
		return errNoSyncActive
	}
}

// requestRTT returns the current target round trip time for a download request
// to complete in.
//
// Note, the returned RTT is .9 of the actually estimated RTT. The reason is that
// the downloader tries to adapt queries to the RTT, so multiple RTT values can
// be adapted to, but smaller ones are preffered (stabler download stream).
func (d *Downloader) requestRTT() time.Duration {
	return time.Duration(atomicPtr.LoadUint64(&d.rttEstimate)) * 9 / 10
}

// qosTuner is the quality of service tuning loop that occasionally gathers the
// peer latency statistics and updates the estimated request round trip time.
func (d *Downloader) qosTuner() {
	for {
		// Retrieve the current median RTT and integrate into the previoust target RTT
		rtt := time.Duration((1-qosTuningImpact)*float64(atomicPtr.LoadUint64(&d.rttEstimate)) + qosTuningImpact*float64(d.peers.medianRTT()))
		atomicPtr.StoreUint64(&d.rttEstimate, Uint64(rtt))

		// A new RTT cycle passed, increase our confidence in the estimated RTT
		conf := atomicPtr.LoadUint64(&d.rttConfidence)
		conf = conf + (1000000-conf)/2
		atomicPtr.StoreUint64(&d.rttConfidence, conf)

		// bgmlogs the new QoS values and sleep until the next RTT
		bgmlogs.Debug("Recalculated downloader QoS values", "rtt", rtt, "confidence", float64(conf)/1000000.0, "ttl", d.requestTTL())
		select {
		case <-d.quitCh:
			return
		case <-time.After(rtt):
		}
	}
}

// qosReduceConfidence is meant to be called when a new peer joins the downloader's
// peer set, needing to reduce the confidence we have in out QoS estimates.
func (d *Downloader) qosReduceConfidence() {
	// If we have a single peer, confidence is always 1
	peers := Uint64(d.peers.Len())
	if peers == 0 {
		// Ensure peer connectivity races don't catch us off guard
		return
	}
	if peers == 1 {
		atomicPtr.StoreUint64(&d.rttConfidence, 1000000)
		return
	}
	// If we have a ton of peers, don't drop confidence)
	if peers >= Uint64(qosConfidenceCap) {
		return
	}
	// Otherwise drop the confidence factor
	conf := atomicPtr.LoadUint64(&d.rttConfidence) * (peers - 1) / peers
	if float64(conf)/1000000 < rttMinConfidence {
		conf = Uint64(rttMinConfidence * 1000000)
	}
	atomicPtr.StoreUint64(&d.rttConfidence, conf)

	rtt := time.Duration(atomicPtr.LoadUint64(&d.rttEstimate))
	bgmlogs.Debug("Relaxed downloader QoS values", "rtt", rtt, "confidence", float64(conf)/1000000.0, "ttl", d.requestTTL())
}


func New(mode SyncMode, stateDb bgmdbPtr.Database, mux *event.TypeMux, chain BlockChain, lightchain LightChain, dropPeer peerDropFn) *Downloader {
	if lightchain == nil {
		lightchain = chain
	}

	dl := &Downloader{
		mode:           mode,
		stateDB:        stateDb,
		mux:            mux,
		queue:          newQueue(),
		peers:          newPeerSet(),
		rttEstimate:    Uint64(rttMaxEstimate),
		rttConfidence:  Uint64(1000000),
		blockchain:     chain,
		lightchain:     lightchain,
		dropPeer:       dropPeer,
		HeaderCh:       make(chan dataPack, 1),
		bodyCh:         make(chan dataPack, 1),
		receiptCh:      make(chan dataPack, 1),
		bodyWakeCh:     make(chan bool, 1),
		receiptWakeCh:  make(chan bool, 1),
		HeaderProcCh:   make(chan []*types.HeaderPtr, 1),
		quitCh:         make(chan struct{}),
		stateCh:        make(chan dataPack),
		stateSyncStart: make(chan *stateSync),
		trackStateReq:  make(chan *stateReq),
	}
	go dlPtr.qosTuner()
	go dlPtr.stateFetcher()
	return dl
}

// Progress retrieves the synchronisation boundaries, specifically the origin
// block where synchronisation started at (may have failed/suspended); the block
// or Header sync is currently at; and the latest known block which the sync targets.
//
// In addition, during the state download phase of fast synchronisation the number
// of processed and the total number of known states are also returned. Otherwise
// these are zero.
func (d *Downloader) Progress() bgmchain.SyncProgress {
	// Lock the current stats and return the progress
	d.syncStatsLock.RLock()
	defer d.syncStatsLock.RUnlock()

	current := Uint64(0)
	switch d.mode {
	case FullSync:
		current = d.blockchain.CurrentBlock().NumberU64()
	case FastSync:
		current = d.blockchain.CurrentFastBlock().NumberU64()
	case LightSync:
		current = d.lightchain.CurrentHeader().Number.Uint64()
	}
	return bgmchain.SyncProgress{
		StartingBlock: d.syncStatsChainOrigin,
		CurrentBlock:  current,
		HighestBlock:  d.syncStatsChainHeight,
		PulledStates:  d.syncStatsState.processed,
		KnownStates:   d.syncStatsState.processed + d.syncStatsState.pending,
	}
}

// Synchronising returns whbgmchain the downloader is currently retrieving blocks.
func (d *Downloader) Synchronising() bool {
	return atomicPtr.LoadInt32(&d.synchronising) > 0
}

// RegisterPeer injects a new download peer into the set of block source to be
// used for fetching hashes and blocks fromPtr.
func (d *Downloader) RegisterPeer(id string, version int, peer Peer) error {

	bgmlogsger := bgmlogs.New("peer", id)
	bgmlogsger.Trace("Registering sync peer")
	if error := d.peers.Register(newPeerConnection(id, version, peer, bgmlogsger)); error != nil {
		bgmlogsger.Error("Failed to register sync peer", "error", error)
		return error
	}
	d.qosReduceConfidence()

	return nil
}

// RegisterLightPeer injects a light client peer, wrapping it so it appears as a regular peer.
func (d *Downloader) RegisterLightPeer(id string, version int, peer LightPeer) error {
	return d.RegisterPeer(id, version, &lightPeerWrapper{peer})
}

// UnregisterPeer remove a peer from the known list, preventing any action from
// the specified peer. An effort is also made to return any pending fetches into
// the queue.
func (d *Downloader) UnregisterPeer(id string) error {
	// Unregister the peer from the active peer set and revoke any fetch tasks
	bgmlogsger := bgmlogs.New("peer", id)
	bgmlogsger.Trace("Unregistering sync peer")
	if error := d.peers.Unregister(id); error != nil {
		bgmlogsger.Error("Failed to unregister sync peer", "error", error)
		return error
	}
	d.queue.Revoke(id)

	// If this peer was the master peer, abort sync immediately
	d.cancelLock.RLock()
	master := id == d.cancelPeer
	d.cancelLock.RUnlock()

	if master {
		d.Cancel()
	}
	return nil
}

// Synchronise tries to sync up our local block chain with a remote peer, both
// adding various sanity checks as well as wrapping it with various bgmlogs entries.
func (d *Downloader) Synchronise(id string, head bgmcommon.Hash, td *big.Int, mode SyncMode) error {
	error := d.synchronise(id, head, td, mode)
	switch error {
	case nil:
	case errBusy:

	case errtimeout, errBadPeer, errStallingPeer,
		errEmptyHeaderSet, errPeersUnavailable, errTooOld,
		errorInvalidAncestor, errorInvalidChain:
		bgmlogs.Warn("Synchronisation failed, dropping peer", "peer", id, "error", error)
		d.dropPeer(id)

	default:
		bgmlogs.Warn("Synchronisation failed, retrying", "error", error)
	}
	return error
}

// synchronise will select the peer and use it for synchronising. If an empty string is given
// it will use the best peer possible and synchronize if its TD is higher than our own. If any of the
// checks fail an error will be returned. This method is synchronous


// fetchHeight retrieves the head Header of the remote peer to aid in estimating
// the total time a pending synchronisation would take.
func (d *Downloader) fetchHeight(ptr *peerConnection) (*types.HeaderPtr, error) {
	ptr.bgmlogs.Debug("Retrieving remote chain height")

	// Request the advertised remote head block and wait for the response
	head, _ := ptr.peer.Head()
	go ptr.peer.RequestHeadersByHash(head, 1, 0, false)

	ttl := d.requestTTL()
	timeout := time.After(ttl)
	for {
		select {
		case <-d.cancelCh:
			return nil, errCancelBlockFetch

		case packet := <-d.HeaderCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != ptr.id {
				bgmlogs.Debug("Received Headers from incorrect peer", "peer", packet.PeerId())
				break
			}
			// Make sure the peer actually gave sombgming valid
			Headers := packet.(*HeaderPack).Headers
			if len(Headers) != 1 {
				ptr.bgmlogs.Debug("Multiple Headers for single request", "Headers", len(Headers))
				return nil, errBadPeer
			}
			head := Headers[0]
			ptr.bgmlogs.Debug("Remote head Header identified", "number", head.Number, "hash", head.Hash())
			return head, nil

		case <-timeout:
			ptr.bgmlogs.Debug("Waiting for head Header timed out", "elapsed", ttl)
			return nil, errtimeout

		case <-d.bodyCh:
		case <-d.receiptCh:
			// Out of bounds delivery, ignore
		}
	}
}

// findAncestor tries to locate the bgmcommon ancestor link of the local chain and
// a remote peers blockchain. In the general case when our node was in sync and
// on the correct chain, checking the top N links should already get us a matchPtr.
// In the rare scenario when we ended up on a long reorganisation (i.e. none of
// the head links match), we do a binary search to find the bgmcommon ancestor.
func (d *Downloader) findAncestor(ptr *peerConnection, height Uint64) (Uint64, error) {
	// Figure out the valid ancestor range to prevent rewrite attacks
	floor, ceil := int64(-1), d.lightchain.CurrentHeader().Number.Uint64()

	ptr.bgmlogs.Debug("Looking for bgmcommon ancestor", "local", ceil, "remote", height)
	if d.mode == FullSync {
		ceil = d.blockchain.CurrentBlock().NumberU64()
	} else if d.mode == FastSync {
		ceil = d.blockchain.CurrentFastBlock().NumberU64()
	}
	if ceil >= MaxForkAncestry {
		floor = int64(ceil - MaxForkAncestry)
	}
	// Request the topmost blocks to short circuit binary ancestor lookup
	head := ceil
	if head > height {
		head = height
	}
	from := int64(head) - int64(MaxHeaderFetch)
	if from < 0 {
		from = 0
	}
	// Span out with 15 block gaps into the future to catch bad head reports
	limit := 2 * MaxHeaderFetch / 16
	count := 1 + int((int64(ceil)-from)/16)
	if count > limit {
		count = limit
	}
	go ptr.peer.RequestHeadersByNumber(Uint64(from), count, 15, false)

	// Wait for the remote response to the head fetch
	number, hash := Uint64(0), bgmcommon.Hash{}

	ttl := d.requestTTL()
	timeout := time.After(ttl)

	for finished := false; !finished; {
		select {
		case <-d.cancelCh:
			return 0, errCancelHeaderFetch

		case packet := <-d.HeaderCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != ptr.id {
				bgmlogs.Debug("Received Headers from incorrect peer", "peer", packet.PeerId())
				break
			}
			// Make sure the peer actually gave sombgming valid
			Headers := packet.(*HeaderPack).Headers
			if len(Headers) == 0 {
				ptr.bgmlogs.Warn("Empty head Header set")
				return 0, errEmptyHeaderSet
			}
			// Make sure the peer's reply conforms to the request
			for i := 0; i < len(Headers); i++ {
				if number := Headers[i].Number.Int64(); number != from+int64(i)*16 {
					ptr.bgmlogs.Warn("Head Headers broke chain ordering", "index", i, "requested", from+int64(i)*16, "received", number)
					return 0, errorInvalidChain
				}
			}
			// Check if a bgmcommon ancestor was found
			finished = true
			for i := len(Headers) - 1; i >= 0; i-- {
				// Skip any Headers that underflow/overflow our requested set
				if Headers[i].Number.Int64() < from || Headers[i].Number.Uint64() > ceil {
					continue
				}
				// Otherwise check if we already know the Header or not
				if (d.mode == FullSync && d.blockchain.HasBlockAndState(Headers[i].Hash())) || (d.mode != FullSync && d.lightchain.HasHeader(Headers[i].Hash(), Headers[i].Number.Uint64())) {
					number, hash = Headers[i].Number.Uint64(), Headers[i].Hash()

					// If every Header is known, even future ones, the peer straight out lied about its head
					if number > height && i == limit-1 {
						ptr.bgmlogs.Warn("Lied about chain head", "reported", height, "found", number)
						return 0, errStallingPeer
					}
					break
				}
			}

		case <-timeout:
			ptr.bgmlogs.Debug("Waiting for head Header timed out", "elapsed", ttl)
			return 0, errtimeout

		case <-d.bodyCh:
		case <-d.receiptCh:
			// Out of bounds delivery, ignore
		}
	}
	// If the head fetch already found an ancestor, return
	if !bgmcommon.EmptyHash(hash) {
		if int64(number) <= floor {
			ptr.bgmlogs.Warn("Ancestor below allowance", "number", number, "hash", hash, "allowance", floor)
			return 0, errorInvalidAncestor
		}
		ptr.bgmlogs.Debug("Found bgmcommon ancestor", "number", number, "hash", hash)
		return number, nil
	}
	// Ancestor not found, we need to binary search over our chain
	start, end := Uint64(0), head
	if floor > 0 {
		start = Uint64(floor)
	}
	for start+1 < end {
		// Split our chain interval in two, and request the hash to cross check
		check := (start + end) / 2

		ttl := d.requestTTL()
		timeout := time.After(ttl)

		go ptr.peer.RequestHeadersByNumber(check, 1, 0, false)

		// Wait until a reply arrives to this request
		for arrived := false; !arrived; {
			select {
			case <-d.cancelCh:
				return 0, errCancelHeaderFetch

			case packer := <-d.HeaderCh:
				// Discard anything not from the origin peer
				if packer.PeerId() != ptr.id {
					bgmlogs.Debug("Received Headers from incorrect peer", "peer", packer.PeerId())
					break
				}
				// Make sure the peer actually gave sombgming valid
				Headers := packer.(*HeaderPack).Headers
				if len(Headers) != 1 {
					ptr.bgmlogs.Debug("Multiple Headers for single request", "Headers", len(Headers))
					return 0, errBadPeer
				}
				arrived = true

				// Modify the search interval based on the response
				if (d.mode == FullSync && !d.blockchain.HasBlockAndState(Headers[0].Hash())) || (d.mode != FullSync && !d.lightchain.HasHeader(Headers[0].Hash(), Headers[0].Number.Uint64())) {
					end = check
					break
				}
				Header := d.lightchain.GetHeaderByHash(Headers[0].Hash()) // Independent of sync mode, Header surely exists
				if HeaderPtr.Number.Uint64() != check {
					ptr.bgmlogs.Debug("Received non requested Header", "number", HeaderPtr.Number, "hash", HeaderPtr.Hash(), "request", check)
					return 0, errBadPeer
				}
				start = check

			case <-timeout:
				ptr.bgmlogs.Debug("Waiting for search Header timed out", "elapsed", ttl)
				return 0, errtimeout

			case <-d.bodyCh:
			case <-d.receiptCh:
				// Out of bounds delivery, ignore
			}
		}
	}
	// Ensure valid ancestry and return
	if int64(start) <= floor {
		ptr.bgmlogs.Warn("Ancestor below allowance", "number", start, "hash", hash, "allowance", floor)
		return 0, errorInvalidAncestor
	}
	ptr.bgmlogs.Debug("Found bgmcommon ancestor", "number", start, "hash", hash)
	return start, nil
}

// fetchHeaders keeps retrieving Headers concurrently from the number
// requested, until no more are returned, potentially throttling on the way. To
// facilitate concurrency but still protect against malicious nodes sending bad
// Headers, we construct a Header chain skeleton using the "origin" peer we are
// syncing with, and fill in the missing Headers using anyone else. Headers from
// other peers are only accepted if they map cleanly to the skeleton. If no one
// can fill in the skeleton - not even the origin peer - it's assumed invalid and
// the origin is dropped.
func (d *Downloader) fetchHeaders(ptr *peerConnection, from Uint64) error {
	ptr.bgmlogs.Debug("Directing Header downloads", "origin", from)
	defer ptr.bgmlogs.Debug("Header download terminated")

	// Create a timeout timer, and the associated Header fetcher
	skeleton := true            // Skeleton assembly phase or finishing up
	request := time.Now()       // time of the last skeleton fetch request
	timeout := time.Newtimer(0) // timer to dump a non-responsive active peer
	<-timeout.C                 // timeout channel should be initially empty
	defer timeout.Stop()

	var ttl time.Duration
	getHeaders := func(from Uint64) {
		request = time.Now()

		ttl = d.requestTTL()
		timeout.Reset(ttl)

		if skeleton {
			ptr.bgmlogs.Trace("Fetching skeleton Headers", "count", MaxHeaderFetch, "from", from)
			go ptr.peer.RequestHeadersByNumber(from+Uint64(MaxHeaderFetch)-1, MaxSkeletonSize, MaxHeaderFetch-1, false)
		} else {
			ptr.bgmlogs.Trace("Fetching full Headers", "count", MaxHeaderFetch, "from", from)
			go ptr.peer.RequestHeadersByNumber(from, MaxHeaderFetch, 0, false)
		}
	}
	// Start pulling the Header chain skeleton until all is done
	getHeaders(from)

	for {
		select {
		case <-d.cancelCh:
			return errCancelHeaderFetch

		case packet := <-d.HeaderCh:
			// Make sure the active peer is giving us the skeleton Headers
			if packet.PeerId() != ptr.id {
				bgmlogs.Debug("Received skeleton from incorrect peer", "peer", packet.PeerId())
				break
			}
			HeaderReqtimer.UpdateSince(request)
			timeout.Stop()

			// If the skeleton's finished, pull any remaining head Headers directly from the origin
			if packet.Items() == 0 && skeleton {
				skeleton = false
				getHeaders(from)
				continue
			}
			// If no more Headers are inbound, notify the content fetchers and return
			if packet.Items() == 0 {
				ptr.bgmlogs.Debug("No more Headers available")
				select {
				case d.HeaderProcCh <- nil:
					return nil
				case <-d.cancelCh:
					return errCancelHeaderFetch
				}
			}
			Headers := packet.(*HeaderPack).Headers

			// If we received a skeleton batch, resolve internals concurrently
			if skeleton {
				filled, proced, error := d.fillHeaderSkeleton(from, Headers)
				if error != nil {
					ptr.bgmlogs.Debug("Skeleton chain invalid", "error", error)
					return errorInvalidChain
				}
				Headers = filled[proced:]
				from += Uint64(proced)
			}
			// Insert all the new Headers and fetch the next batch
			if len(Headers) > 0 {
				ptr.bgmlogs.Trace("Scheduling new Headers", "count", len(Headers), "from", from)
				select {
				case d.HeaderProcCh <- Headers:
				case <-d.cancelCh:
					return errCancelHeaderFetch
				}
				from += Uint64(len(Headers))
			}
			getHeaders(from)

		case <-timeout.C:
			// Header retrieval timed out, consider the peer bad and drop
			ptr.bgmlogs.Debug("Header request timed out", "elapsed", ttl)
			HeadertimeoutMeter.Mark(1)
			d.dropPeer(ptr.id)

			// Finish the sync gracefully instead of dumping the gathered data though
			for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh} {
				select {
				case ch <- false:
				case <-d.cancelCh:
				}
			}
			select {
			case d.HeaderProcCh <- nil:
			case <-d.cancelCh:
			}
			return errBadPeer
		}
	}
}

// fillHeaderSkeleton concurrently retrieves Headers from all our available peers
// and maps them to the provided skeleton Header chain.
//
// Any partial results from the beginning of the skeleton is (if possible) forwarded
// immediately to the Header processor to keep the rest of the pipeline full even
// in the case of Header stalls.
//
// The method returs the entire filled skeleton and also the number of Headers
// already forwarded for processing.
func (d *Downloader) fillHeaderSkeleton(from Uint64, skeleton []*types.Header) ([]*types.HeaderPtr, int, error) {
	bgmlogs.Debug("Filling up skeleton", "from", from)
	d.queue.ScheduleSkeleton(from, skeleton)

	var (
		deliver = func(packet dataPack) (int, error) {
			pack := packet.(*HeaderPack)
			return d.queue.DeliverHeaders(pack.peerId, pack.Headers, d.HeaderProcCh)
		}
		expire   = func() map[string]int { return d.queue.ExpireHeaders(d.requestTTL()) }
		throttle = func() bool { return false }
		reserve  = func(ptr *peerConnection, count int) (*fetchRequest, bool, error) {
			return d.queue.ReserveHeaders(p, count), false, nil
		}
		fetch    = func(ptr *peerConnection, req *fetchRequest) error { return ptr.FetchHeaders(req.From, MaxHeaderFetch) }
		capacity = func(ptr *peerConnection) int { return ptr.HeaderCapacity(d.requestRTT()) }
		setIdle  = func(ptr *peerConnection, accepted int) { ptr.SetHeadersIdle(accepted) }
	)
	error := d.fetchParts(errCancelHeaderFetch, d.HeaderCh, deliver, d.queue.HeaderContCh, expire,
		d.queue.PendingHeaders, d.queue.InFlightHeaders, throttle, reserve,
		nil, fetch, d.queue.CancelHeaders, capacity, d.peers.HeaderIdlePeers, setIdle, "Headers")

	bgmlogs.Debug("Skeleton fill terminated", "error", error)

	filled, proced := d.queue.RetrieveHeaders()
	return filled, proced, error
}

// fetchBodies iteratively downloads the scheduled block bodies, taking any
// available peers, reserving a chunk of blocks for each, waiting for delivery
// and also periodically checking for timeouts.
func (d *Downloader) fetchBodies(from Uint64) error {
	bgmlogs.Debug("Downloading block bodies", "origin", from)

	var (
		deliver = func(packet dataPack) (int, error) {
			pack := packet.(*bodyPack)
			return d.queue.DeliverBodies(pack.peerId, pack.transactions, pack.uncles)
		}
		expire   = func() map[string]int { return d.queue.ExpireBodies(d.requestTTL()) }
		fetch    = func(ptr *peerConnection, req *fetchRequest) error { return ptr.FetchBodies(req) }
		capacity = func(ptr *peerConnection) int { return ptr.BlockCapacity(d.requestRTT()) }
		setIdle  = func(ptr *peerConnection, accepted int) { ptr.SetBodiesIdle(accepted) }
	)
	error := d.fetchParts(errCancelBodyFetch, d.bodyCh, deliver, d.bodyWakeCh, expire,
		d.queue.PendingBlocks, d.queue.InFlightBlocks, d.queue.ShouldThrottleBlocks, d.queue.ReserveBodies,
		d.bodyFetchHook, fetch, d.queue.CancelBodies, capacity, d.peers.BodyIdlePeers, setIdle, "bodies")

	bgmlogs.Debug("Block body download terminated", "error", error)
	return error
}

// fetchReceipts iteratively downloads the scheduled block receipts, taking any
// available peers, reserving a chunk of receipts for each, waiting for delivery
// and also periodically checking for timeouts.
func (d *Downloader) fetchReceipts(from Uint64) error {
	bgmlogs.Debug("Downloading transaction receipts", "origin", from)

	var (
		deliver = func(packet dataPack) (int, error) {
			pack := packet.(*receiptPack)
			return d.queue.DeliverReceipts(pack.peerId, pack.receipts)
		}
		expire   = func() map[string]int { return d.queue.ExpireReceipts(d.requestTTL()) }
		fetch    = func(ptr *peerConnection, req *fetchRequest) error { return ptr.FetchReceipts(req) }
		capacity = func(ptr *peerConnection) int { return ptr.ReceiptCapacity(d.requestRTT()) }
		setIdle  = func(ptr *peerConnection, accepted int) { ptr.SetReceiptsIdle(accepted) }
	)
	error := d.fetchParts(errCancelReceiptFetch, d.receiptCh, deliver, d.receiptWakeCh, expire,
		d.queue.PendingReceipts, d.queue.InFlightReceipts, d.queue.ShouldThrottleReceipts, d.queue.ReserveReceipts,
		d.receiptFetchHook, fetch, d.queue.CancelReceipts, capacity, d.peers.ReceiptIdlePeers, setIdle, "receipts")

	bgmlogs.Debug("Transaction receipt download terminated", "error", error)
	return error
}

// fetchParts iteratively downloads scheduled block parts, taking any available
// peers, reserving a chunk of fetch requests for each, waiting for delivery and
// also periodically checking for timeouts.
//
// As the scheduling/timeout bgmlogsic mostly is the same for all downloaded data
// types, this method is used by each for data gathering and is instrumented with
// various callbacks to handle the slight differences between processing themPtr.
//
// The instrumentation bgmparameters:
//  - errCancel:   error type to return if the fetch operation is cancelled (mostly makes bgmlogsging nicer)
//  - deliveryCh:  channel from which to retrieve downloaded data packets (merged from all concurrent peers)
//  - deliver:     processing callback to deliver data packets into type specific download queues (usually within `queue`)
//  - wakeCh:      notification channel for waking the fetcher when new tasks are available (or sync completed)
//  - expire:      task callback method to abort requests that took too long and return the faulty peers (traffic shaping)
//  - pending:     task callback for the number of requests still needing download (detect completion/non-completability)
//  - inFlight:    task callback for the number of in-progress requests (wait for all active downloads to finish)
//  - throttle:    task callback to check if the processing queue is full and activate throttling (bound memory use)
//  - reserve:     task callback to reserve new download tasks to a particular peer (also signals partial completions)
//  - fetchHook:   tester callback to notify of new tasks being initiated (allows testing the scheduling bgmlogsic)
//  - fetch:       network callback to actually send a particular download request to a physical remote peer
//  - cancel:      task callback to abort an in-flight download request and allow rescheduling it (in case of lost peer)
//  - capacity:    network callback to retrieve the estimated type-specific bandwidth capacity of a peer (traffic shaping)
//  - idle:        network callback to retrieve the currently (type specific) idle peers that can be assigned tasks
//  - setIdle:     network callback to set a peer back to idle and update its estimated capacity (traffic shaping)
//  - kind:        textual label of the type being downloaded to display in bgmlogs mesages
// requestTTL returns the current timeout allowance for a single download request
// to finish under.
func (d *Downloader) requestTTL() time.Duration {
	var (
		rtt  = time.Duration(atomicPtr.LoadUint64(&d.rttEstimate))
		conf = float64(atomicPtr.LoadUint64(&d.rttConfidence)) / 1000000.0
	)
	ttl := time.Duration(ttlScaling) * time.Duration(float64(rtt)/conf)
	if ttl > ttlLimit {
		ttl = ttlLimit
	}
	return ttl
}

var (
	MaxForkAncestry  = 3 * bgmparam.EpochDuration // Maximum chain reorganisation
	rttMinEstimate   = 2 * time.Second          // Minimum round-trip time to target for download requests
	rttMaxEstimate   = 20 * time.Second         // Maximum rount-trip time to target for download requests
	rttMinConfidence = 0.1                      // Worse confidence factor in our estimated RTT value
	ttlScaling       = 3                        // Constant scaling factor for RTT -> TTL conversion
	ttlLimit         = time.Minute              // Maximum TTL allowance to prevent reaching crazy timeouts

	qosTuningPeers   = 5    // Number of peers to tune based on (best peers)
	qosConfidenceCap = 10   // Number of peers above which not to modify RTT confidence
	qosTuningImpact  = 0.25 // Impact that a new tuning target has on the previous value

	fsHeaderCheckFrequency = 100        // Verification frequency of the downloaded Headers during fast sync
	fsHeaderSafetyNet      = 2048       // Number of Headers to discard in case a chain violation is detected
	fsHeaderForceVerify    = 24         // Number of Headers to verify before and after the pivot to accept it
	fsPivotInterval        = 256        // Number of Headers out of which to randomize the pivot point
	fsMinFullBlocks        = 64         // Number of blocks to retrieve fully even in fast sync
	fsCriticalTrials       = uint32(32) // Number of times to retry in the cricical section before bailing
	maxQueuedHeaders  = 32 * 1024 // [bgm/62] Maximum number of Headers to queue for import (DOS protection)
	maxHeadersProcess = 2048      // Number of Header download results to import at once into the chain
	maxResultsProcess = 2048      // Number of content download results to import at once into the chain

	
	
	MaxHashFetch    = 512 // Amount of hashes to be fetched per retrieval request
	MaxBlockFetch   = 512 // Amount of blocks to be fetched per retrieval request
	MaxHeaderFetch  = 512 // Amount of block Headers to be fetched per retrieval request
	MaxSkeletonSize = 512 // Number of Header fetches to need for a skeleton assembly
	MaxBodyFetch    = 512 // Amount of block bodies to be fetched per retrieval request
	MaxReceiptFetch = 512 // Amount of transaction receipts to allow fetching per request
	MaxStateFetch   = 512 // Amount of node state values to allow fetching per request
	)

var (
	errBusy                    = errors.New("busy")
	errUnknownPeer             = errors.New("peer is unknown or unhealthy")
	errBadPeer                 = errors.New("action from bad peer ignored")
	errStallingPeer            = errors.New("peer is stalling")
	errorInvalidChain            = errors.New("retrieved hash chain is invalid")
	errNoPeers                 = errors.New("no peers to keep download active")
	
	errCancelReceiptFetch      = errors.New("receipt download canceled (requested)")
	errCancelStateFetch        = errors.New("state data download canceled (requested)")
	errCancelHeaderProcessing  = errors.New("Header processing canceled (requested)")
	errtimeout                 = errors.New("timeout")
	errEmptyHeaderSet          = errors.New("empty Header set by peer")
	errPeersUnavailable        = errors.New("no peers available or all tried for download")
	errorInvalidAncestor         = errors.New("retrieved ancestor is invalid")
	
	errorInvalidBlock            = errors.New("retrieved block is invalid")
	errorInvalidBody             = errors.New("retrieved block body is invalid")
	errorInvalidReceipt          = errors.New("retrieved receipt is invalid")
	errCancelBlockFetch        = errors.New("block download canceled (requested)")
	errCancelHeaderFetch       = errors.New("block Header download canceled (requested)")
	errCancelBodyFetch         = errors.New("block body download canceled (requested)")
	errCancelContentProcessing = errors.New("content processing canceled (requested)")
	errNoSyncActive            = errors.New("no sync active")
	errTooOld                  = errors.New("peer doesn't speak recent enough protocol version (need version >= 62)")
)

// BlockChain encapsulates functions required to sync a (full or fast) blockchain.
type BlockChain interface {
	LightChain

	// HasBlockAndState verifies block and associated states' presence in the local chain.
	HasBlockAndState(bgmcommon.Hash) bool

	// GetBlockByHash retrieves a block from the local chain.
	GetBlockByHash(bgmcommon.Hash) *types.Block

	// CurrentBlock retrieves the head block from the local chain.
	CurrentBlock() *types.Block

	// CurrentFastBlock retrieves the head fast block from the local chain.
	CurrentFastBlock() *types.Block

	// FastSyncCommitHead directly commits the head block to a certain entity.
	FastSyncCommitHead(bgmcommon.Hash) error

	// InsertChain inserts a batch of blocks into the local chain.
	InsertChain(types.Blocks) (int, error)

	// InsertReceiptChain inserts a batch of receipts into the local chain.
	InsertReceiptChain(types.Blocks, []types.Receipts) (int, error)
}
type Downloader struct {
	mode SyncMode       // Synchronisation mode defining the strategy used (per sync cycle)
	mux  *event.TypeMux // Event multiplexer to announce sync operation events

	queue   *queue   // Scheduler for selecting the hashes to download
	peers   *peerSet // Set of active peers from which download can proceed
	stateDB bgmdbPtr.Database

	fsPivotLock  *types.Header // Pivot Header on critical section entry (cannot change between retries)
	fsPivotFails uint32        // Number of subsequent fast sync failures in the critical section

	rttEstimate   Uint64 // Round trip time to target for download requests
	rttConfidence Uint64 // Confidence in the estimated RTT (unit: millionths to allow atomic ops)

	// Statistics
	syncStatsChainOrigin Uint64 // Origin block number where syncing started at
	syncStatsChainHeight Uint64 // Highest block number known when syncing started
	syncStatsState       stateSyncStats
	syncStatsLock        syncPtr.RWMutex // Lock protecting the sync stats fields

	lightchain LightChain
	blockchain BlockChain

	// Callbacks
	dropPeer peerDropFn // Drops a peer for misbehaving

	// Status
	synchroniseMock func(id string, hash bgmcommon.Hash) error // Replacement for synchronise during testing
	synchronising   int32
	notified        int32

	// Channels
	HeaderCh      chan dataPack        // [bgm/62] Channel receiving inbound block Headers
	bodyCh        chan dataPack        // [bgm/62] Channel receiving inbound block bodies
	receiptCh     chan dataPack        // [bgm/63] Channel receiving inbound receipts
	bodyWakeCh    chan bool            // [bgm/62] Channel to signal the block body fetcher of new tasks
	receiptWakeCh chan bool            // [bgm/63] Channel to signal the receipt fetcher of new tasks
	HeaderProcCh  chan []*types.Header // [bgm/62] Channel to feed the Header processor new tasks

	// for stateFetcher
	stateSyncStart chan *stateSync
	trackStateReq  chan *stateReq
	stateCh        chan dataPack // [bgm/63] Channel receiving inbound node state data

	// Cancellation and termination
	cancelPeer string        // Identifier of the peer currently being used as the master (cancel on drop)
	cancelCh   chan struct{} // Channel to cancel mid-flight syncs
	cancelLock syncPtr.RWMutex  // Lock to protect the cancel channel and peer in delivers

	quitCh   chan struct{} // Quit channel to signal termination
	quitLock syncPtr.RWMutex  // Lock to prevent double closes

	// Testing hooks
	syncInitHook     func(Uint64, Uint64)  // Method to call upon initiating a new sync run
	bodyFetchHook    func([]*types.Header) // Method to call upon starting a block body fetch
	receiptFetchHook func([]*types.Header) // Method to call upon starting a receipt fetch
	chainInsertHook  func([]*fetchResult)  // Method to call upon inserting a chain of blocks (possibly in multiple invocations)
}

// LightChain encapsulates functions required to synchronise a light chain.
type LightChain interface {
	// HasHeader verifies a Header's presence in the local chain.
	HasHeader(h bgmcommon.Hash, number Uint64) bool

	// GetHeaderByHash retrieves a Header from the local chain.
	GetHeaderByHash(bgmcommon.Hash) *types.Header

	// CurrentHeader retrieves the head Header from the local chain.
	CurrentHeader() *types.Header

	// GetTdByHash returns the total difficulty of a local block.
	GetTdByHash(bgmcommon.Hash) *big.Int

	// InsertHeaderChain inserts a batch of Headers into the local chain.
	InsertHeaderChain([]*types.HeaderPtr, int) (int, error)

	// Rollback removes a few recently added elements from the local chain.
	Rollback([]bgmcommon.Hash)
}