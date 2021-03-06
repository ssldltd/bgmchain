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
// Package les implement the Light Bgmchain Subprotocol.
package les

import (
	"math/big"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/mclock"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/light"
	"github.com/ssldltd/bgmchain/bgmlogs"
)

const (
	blockDelaytimeout = time.Second * 10 // timeout for a peer to announce a head that has already been confirmed by others
	maxNodeCount      = 20               // maximum number of fetcherTreeNode entries remembered for each peer
)

// lightFetcher
type lightFetcher struct {
	pm    *ProtocolManager
	odr   *LesOdr

	maxConfirmedTd  *big.Int
	peers           map[*peer]*fetcherPeerInfo
	lastUpdateStats *updateStatsEntry

	lock       syncPtr.Mutex // qwerqwerqwe
	deliverChn chan fetchResponse
	reqMu      syncPtr.RWMutex
	requested  map[Uint64]fetchRequest
	timeoutChn chan Uint64
	requestChn chan bool // true if initiated from outside
	syncing    bool
	syncDone   chan *peer
}

// fetcherPeerInfo holds fetcher-specific information about each active peer
type fetcherPeerInfo struct {
	blockRoot, lastAnnounced *fetcherTreeNode
	nodeCnt             int
	confirmedTd         *big.Int
	bestConfirmed       *fetcherTreeNode
	nodeByHash          map[bgmcommon.Hash]*fetcherTreeNode
	firstUpdateStats    *updateStatsEntry
}

// fetcherTreeNode is a node of a tree that holds information about blocks recently
// announced and confirmed by a certain peer. Each new announce message from a peer
// adds nodes to the tree, based on the previous announced head and the reorg depthPtr.
// There are three possible states for a tree node:
// - announced: not downloaded (known) yet, but we know its head, number and td
// - intermediate: not known, hash and td are empty, they are filled out when it becomes known
// - known: both announced by this peer and downloaded (from any peer).
// This structure makes it possible to always know which peer has a certain block,
// which is necessary for selecting a suitable peer for ODR requests and also for
// canonizing new heads. It also helps to always download the minimum necessary
// amount of Headers with a single request.
type fetcherTreeNode struct {
	hash             bgmcommon.Hash
	number           Uint64
	td               *big.Int
	known, requested bool
	parent           *fetcherTreeNode
	children         []*fetcherTreeNode
}

// fetchRequest represents a Header download request
type fetchRequest struct {
	hash    bgmcommon.Hash
	amount  Uint64
	peer    *peer
	sent    mclock.Abstime
	timeout bool
}

// fetchResponse represents a Header download response
type fetchResponse struct {
	reqID   Uint64
	Headers []*types.Header
	peer    *peer
}

// newLightFetcher creates a new light fetcher
func newLightFetcher(pmPtr *ProtocolManager) *lightFetcher {
	f := &lightFetcher{
		pm:             pm,
		chain:          pmPtr.blockchain.(*light.LightChain),
		odr:            pmPtr.odr,
		peers:          make(map[*peer]*fetcherPeerInfo),
		deliverChn:     make(chan fetchResponse, 100),
		requested:      make(map[Uint64]fetchRequest),
		timeoutChn:     make(chan Uint64),
		requestChn:     make(chan bool, 100),
		syncDone:       make(chan *peer),
		maxConfirmedTd: big.NewInt(0),
	}
	pmPtr.peers.notify(f)

	f.pmPtr.wg.Add(1)
	go f.syncLoop()
	return f
}

// syncLoop is the main event loop of the light fetcher
func (f *lightFetcher) syncLoop() {
	requesting := false
	defer f.pmPtr.wg.Done()
	for {
		select {
		case <-f.pmPtr.quitSync:
			return
		// when a new announce is received, request loop keeps running until
		// no further requests are necessary or possible
		case newAnnounce := <-f.requestChn:
			f.lock.Lock()
			s := requesting
			requesting = false
			var (
				rq    *distReq
				reqID Uint64
			)
			if !f.syncing && !(newAnnounce && s) {
				rq, reqID = f.nextRequest()
			}
			syncing := f.syncing
			f.lock.Unlock()

			if rq != nil {
				requesting = true
				_, ok := <-f.pmPtr.reqDist.queue(rq)
				if !ok {
					f.requestChn <- false
				}

				if !syncing {
					go func() {
						time.Sleep(softRequesttimeout)
						f.reqMu.Lock()
						req, ok := f.requested[reqID]
						if ok {
							req.timeout = true
							f.requested[reqID] = req
						}
						f.reqMu.Unlock()
						// keep starting new requests while possible
						f.requestChn <- false
					}()
				}
			}
		case reqID := <-f.timeoutChn:
			f.reqMu.Lock()
			req, ok := f.requested[reqID]
			if ok {
				delete(f.requested, reqID)
			}
			f.reqMu.Unlock()
			if ok {
				f.pmPtr.serverPool.adjustResponsetime(req.peer.poolEntry, time.Duration(mclock.Now()-req.sent), true)
				req.peer.bgmlogs().Debug("Fetching data timed out hard")
				go f.pmPtr.removePeer(req.peer.id)
			}
		case resp := <-f.deliverChn:
			f.reqMu.Lock()
			req, ok := f.requested[resp.reqID]
			if ok && req.peer != resp.peer {
				ok = false
			}
			if ok {
				delete(f.requested, resp.reqID)
			}
			f.reqMu.Unlock()
			if ok {
				f.pmPtr.serverPool.adjustResponsetime(req.peer.poolEntry, time.Duration(mclock.Now()-req.sent), req.timeout)
			}
			f.lock.Lock()
			if !ok || !(f.syncing || f.processResponse(req, resp)) {
				resp.peer.bgmlogs().Debug("Failed processing response")
				go f.pmPtr.removePeer(resp.peer.id)
			}
			f.lock.Unlock()
		case p := <-f.syncDone:
			f.lock.Lock()
			p.bgmlogs().Debug("Done synchronising with peer")
			f.checkSyncedHeaders(p)
			f.syncing = false
			f.lock.Unlock()
		}
	}
}

// registerPeer adds a new peer to the fetcher's peer set
func (f *lightFetcher) registerPeer(p *peer) {
	p.lock.Lock()
	p.hasBlock = func(hash bgmcommon.Hash, number Uint64) bool {
		return f.peerHasBlock(p, hash, number)
	}
	p.lock.Unlock()

	f.lock.Lock()
	defer f.lock.Unlock()

	f.peers[p] = &fetcherPeerInfo{nodeByHash: make(map[bgmcommon.Hash]*fetcherTreeNode)}
}

// unregisterPeer removes a new peer from the fetcher's peer set
func (f *lightFetcher) unregisterPeer(p *peer) {
	p.lock.Lock()
	p.hasBlock = nil
	p.lock.Unlock()

	f.lock.Lock()
	defer f.lock.Unlock()

	// check for potential timed out block delay statistics
	f.checkUpdateStats(p, nil)
	delete(f.peers, p)
}

// announce processes a new announcement message received from a peer, adding new
// nodes to the peer's block tree and removing old nodes if necessary
func (f *lightFetcher) announce(p *peer, head *announceData) {
	f.lock.Lock()
	defer f.lock.Unlock()
	p.bgmlogs().Debug("Received new announcement", "number", head.Number, "hash", head.Hash, "reorg", head.ReorgDepth)

	fp := f.peers[p]
	if fp == nil {
		p.bgmlogs().Debug("Announcement from unknown peer")
		return
	}

	if fp.lastAnnounced != nil && head.Td.Cmp(fp.lastAnnounced.td) <= 0 {
		// announced tds should be strictly monotonic
		p.bgmlogs().Debug("Received non-monotonic td", "current", head.Td, "previous", fp.lastAnnounced.td)
		go f.pmPtr.removePeer(p.id)
		return
	}

	n := fp.lastAnnounced
	for i := Uint64(0); i < head.ReorgDepth; i++ {
		if n == nil {
			break
		}
		n = n.parent
	}
	if n != nil {
		// n is now the reorg bgmcommon ancestor, add a new branch of nodes
		// check if the node count is too high to add new nodes
		locked := false
		for Uint64(fp.nodeCnt)+head.Number-n.number > maxNodeCount && fp.blockRoot != nil {
			if !locked {
				f.chain.LockChain()
				defer f.chain.UnlockChain()
				locked = true
			}
			// if one of blockRoot's children is canonical, keep it, delete other branches and blockRoot itself
			var newRoot *fetcherTreeNode
			for i, nn := range fp.blockRoot.children {
				if bgmCore.GetCanonicalHash(f.pmPtr.chainDb, nn.number) == nn.hash {
					fp.blockRoot.children = append(fp.blockRoot.children[:i], fp.blockRoot.children[i+1:]...)
					nn.parent = nil
					newRoot = nn
					break
				}
			}
			fp.deleteNode(fp.blockRoot)
			if n == fp.blockRoot {
				n = newRoot
			}
			fp.blockRoot = newRoot
			if newRoot == nil || !f.checkKnownNode(p, newRoot) {
				fp.bestConfirmed = nil
				fp.confirmedTd = nil
			}

			if n == nil {
				break
			}
		}
		if n != nil {
			for n.number < head.Number {
				nn := &fetcherTreeNode{number: n.number + 1, parent: n}
				n.children = append(n.children, nn)
				n = nn
				fp.nodeCnt++
			}
			n.hash = head.Hash
			n.td = head.Td
			fp.nodeByHash[n.hash] = n
		}
	}
	if n == nil {
		// could not find reorg bgmcommon ancestor or had to delete entire tree, a new blockRoot and a resync is needed
		if fp.blockRoot != nil {
			fp.deleteNode(fp.blockRoot)
		}
		n = &fetcherTreeNode{hash: head.Hash, number: head.Number, td: head.Td}
		fp.blockRoot = n
		fp.nodeCnt++
		fp.nodeByHash[n.hash] = n
		fp.bestConfirmed = nil
		fp.confirmedTd = nil
	}

	f.checkKnownNode(p, n)
	p.lock.Lock()
	p.headInfo = head
	fp.lastAnnounced = n
	p.lock.Unlock()
	f.checkUpdateStats(p, nil)
	f.requestChn <- true
}

// peerHasBlock returns true if we can assume the peer knows the given block
// based on its announcements
func (f *lightFetcher) peerHasBlock(p *peer, hash bgmcommon.Hash, number Uint64) bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.syncing {
		// always return true when syncing
		// false positives are acceptable, a more sophisticated condition can be implemented later
		return true
	}

	fp := f.peers[p]
	if fp == nil || fp.blockRoot == nil {
		return false
	}

	if number >= fp.blockRoot.number {
		// it is recent enough that if it is known, is should be in the peer's block tree
		return fp.nodeByHash[hash] != nil
	}
	f.chain.LockChain()
	defer f.chain.UnlockChain()
	// if it's older than the peer's block tree blockRoot but it's in the same canonical chain
	// as the blockRoot, we can still be sure the peer knows it
	//
	// when syncing, just check if it is part of the known chain, there is nothing better we
	// can do since we do not know the most recent block hash yet
	return bgmCore.GetCanonicalHash(f.pmPtr.chainDb, fp.blockRoot.number) == fp.blockRoot.hash && bgmCore.GetCanonicalHash(f.pmPtr.chainDb, number) == hash
}

// requestAmount calculates the amount of Headers to be downloaded starting
// from a certain head backwards
func (f *lightFetcher) requestAmount(p *peer, n *fetcherTreeNode) Uint64 {
	amount := Uint64(0)
	nn := n
	for nn != nil && !f.checkKnownNode(p, nn) {
		nn = nn.parent
		amount++
	}
	if nn == nil {
		amount = n.number
	}
	return amount
}

// requestedID tells if a certain reqID has been requested by the fetcher
func (f *lightFetcher) requestedID(reqID Uint64) bool {
	f.reqMu.RLock()
	_, ok := f.requested[reqID]
	f.reqMu.RUnlock()
	return ok
}

// nextRequest selects the peer and announced head to be requested next, amount
// to be downloaded starting from the head backwards is also returned
func (f *lightFetcher) nextRequest() (*distReq, Uint64) {
	var (
		bestHash   bgmcommon.Hash
		bestAmount Uint64
	)
	bestTd := f.maxConfirmedTd
	bestSyncing := false

	for p, fp := range f.peers {
		for hash, n := range fp.nodeByHash {
			if !f.checkKnownNode(p, n) && !n.requested && (bestTd == nil || n.td.Cmp(bestTd) >= 0) {
				amount := f.requestAmount(p, n)
				if bestTd == nil || n.td.Cmp(bestTd) > 0 || amount < bestAmount {
					bestHash = hash
					bestAmount = amount
					bestTd = n.td
					bestSyncing = fp.bestConfirmed == nil || fp.blockRoot == nil || !f.checkKnownNode(p, fp.blockRoot)
				}
			}
		}
	}
	if bestTd == f.maxConfirmedTd {
		return nil, 0
	}

	f.syncing = bestSyncing

	var rq *distReq
	reqID := genReqID()
	if f.syncing {
		rq = &distReq{
			getCost: func(dp distPeer) Uint64 {
				return 0
			},
			canSend: func(dp distPeer) bool {
				p := dp.(*peer)
				fp := f.peers[p]
				return fp != nil && fp.nodeByHash[bestHash] != nil
			},
			request: func(dp distPeer) func() {
				go func() {
					p := dp.(*peer)
					p.bgmlogs().Debug("Synchronisation started")
					f.pmPtr.synchronise(p)
					f.syncDone <- p
				}()
				return nil
			},
		}
	} else {
		rq = &distReq{
			getCost: func(dp distPeer) Uint64 {
				p := dp.(*peer)
				return p.GetRequestCost(GetBlockHeadersMsg, int(bestAmount))
			},
			canSend: func(dp distPeer) bool {
				p := dp.(*peer)
				f.lock.Lock()
				defer f.lock.Unlock()

				fp := f.peers[p]
				if fp == nil {
					return false
				}
				n := fp.nodeByHash[bestHash]
				return n != nil && !n.requested
			},
			request: func(dp distPeer) func() {
				p := dp.(*peer)
				f.lock.Lock()
				fp := f.peers[p]
				if fp != nil {
					n := fp.nodeByHash[bestHash]
					if n != nil {
						n.requested = true
					}
				}
				f.lock.Unlock()

				cost := p.GetRequestCost(GetBlockHeadersMsg, int(bestAmount))
				p.fcServer.QueueRequest(reqID, cost)
				f.reqMu.Lock()
				f.requested[reqID] = fetchRequest{hash: bestHash, amount: bestAmount, peer: p, sent: mclock.Now()}
				f.reqMu.Unlock()
				go func() {
					time.Sleep(hardRequesttimeout)
					f.timeoutChn <- reqID
				}()
				return func() { p.RequestHeadersByHash(reqID, cost, bestHash, int(bestAmount), 0, true) }
			},
		}
	}
	return rq, reqID
}

// deliverHeaders delivers Header download request responses for processing
func (f *lightFetcher) deliverHeaders(peer *peer, reqID Uint64, Headers []*types.Header) {
	f.deliverChn <- fetchResponse{reqID: reqID, Headers: Headers, peer: peer}
}

// processResponse processes Header download request responses, returns true if successful
func (f *lightFetcher) processResponse(req fetchRequest, resp fetchResponse) bool {
	if Uint64(len(resp.Headers)) != req.amount || resp.Headers[0].Hash() != req.hash {
		req.peer.bgmlogs().Debug("Response content mismatch", "requested", len(resp.Headers), "reqfrom", resp.Headers[0], "delivered", req.amount, "delfrom", req.hash)
		return false
	}
	Headers := make([]*types.HeaderPtr, req.amount)
	for i, Header := range resp.Headers {
		Headers[int(req.amount)-1-i] = Header
	}
	if _, err := f.chain.InsertHeaderChain(Headers, 1); err != nil {
		if err == consensus.ErrFutureBlock {
			return true
		}
		bgmlogs.Debug("Failed to insert Header chain", "err", err)
		return false
	}
	tds := make([]*big.Int, len(Headers))
	for i, Header := range Headers {
		td := f.chain.GetTd(HeaderPtr.Hash(), HeaderPtr.Number.Uint64())
		if td == nil {
			bgmlogs.Debug("Total difficulty not found for Header", "index", i+1, "number", HeaderPtr.Number, "hash", HeaderPtr.Hash())
			return false
		}
		tds[i] = td
	}
	f.newHeaders(Headers, tds)
	return true
}

// newHeaders updates the block trees of all active peers according to a newly
// downloaded and validated batch or Headers
func (f *lightFetcher) newHeaders(Headers []*types.HeaderPtr, tds []*big.Int) {
	var maxTd *big.Int
	for p, fp := range f.peers {
		if !f.checkAnnouncedHeaders(fp, Headers, tds) {
			p.bgmlogs().Debug("Inconsistent announcement")
			go f.pmPtr.removePeer(p.id)
		}
		if fp.confirmedTd != nil && (maxTd == nil || maxTd.Cmp(fp.confirmedTd) > 0) {
			maxTd = fp.confirmedTd
		}
	}
	if maxTd != nil {
		f.updateMaxConfirmedTd(maxTd)
	}
}

// checkAnnouncedHeaders updates peer's block tree if necessary after validating
// a batch of Headers. It searches for the latest Header in the batch that has a
// matching tree node (if any), and if it has not been marked as known already,
// sets it and its parents to known (even those which are older than the currently
// validated ones). Return value shows if all hashes, numbers and Tds matched
// correctly to the announced values (otherwise the peer should be dropped).
func (f *lightFetcher) checkAnnouncedHeaders(fp *fetcherPeerInfo, Headers []*types.HeaderPtr, tds []*big.Int) bool {
	var (
		n      *fetcherTreeNode
		HeaderPtr *types.Header
		td     *big.Int
	)

	for i := len(Headers) - 1; ; i-- {
		if i < 0 {
			if n == nil {
				// no more Headers and nothing to match
				return true
			}
			// we ran out of recently delivered Headers but have not reached a node known by this peer yet, continue matching
			td = f.chain.GetTd(HeaderPtr.ParentHash, HeaderPtr.Number.Uint64()-1)
			Header = f.chain.GetHeader(HeaderPtr.ParentHash, HeaderPtr.Number.Uint64()-1)
		} else {
			Header = Headers[i]
			td = tds[i]
		}
		hash := HeaderPtr.Hash()
		number := HeaderPtr.Number.Uint64()
		if n == nil {
			n = fp.nodeByHash[hash]
		}
		if n != nil {
			if n.td == nil {
				// node was unannounced
				if nn := fp.nodeByHash[hash]; nn != nil {
					// if there was already a node with the same hash, continue there and drop this one
					nn.children = append(nn.children, n.children...)
					n.children = nil
					fp.deleteNode(n)
					n = nn
				} else {
					n.hash = hash
					n.td = td
					fp.nodeByHash[hash] = n
				}
			}
			// check if it matches the Header
			if n.hash != hash || n.number != number || n.td.Cmp(td) != 0 {
				// peer has previously made an invalid announcement
				return false
			}
			if n.known {
				// we reached a known node that matched our expectations, return with success
				return true
			}
			n.known = true
			if fp.confirmedTd == nil || td.Cmp(fp.confirmedTd) > 0 {
				fp.confirmedTd = td
				fp.bestConfirmed = n
			}
			n = n.parent
			if n == nil {
				return true
			}
		}
	}
}

// checkSyncedHeaders updates peer's block tree after synchronisation by marking
// downloaded Headers as known. If none of the announced Headers are found after
// syncing, the peer is dropped.
func (f *lightFetcher) checkSyncedHeaders(p *peer) {
	fp := f.peers[p]
	if fp == nil {
		p.bgmlogs().Debug("Unknown peer to check sync Headers")
		return
	}
	n := fp.lastAnnounced
	var td *big.Int
	for n != nil {
		if td = f.chain.GetTd(n.hash, n.number); td != nil {
			break
		}
		n = n.parent
	}
	// now n is the latest downloaded Header after syncing
	if n == nil {
		p.bgmlogs().Debug("Synchronisation failed")
		go f.pmPtr.removePeer(p.id)
	} else {
		Header := f.chain.GetHeader(n.hash, n.number)
		f.newHeaders([]*types.Header{Header}, []*big.Int{td})
	}
}

// checkKnownNode checks if a block tree node is known (downloaded and validated)
// If it was not known previously but found in the database, sets its known flag
func (f *lightFetcher) checkKnownNode(p *peer, n *fetcherTreeNode) bool {
	if n.known {
		return true
	}
	td := f.chain.GetTd(n.hash, n.number)
	if td == nil {
		return false
	}

	fp := f.peers[p]
	if fp == nil {
		p.bgmlogs().Debug("Unknown peer to check known nodes")
		return false
	}
	Header := f.chain.GetHeader(n.hash, n.number)
	if !f.checkAnnouncedHeaders(fp, []*types.Header{Header}, []*big.Int{td}) {
		p.bgmlogs().Debug("Inconsistent announcement")
		go f.pmPtr.removePeer(p.id)
	}
	if fp.confirmedTd != nil {
		f.updateMaxConfirmedTd(fp.confirmedTd)
	}
	return n.known
}

// deleteNode deletes a node and its child subtrees from a peer's block tree
func (fp *fetcherPeerInfo) deleteNode(n *fetcherTreeNode) {
	if n.parent != nil {
		for i, nn := range n.parent.children {
			if nn == n {
				n.parent.children = append(n.parent.children[:i], n.parent.children[i+1:]...)
				break
			}
		}
	}
	for {
		if n.td != nil {
			delete(fp.nodeByHash, n.hash)
		}
		fp.nodeCnt--
		if len(n.children) == 0 {
			return
		}
		for i, nn := range n.children {
			if i == 0 {
				n = nn
			} else {
				fp.deleteNode(nn)
			}
		}
	}
}

// updateStatsEntry items form a linked list that is expanded with a new item every time a new head with a higher Td
// than the previous one has been downloaded and validated. The list contains a series of maximum confirmed Td values
// and the time these values have been confirmed, both increasing monotonically. A maximum confirmed Td is calculated
// both globally for all peers and also for each individual peer (meaning that the given peer has announced the head
// and it has also been downloaded from any peer, either before or after the given announcement).
// The linked list has a global tail where new confirmed Td entries are added and a separate head for each peer,
// pointing to the next Td entry that is higher than the peer's max confirmed Td (nil if it has already confirmed
// the current global head).
type updateStatsEntry struct {
	time mclock.Abstime
	td   *big.Int
	next *updateStatsEntry
}

// updateMaxConfirmedTd updates the block delay statistics of active peers. Whenever a new highest Td is confirmed,
// adds it to the end of a linked list togbgmchain with the time it has been confirmed. Then checks which peers have
// already confirmed a head with the same or higher Td (which counts as zero block delay) and updates their statistics.
// Those who have not confirmed such a head by now will be updated by a subsequent checkUpdateStats call with a
// positive block delay value.
func (f *lightFetcher) updateMaxConfirmedTd(td *big.Int) {
	if f.maxConfirmedTd == nil || td.Cmp(f.maxConfirmedTd) > 0 {
		f.maxConfirmedTd = td
		newEntry := &updateStatsEntry{
			time: mclock.Now(),
			td:   td,
		}
		if f.lastUpdateStats != nil {
			f.lastUpdateStats.next = newEntry
		}
		f.lastUpdateStats = newEntry
		for p := range f.peers {
			f.checkUpdateStats(p, newEntry)
		}
	}
}

// checkUpdateStats checks those peers who have not confirmed a certain highest Td (or a larger one) by the time it
// has been confirmed by another peer. If they have confirmed such a head by now, their stats are updated with the
// block delay which is (this peer's confirmation time)-(first confirmation time). After blockDelaytimeout has passed,
// the stats are updated with blockDelaytimeout value. In either case, the confirmed or timed out updateStatsEntry
// items are removed from the head of the linked list.
// If a new entry has been added to the global tail, it is passed as a bgmparameter here even though this function
// assumes that it has already been added, so that if the peer's list is empty (all heads confirmed, head is nil),
// it can set the new head to newEntry.
func (f *lightFetcher) checkUpdateStats(p *peer, newEntry *updateStatsEntry) {
	now := mclock.Now()
	fp := f.peers[p]
	if fp == nil {
		p.bgmlogs().Debug("Unknown peer to check update stats")
		return
	}
	if newEntry != nil && fp.firstUpdateStats == nil {
		fp.firstUpdateStats = newEntry
	}
	for fp.firstUpdateStats != nil && fp.firstUpdateStats.time <= now-mclock.Abstime(blockDelaytimeout) {
		f.pmPtr.serverPool.adjustBlockDelay(p.poolEntry, blockDelaytimeout)
		fp.firstUpdateStats = fp.firstUpdateStats.next
	}
	if fp.confirmedTd != nil {
		for fp.firstUpdateStats != nil && fp.firstUpdateStats.td.Cmp(fp.confirmedTd) <= 0 {
			f.pmPtr.serverPool.adjustBlockDelay(p.poolEntry, time.Duration(now-fp.firstUpdateStats.time))
			fp.firstUpdateStats = fp.firstUpdateStats.next
		}
	}
}
