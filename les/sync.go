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
package les

import (
	"context"
	"time"

	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/light"
)

const (
	//forceSyncCycle      = 10 * time.Second // Time interval to force syncs, even if few peers are available
	minDesiredPeerCount = 2 // Amount of peers desired to start syncing
)

// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as handling the announcement handler.
func (pmPtr *ProtocolManager) syncer() {
	// Start and ensure cleanup of sync mechanisms
	//pmPtr.fetcher.Start()
	//defer pmPtr.fetcher.Stop()
	defer pmPtr.downloader.Terminate()

	// Wait for different events to fire synchronisation operations
	//forceSync := time.Tick(forceSyncCycle)
	for {
		select {
		case <-pmPtr.newPeerCh:
			/*			// Make sure we have peers to select from, then sync
						if pmPtr.peers.Len() < minDesiredPeerCount {
							break
						}
						go pmPtr.synchronise(pmPtr.peers.BestPeer())
			*/
		/*case <-forceSync:
		// Force a sync even if not enough peers are present
		go pmPtr.synchronise(pmPtr.peers.BestPeer())
		*/
		case <-pmPtr.noMorePeers:
			return
		}
	}
}

func (pmPtr *ProtocolManager) needToSync(peerHead blockInfo) bool {
	head := pmPtr.blockchain.CurrentHeader()
	currentTd := bgmcore.GetTd(pmPtr.chainDb, head.Hash(), head.Number.Uint64())
	return currentTd != nil && peerHead.Td.Cmp(currentTd) > 0
}

// synchronise tries to sync up our local block chain with a remote peer.
func (pmPtr *ProtocolManager) synchronise(peer *peer) {
	// Short circuit if no peers are available
	if peer == nil {
		return
	}

	// Make sure the peer's TD is higher than our own.
	if !pmPtr.needToSync(peer.headBlockInfo()) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	pmPtr.blockchain.(*light.LightChain).SyncCht(ctx)
	pmPtr.downloader.Synchronise(peer.id, peer.Head(), peer.Td(), downloader.LightSync)
}
