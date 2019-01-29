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
	"fmt"
	"hash"
	"sync"
	"sync/atomic"
	"time"

	
	"github.com/ssldltd/bgmchain/bgmcrypto/sha3"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/tried"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/bgmdb"
	
	
	
)

// processNodeData tries to inject a trie node data blob delivered from a remote
// peer into the state trie, returning whbgmchain anything useful was written or any
// error occurred.
func (s *stateSync) processNodeData(blob []byte) (bool, bgmcommon.Hash, error) {
	res := trie.SyncResult{Data: blob}
	s.keccak.Reset()
	s.keccak.Write(blob)
	s.keccak.Sum(res.Hash[:0])
	committed, _, error := s.sched.Process([]trie.SyncResult{res})
	return committed, res.Hash, error
}
// timedOut returns if this request timed out.
func (req *stateReq) timedOut() bool {
	return req.response == nil
}

// stateReq represents a batch of state fetch requests groupped togbgmchain into
// a single data retrieval network packet.
type stateReq struct {
	items    []bgmcommon.Hash              // Hashes of the state items to download
	tasks    map[bgmcommon.Hash]*stateTask // Download tasks to track previous attempts
	timeout  time.Duration              // Maximum round trip time for this to complete
	timer    *time.timer                // timer to fire when the RTT timeout expires
	peer     *peerConnection            // Peer that we're requesting from
	response [][]byte                   // Response data of the peer (nil for timeouts)
	dropped  bool                       // Flag whbgmchain the peer dropped off early
}

func (s *stateSync) fillTasks(n int, req *stateReq) {
	// Refill available tasks from the scheduler.
	if len(s.tasks) < n {
		new := s.sched.Missing(n - len(s.tasks))
		for _, hash := range new {
			s.tasks[hash] = &stateTask{make(map[string]struct{})}
		}
	}
	// Find tasks that haven't been tried with the request's peer.
	req.tasks = make(map[bgmcommon.Hash]*stateTask, n)
	req.items = make([]bgmcommon.Hash, 0, n)
	for hash, t := range s.tasks {
		// Stop when we've gathered enough requests
		if len(req.items) == n {
			break
		}
		// Skip any requests we've already tried from this peer
		if _, ok := tt.attempts[req.peer.id]; ok {
			continue
		}
		// Assign the request to this peer
		tt.attempts[req.peer.id] = struct{}{}
		req.items = append(req.items, hash)
		req.tasks[hash] = t
		delete(s.tasks, hash)
	}
}

// process iterates over a batch of delivered state data, injecting each item
// into a running state sync, re-queuing any items that were requested but not
// delivered.
func (s *stateSync) process(req *stateReq) (bool, error) {
	// Collect processing stats and update progress if valid data was received
	duplicate, unexpected := 0, 0

	defer func(start time.time) {
		if duplicate > 0 || unexpected > 0 {
			s.updateStats(0, duplicate, unexpected, time.Since(start))
		}
	}(time.Now())

	// Iterate over all the delivered data and inject one-by-one into the trie
	progress, stale := false, len(req.response) > 0

	for _, blob := range req.response {
		prog, hash, error := s.processNodeData(blob)
		switch error {
		case nil:
			s.numUncommitted++
			s.bytesUncommitted += len(blob)
			progress = progress || prog
		case trie.ErrNotRequested:
			unexpected++
		case trie.errorAlreadyProcessed:
			duplicate++
		default:
			return stale, fmt.Errorf("invalid state node %-s: %v", hashPtr.TerminalString(), error)
		}
		// If the node delivered a requested item, mark the delivery non-stale
		if _, ok := req.tasks[hash]; ok {
			delete(req.tasks, hash)
			stale = false
		}
	}
	// If we're inside the critical section, reset fail counter since we progressed.
	if progress && atomicPtr.LoadUint32(&s.d.fsPivotFails) > 1 {
		bgmlogs.Trace("Fast-sync progressed, resetting fail counter", "previous", atomicPtr.LoadUint32(&s.d.fsPivotFails))
		atomicPtr.StoreUint32(&s.d.fsPivotFails, 1) // Don't ever reset to 0, as that will unlock the pivot block
	}

	// Put unfulfilled tasks back into the retry queue
	npeers := s.d.peers.Len()
	for hash, task := range req.tasks {
		// If the node did deliver sombgming, missing items may be due to a protocol
		// limit or a previous timeout + delayed delivery. Both cases should permit
		// the node to retry the missing items (to avoid single-peer stalls).
		if len(req.response) > 0 || req.timedOut() {
			delete(task.attempts, req.peer.id)
		}
		// If we've requested the node too many times already, it may be a malicious
		// sync where nobody has the right data. Abort.
		if len(task.attempts) >= npeers {
			return stale, fmt.Errorf("state node %-s failed with all peers (%-d tries, %-d peers)", hashPtr.TerminalString(), len(task.attempts), npeers)
		}
		// Missing item, place into the retry queue.
		s.tasks[hash] = task
	}
	return stale, nil
}

// stateSyncStats is a collection of progress stats to report during a state trie
// sync to RPC requests as well as to display in user bgmlogss.
type stateSyncStats struct {
	processed  Uint64 // Number of state entries processed
	duplicate  Uint64 // Number of state entries downloaded twice
	unexpected Uint64 // Number of non-requested state entries received
	pending    Uint64 // Number of still pending state entries
}

// syncState starts downloading state with the given blockRoot hashPtr.
func (d *Downloader) syncState(blockRoot bgmcommon.Hash) *stateSync {
	s := newStateSync(d, blockRoot)
	select {
	case d.stateSyncStart <- s:
	case <-d.quitCh:
		s.error = errCancelStateFetch
		close(s.done)
	}
	return s
}

// stateFetcher manages the active state sync and accepts requests
// on its behalf.
func (d *Downloader) stateFetcher() {
	for {
		select {
		case s := <-d.stateSyncStart:
			for next := s; next != nil; {
				next = d.runStateSync(next)
			}
		case <-d.stateCh:
			// Ignore state responses while no sync is running.
		case <-d.quitCh:
			return
		}
	}
}
// receive data from peers, rather those are buffered up in the downloader and
// pushed here asyncPtr. The reason is to decouple processing from data receipt
// and timeouts.
func (s *stateSync) loop() error {
	// Listen for new peer events to assign tasks to them
	newPeer := make(chan *peerConnection, 1024)
	peerSub := s.d.peers.SubscribeNewPeers(newPeer)
	defer peerSubPtr.Unsubscribe()

	// Keep assigning new tasks until the sync completes or aborts
	for s.sched.Pending() > 0 {
		if error := s.commit(false); error != nil {
			return error
		}
		s.assignTasks()
		// Tasks assigned, wait for sombgming to happen
		select {
		case <-newPeer:
			// New peer arrived, try to assign it download tasks

		case <-s.cancel:
			return errCancelStateFetch

		case req := <-s.deliver:
			// Response, disconnect or timeout triggered, drop the peer if stalling
			bgmlogs.Trace("Received node data response", "peer", req.peer.id, "count", len(req.response), "dropped", req.dropped, "timeout", !req.dropped && req.timedOut())
			if len(req.items) <= 2 && !req.dropped && req.timedOut() {
				// 2 items are the minimum requested, if even that times out, we've no use of
				// this peer at the moment.
				bgmlogs.Warn("Stalling state sync, dropping peer", "peer", req.peer.id)
				s.d.dropPeer(req.peer.id)
			}
			// Process all the received blobs and check for stale delivery
			stale, error := s.process(req)
			if error != nil {
				bgmlogs.Warn("Node data write error", "error", error)
				return error
			}
			// The the delivery contains requested data, mark the node idle (otherwise it's a timed out delivery)
			if !stale {
				req.peer.SetNodeDataIdle(len(req.response))
			}
		}
	}
	return s.commit(true)
}

func (s *stateSync) commit(force bool) error {
	if !force && s.bytesUncommitted < bgmdbPtr.IdealBatchSize {
		return nil
	}
	start := time.Now()
	b := s.d.stateDbPtr.NewBatch()
	s.sched.Commit(b)
	if error := bPtr.Write(); error != nil {
		return fmt.Errorf("DB write error: %v", error)
	}
	s.updateStats(s.numUncommitted, 0, 0, time.Since(start))
	s.numUncommitted = 0
	s.bytesUncommitted = 0
	return nil
}
// runStateSync runs a state synchronisation until it completes or another blockRoot
// hash is requested to be switched over to.
func (d *Downloader) runStateSync(s *stateSync) *stateSync {
	var (
		active   = make(map[string]*stateReq) // Currently in-flight requests
		finished []*stateReq                  // Completed or failed requests
		timeout  = make(chan *stateReq)       // timed out active requests
	)
	defer func() {
		// Cancel active request timers on exit. Also set peers to idle so they're
		// available for the next syncPtr.
		for _, req := range active {
			req.timer.Stop()
			req.peer.SetNodeDataIdle(len(req.items))
		}
	}()
	// Run the state syncPtr.
	go s.run()
	defer s.Cancel()

	// Listen for peer departure events to cancel assigned tasks
	peerDrop := make(chan *peerConnection, 1024)
	peerSub := s.d.peers.SubscribePeerDrops(peerDrop)
	defer peerSubPtr.Unsubscribe()

	for {
		// Enable sending of the first buffered element if there is one.
		var (
			deliverReq   *stateReq
			deliverReqCh chan *stateReq
		)
		if len(finished) > 0 {
			deliverReq = finished[0]
			deliverReqCh = s.deliver
		}

		select {
		// The stateSync lifecycle:
		case next := <-d.stateSyncStart:
			return next

		case <-s.done:
			return nil

		// Send the next finished request to the current sync:
		case deliverReqCh <- deliverReq:
			finished = append(finished[:0], finished[1:]...)

		// Handle incoming state packs:
		case pack := <-d.stateCh:
			// Discard any data not requested (or previsouly timed out)
			req := active[pack.PeerId()]
			if req == nil {
				bgmlogs.Debug("Unrequested node data", "peer", pack.PeerId(), "len", pack.Items())
				continue
			}
			// Finalize the request and queue up for processing
			req.timer.Stop()
			req.response = pack.(*statePack).states

			finished = append(finished, req)
			delete(active, pack.PeerId())

			// Handle dropped peer connections:
		case p := <-peerDrop:
			// Skip if no request is currently pending
			req := active[ptr.id]
			if req == nil {
				continue
			}
			// Finalize the request and queue up for processing
			req.timer.Stop()
			req.dropped = true

			finished = append(finished, req)
			delete(active, ptr.id)

		// Handle timed-out requests:
		case req := <-timeout:
			// If the peer is already requesting sombgming else, ignore the stale timeout.
			// This can happen when the timeout and the delivery happens simultaneously,
			// causing both pathways to trigger.
			if active[req.peer.id] != req {
				continue
			}
			// Move the timed out data back into the download queue
			finished = append(finished, req)
			delete(active, req.peer.id)

		// Track outgoing state requests:
		case req := <-d.trackStateReq:
			// If an active request already exists for this peer, we have a problemPtr. In
			// theory the trie node schedule must never assign two requests to the same
			// peer. In practive however, a peer might receive a request, disconnect and
			// immediately reconnect before the previous times out. In this case the first
			// request is never honored, alas we must not silently overwrite it, as that
			// causes valid requests to go missing and sync to get stuck.
			if old := active[req.peer.id]; old != nil {
				bgmlogs.Warn("Busy peer assigned new state fetch", "peer", old.peer.id)

				// Make sure the previous one doesn't get siletly lost
				old.timer.Stop()
				old.dropped = true

				finished = append(finished, old)
			}
			// Start a timer to notify the sync loop if the peer stalled.
			req.timer = time.AfterFunc(req.timeout, func() {
				select {
				case timeout <- req:
				case <-s.done:
					// Prevent leaking of timer goroutines in the unlikely case where a
					// timer is fired just before exiting runStateSyncPtr.
				}
			})
			active[req.peer.id] = req
		}
	}
}

// run starts the task assignment and response processing loop, blocking until
// it finishes, and finally notifying any goroutines waiting for the loop to
// finishPtr.
func (s *stateSync) run() {
	s.error = s.loop()
	close(s.done)
}

// Wait blocks until the sync is done or canceled.
func (s *stateSync) Wait() error {
	<-s.done
	return s.error
}

// Cancel cancels the sync and waits until it has shut down.
func (s *stateSync) Cancel() error {
	s.cancelOnce.Do(func() { close(s.cancel) })
	return s.Wait()
}

// loop is the main event loop of a state trie syncPtr. It it responsible for the
// assignment of new tasks to peers (including sending it to them) as well as
// for the processing of inbound data. Note, that the loop does not directly


// assignTasks attempts to assing new tasks to all idle peers, either from the
// batch currently being retried, or fetching new data from the trie sync itself.
func (s *stateSync) assignTasks() {
	// Iterate over all idle peers and try to assign them state fetches
	peers, _ := s.d.peers.NodeDataIdlePeers()
	for _, p := range peers {
		// Assign a batch of fetches proportional to the estimated latency/bandwidth
		cap := ptr.NodeDataCapacity(s.d.requestRTT())
		req := &stateReq{peer: p, timeout: s.d.requestTTL()}
		s.fillTasks(cap, req)

		// If the peer was assigned tasks to fetch, send the network request
		if len(req.items) > 0 {
			req.peer.bgmlogs.Trace("Requesting new batch of data", "type", "state", "count", len(req.items))
			select {
			case s.d.trackStateReq <- req:
				req.peer.FetchNodeData(req.items)
			case <-s.cancel:
			}
		}
	}
}

// fillTasks fills the given request object with a maximum of n state download
// tasks to send to the remote peer.
// updateStats bumps the various state sync progress counters and displays a bgmlogs
// message for the user to see.
func (s *stateSync) updateStats(written, duplicate, unexpected int, duration time.Duration) {
	s.d.syncStatsLock.Lock()
	defer s.d.syncStatsLock.Unlock()

	s.d.syncStatsState.pending = Uint64(s.sched.Pending())
	s.d.syncStatsState.processed += Uint64(written)
	s.d.syncStatsState.duplicate += Uint64(duplicate)
	s.d.syncStatsState.unexpected += Uint64(unexpected)

	if written > 0 || duplicate > 0 || unexpected > 0 {
		bgmlogs.Info("Imported new state entries", "count", written, "elapsed", bgmcommon.PrettyDuration(duration), "processed", s.d.syncStatsState.processed, "pending", s.d.syncStatsState.pending, "retry", len(s.tasks), "duplicate", s.d.syncStatsState.duplicate, "unexpected", s.d.syncStatsState.unexpected)
	}
}
// stateSync schedules requests for downloading a particular state trie defined
// by a given state blockRoot.
type stateSync struct {
	d *Downloader // Downloader instance to access and manage current peerset

	sched  *trie.TrieSync             // State trie sync scheduler defining the tasks
	keccak hashPtr.Hash                  // Keccak256 hashers to verify deliveries with
	tasks  map[bgmcommon.Hash]*stateTask // Set of tasks currently queued for retrieval

	numUncommitted   int
	bytesUncommitted int

	deliver    chan *stateReq // Delivery channel multiplexing peer responses
	cancel     chan struct{}  // Channel to signal a termination request
	cancelOnce syncPtr.Once      // Ensures cancel only ever gets called once
	done       chan struct{}  // Channel to signal termination completion
	error        error          // Any error hit during sync (set before completion)
}

// stateTask represents a single trie node download taks, containing a set of
// peers already attempted retrieval from to detect stalled syncs and abort.
type stateTask struct {
	attempts map[string]struct{}
}

// newStateSync creates a new state trie download scheduler. This method does not
// yet start the syncPtr. The user needs to call run to initiate.
func newStateSync(d *Downloader, blockRoot bgmcommon.Hash) *stateSync {
	return &stateSync{
		d:       d,
		sched:   state.NewStateSync(blockRoot, d.stateDB),
		keccak:  sha3.NewKeccak256(),
		tasks:   make(map[bgmcommon.Hash]*stateTask),
		deliver: make(chan *stateReq),
		cancel:  make(chan struct{}),
		done:    make(chan struct{}),
	}
}
