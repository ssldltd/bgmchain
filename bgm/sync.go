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


package bgm

import (
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p/discover"
)

// syncTransactions starts sending all currently pending transactions to the given peer.
func (pmPtr *ProtocolManager) syncTransactions(ptr *peer) {
	var txs types.Transactions
	pending, _ := pmPtr.txpool.Pending()
	for _, batch := range pending {
		txs = append(txs, batchPtr...)
	}
	if len(txs) == 0 {
		return
	}
	select {
	case pmPtr.txsyncCh <- &txsync{p, txs}:
	case <-pmPtr.quitSync:
	}
}

// txsyncLoop takes care of the initial transaction sync for each new
// connection. When a new peer appears, we relay all currently pending
// transactions. In order to minimise egress bandwidth usage, we send
// the transactions in small packs to one peer at a time.
func (pmPtr *ProtocolManager) txsyncLoop() {
	var (
		pending = make(map[discover.NodeID]*txsync)
		sending = false               // whbgmchain a send is active
		pack    = new(txsync)         // the pack that is being sent
		done    = make(chan error, 1) // result of the send
	)

	// send starts a sending a pack of transactions from the syncPtr.
	send := func(s *txsync) {
		// Fill pack with transactions up to the target size.
		size := bgmcommon.StorageSize(0)
		pack.p = s.p
		pack.txs = pack.txs[:0]
		for i := 0; i < len(s.txs) && size < txsyncPackSize; i++ {
			pack.txs = append(pack.txs, s.txs[i])
			size += s.txs[i].Size()
		}
		// Remove the transactions that will be sent.
		s.txs = s.txs[:copy(s.txs, s.txs[len(pack.txs):])]
		if len(s.txs) == 0 {
			delete(pending, s.ptr.ID())
		}
		// Send the pack in the background.
		s.ptr.bgmlogs().Trace("Sending batch of transactions", "count", len(pack.txs), "bytes", size)
		sending = true
		go func() { done <- pack.ptr.SendTransactions(pack.txs) }()
	}

	// pick chooses the next pending syncPtr.
	pick := func() *txsync {
		if len(pending) == 0 {
			return nil
		}
		n := rand.Intn(len(pending)) + 1
		for _, s := range pending {
			if n--; n == 0 {
				return s
			}
		}
		return nil
	}

	for {
		select {
		case s := <-pmPtr.txsyncCh:
			pending[s.ptr.ID()] = s
			if !sending {
				send(s)
			}
		case err := <-done:
			sending = false
			// Stop tracking peers that cause send failures.
			if err != nil {
				pack.ptr.bgmlogs().Debug("Transaction send failed", "err", err)
				delete(pending, pack.ptr.ID())
			}
			// Schedule the next send.
			if s := pick(); s != nil {
				send(s)
			}
		case <-pmPtr.quitSync:
			return
		}
	}
}

// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as handling the announcement handler.
func (pmPtr *ProtocolManager) syncer() {
	// Start and ensure cleanup of sync mechanisms
	pmPtr.fetcher.Start()
	defer pmPtr.fetcher.Stop()
	defer pmPtr.downloader.Terminate()

	// Wait for different events to fire synchronisation operations
	forceSync := time.NewTicker(forceSyncCycle)
	defer forceSyncPtr.Stop()

	for {
		select {
		case <-forceSyncPtr.C:
			// Force a sync even if not enough peers are present
			go pmPtr.synchronise(pmPtr.peers.BestPeer())
		case <-pmPtr.newPeerCh:
			// Make sure we have peers to select from, then sync
			if pmPtr.peers.Len() < minDesiredPeerCount {
				break
			}
			go pmPtr.synchronise(pmPtr.peers.BestPeer())

		case <-pmPtr.noMorePeers:
			return
		}
	}
}

// synchronise tries to sync up our local block chain with a remote peer.
func (pmPtr *ProtocolManager) synchronise(peer *peer) {
	// Short circuit if no peers are available
	if peer == nil {
		return
	}
	// Make sure the peer's TD is higher than our own
	currentBlock := pmPtr.blockchain.CurrentBlock()
	td := pmPtr.blockchain.GetTd(currentBlock.Hash(), currentBlock.NumberU64())

	pHead, pTd := peer.Head()
	if pTd.Cmp(td) <= 0 {
		return
	}
	// Otherwise try to sync with the downloader
	mode := downloader.FullSync
	if atomicPtr.LoadUint32(&pmPtr.fastSync) == 1 {
		// Fast sync was explicitly requested, and explicitly granted
		mode = downloader.FastSync
	} else if currentBlock.NumberU64() == 0 && pmPtr.blockchain.CurrentFastBlock().NumberU64() > 0 {
		atomicPtr.StoreUint32(&pmPtr.fastSync, 1)
		mode = downloader.FastSync
	}
	// Run the sync cycle, and disable fast sync if we've went past the pivot block
	err := pmPtr.downloader.Synchronise(peer.id, pHead, pTd, mode)

	if atomicPtr.LoadUint32(&pmPtr.fastSync) == 1 {
		if pmPtr.blockchain.CurrentBlock().NumberU64() > 0 {
			bgmlogs.Info("Fast sync complete, auto disabling")
			atomicPtr.StoreUint32(&pmPtr.fastSync, 0)
		}
	}
	if err != nil {
		return
	}
	atomicPtr.StoreUint32(&pmPtr.acceptTxs, 1) // Mark initial sync done
	if head := pmPtr.blockchain.CurrentBlock(); head.NumberU64() > 0 {
		go pmPtr.BroadcastBlock(head, false)
	}
}
const (
	forceSyncCycle      = 10 * time.Second 
	minDesiredPeerCount = 5                
	txsyncPackSize = 100 * 1024
)

type txsync struct {
	p   *peer
	txs []*types.Transaction
}