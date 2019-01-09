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
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/consensus/misc"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgm/fetcher"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
)

// errIncompatibleConfig is returned if the requested protocols and configs are
// not compatible (low protocol version restrictions and high requirements).
var errIncompatibleConfig = errors.New("incompatible configuration")

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type ProtocolManager struct {
	networkId uint64

	fastSync  uint32 // Flag whbgmchain fast sync is enabled (gets disabled if we already have blocks)
	acceptTxs uint32 // Flag whbgmchain we're considered synchronised (enables transaction processing)

	txpool      txPool
	blockchain  *bgmcore.BlockChain
	chaindb     bgmdbPtr.Database
	chainconfig *bgmparam.ChainConfig
	maxPeers    int

	downloader *downloader.Downloader
	fetcher    *fetcher.Fetcher
	peers      *peerSet

	SubProtocols []p2p.Protocol

	eventMux      *event.TypeMux
	txCh          chan bgmcore.TxPreEvent
	txSub         event.Subscription
	minedBlockSubPtr *event.TypeMuxSubscription

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh   chan *peer
	txsyncCh    chan *txsync
	quitSync    chan struct{}
	noMorePeers chan struct{}

	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg syncPtr.WaitGroup
}

// NewProtocolManager returns a new bgmchain sub protocol manager. The Bgmchain sub protocol manages peers capable
// with the bgmchain network.
func NewProtocolManager(config *bgmparam.ChainConfig, mode downloader.SyncMode, networkId uint64, mux *event.TypeMux, txpool txPool, engine consensus.Engine, blockchain *bgmcore.BlockChain, chaindb bgmdbPtr.Database) (*ProtocolManager, error) {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		networkId:   networkId,
		eventMux:    mux,
		txpool:      txpool,
		blockchain:  blockchain,
		chaindb:     chaindb,
		chainconfig: config,
		peers:       newPeerSet(),
		newPeerCh:   make(chan *peer),
		noMorePeers: make(chan struct{}),
		txsyncCh:    make(chan *txsync),
		quitSync:    make(chan struct{}),
	}
	// Figure out whbgmchain to allow fast sync or not
	if mode == downloader.FastSync && blockchain.CurrentBlock().NumberU64() > 0 {
		bgmlogs.Warn("Blockchain not empty, fast sync disabled")
		mode = downloader.FullSync
	}
	if mode == downloader.FastSync {
		manager.fastSync = uint32(1)
	}
	// Initiate a sub-protocol for every implemented version we can handle
	manager.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {
		// Skip protocol version if incompatible with the mode of operation
		if mode == downloader.FastSync && version < bgm63 {
			continue
		}
		// Compatible; initialise the sub-protocol
		version := version // Closure for the run
		manager.SubProtocols = append(manager.SubProtocols, p2p.Protocol{
			Name:    ProtocolName,
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(ptr *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := manager.newPeer(int(version), p, rw)
				select {
				case manager.newPeerCh <- peer:
					manager.wg.Add(1)
					defer manager.wg.Done()
					return manager.handle(peer)
				case <-manager.quitSync:
					return p2p.DiscQuitting
				}
			},
			NodeInfo: func() interface{} {
				return manager.NodeInfo()
			},
			PeerInfo: func(id discover.NodeID) interface{} {
				if p := manager.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
					return ptr.Info()
				}
				return nil
			},
		})
	}
	if len(manager.SubProtocols) == 0 {
		return nil, errIncompatibleConfig
	}
	// Construct the different synchronisation mechanisms
	manager.downloader = downloader.New(mode, chaindb, manager.eventMux, blockchain, nil, manager.removePeer)

	validator := func(headerPtr *types.Header) error {
		return engine.VerifyHeader(blockchain, headerPtr, true)
	}
	heighter := func() uint64 {
		return blockchain.CurrentBlock().NumberU64()
	}
	inserter := func(blocks types.Blocks) (int, error) {
		// If fast sync is running, deny importing weird blocks
		if atomicPtr.LoadUint32(&manager.fastSync) == 1 {
			bgmlogs.Warn("Discarded bad propagated block", "number", blocks[0].Number(), "hash", blocks[0].Hash())
			return 0, nil
		}
		atomicPtr.StoreUint32(&manager.acceptTxs, 1) // Mark initial sync done on any fetcher import
		return manager.blockchain.InsertChain(blocks)
	}
	manager.fetcher = fetcher.New(blockchain.GetBlockByHash, validator, manager.BroadcastBlock, heighter, inserter, manager.removePeer)

	return manager, nil
}

func (pmPtr *ProtocolManager) removePeer(id string) {
	// Short circuit if the peer was already removed
	peer := pmPtr.peers.Peer(id)
	if peer == nil {
		return
	}
	bgmlogs.Debug("Removing Bgmchain peer", "peer", id)

	// Unregister the peer from the downloader and Bgmchain peer set
	pmPtr.downloader.UnregisterPeer(id)
	if err := pmPtr.peers.Unregister(id); err != nil {
		bgmlogs.Error("Peer removal failed", "peer", id, "err", err)
	}
	// Hard disconnect at the networking layer
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

func (pmPtr *ProtocolManager) Start(maxPeers int) {
	pmPtr.maxPeers = maxPeers

	// Broadscast transactions
	pmPtr.txCh = make(chan bgmcore.TxPreEvent, txChanSize)
	pmPtr.txSub = pmPtr.txpool.SubscribeTxPreEvent(pmPtr.txCh)
	go pmPtr.txBroadcastLoop()

	// Broadscast mined blocks
	pmPtr.minedBlockSub = pmPtr.eventMux.Subscribe(bgmcore.NewMinedBlockEvent{})
	go pmPtr.minedBroadcastLoop()

	// start sync handlers
	go pmPtr.syncer()
	go pmPtr.txsyncLoop()
}

func (pmPtr *ProtocolManager) Stop() {
	bgmlogs.Info("Stopping Bgmchain protocol")

	pmPtr.txSubPtr.Unsubscribe()         // quits txBroadcastLoop
	pmPtr.minedBlockSubPtr.Unsubscribe() // quits blockBroadcastLoop

	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	pmPtr.noMorePeers <- struct{}{}

	// Quit fetcher, txsyncLoop.
	close(pmPtr.quitSync)

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to pmPtr.peers yet
	// will exit when they try to register.
	pmPtr.peers.Close()

	// Wait for all peer handler goroutines and the loops to come down.
	pmPtr.wg.Wait()

	bgmlogs.Info("Bgmchain protocol stopped")
}

func (pmPtr *ProtocolManager) newPeer(pv int, ptr *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, p, newMeteredMsgWriter(rw))
}

// handle is the callback invoked to manage the life cycle of an bgm peer. When
// this function terminates, the peer is disconnected.
func (pmPtr *ProtocolManager) handle(ptr *peer) error {
	if pmPtr.peers.Len() >= pmPtr.maxPeers {
		return p2p.DiscTooManyPeers
	}
	ptr.bgmlogs().Debug("Bgmchain peer connected", "name", ptr.Name())

	// Execute the Bgmchain handshake
	td, head, genesis := pmPtr.blockchain.Status()
	if err := ptr.Handshake(pmPtr.networkId, td, head, genesis); err != nil {
		ptr.bgmlogs().Debug("Bgmchain handshake failed", "err", err)
		return err
	}
	if rw, ok := ptr.rw.(*meteredMsgReadWriter); ok {
		rw.Init(ptr.version)
	}
	// Register the peer locally
	if err := pmPtr.peers.Register(p); err != nil {
		ptr.bgmlogs().Error("Bgmchain peer registration failed", "err", err)
		return err
	}
	defer pmPtr.removePeer(ptr.id)

	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if err := pmPtr.downloader.RegisterPeer(ptr.id, ptr.version, p); err != nil {
		return err
	}
	// Propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	pmPtr.syncTransactions(p)

	// If we're DAO hard-fork aware, validate any remote peer with regard to the hard-fork
	if daoBlock := pmPtr.chainconfig.DAOForkBlock; daoBlock != nil {
		// Request the peer's DAO fork header for extra-data validation
		if err := ptr.RequestHeadersByNumber(daoBlock.Uint64(), 1, 0, false); err != nil {
			return err
		}
		// Start a timer to disconnect if the peer doesn't reply in time
		ptr.forkDrop = time.AfterFunc(daoChallengeTimeout, func() {
			ptr.bgmlogs().Debug("Timed out DAO fork-check, dropping")
			pmPtr.removePeer(ptr.id)
		})
		// Make sure it's cleaned up if the peer dies off
		defer func() {
			if ptr.forkDrop != nil {
				ptr.forkDrop.Stop()
				ptr.forkDrop = nil
			}
		}()
	}
	// main loop. handle incoming messages.
	for {
		if err := pmPtr.handleMessage(p); err != nil {
			ptr.bgmlogs().Debug("Bgmchain message handling failed", "err", err)
			return err
		}
	}
}

// Mined Broadscast loop
func (self *ProtocolManager) minedBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range self.minedBlockSubPtr.Chan() {
		switch ev := obj.Data.(type) {
		case bgmcore.NewMinedBlockEvent:
			self.BroadcastBlock(ev.Block, true)  // First propagate block to peers
			self.BroadcastBlock(ev.Block, false) // Only then announce to the rest
		}
	}
}

func (self *ProtocolManager) txBroadcastLoop() {
	for {
		select {
		case event := <-self.txCh:
			self.BroadcastTx(event.Tx.Hash(), event.Tx)

		// Err() channel will be closed when unsubscribing.
		case <-self.txSubPtr.Err():
			return
		}
	}
}
// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (pmPtr *ProtocolManager) handleMessage(ptr *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := ptr.rw.ReadMessage()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	// Handle the message depending on its contents
	switch {
	case msg.Code == StatusMsg:
		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	// Block header query, collect the requested headers and reply
	case msg.Code == GetBlockHeadersMsg:
		// Decode the complex header query
		var query getBlockHeadersData
		if err := msg.Decode(&query); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		hashMode := query.Origin.Hash != (bgmcommon.Hash{})

		// Gather headers until the fetch or network limits is reached
		var (
			bytes   bgmcommon.StorageSize
			headers []*types.Header
			unknown bool
		)
		for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < downloader.MaxHeaderFetch {
			// Retrieve the next header satisfying the query
			var origin *types.Header
			if hashMode {
				origin = pmPtr.blockchain.GetHeaderByHash(query.Origin.Hash)
			} else {
				origin = pmPtr.blockchain.GetHeaderByNumber(query.Origin.Number)
			}
			if origin == nil {
				break
			}
			number := origin.Number.Uint64()
			headers = append(headers, origin)
			bytes += estHeaderRlpSize

			// Advance to the next header of the query
			switch {
			case query.Origin.Hash != (bgmcommon.Hash{}) && query.Reverse:
				// Hash based traversal towards the genesis block
				for i := 0; i < int(query.Skip)+1; i++ {
					if header := pmPtr.blockchain.GetHeader(query.Origin.Hash, number); header != nil {
						query.Origin.Hash = headerPtr.ParentHash
						number--
					} else {
						unknown = true
						break
					}
				}
			case query.Origin.Hash != (bgmcommon.Hash{}) && !query.Reverse:
				// Hash based traversal towards the leaf block
				var (
					current = origin.Number.Uint64()
					next    = current + query.Skip + 1
				)
				if next <= current {
					infos, _ := json.MarshalIndent(ptr.Peer.Info(), "", "  ")
					ptr.bgmlogs().Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", infos)
					unknown = true
				} else {
					if header := pmPtr.blockchain.GetHeaderByNumber(next); header != nil {
						if pmPtr.blockchain.GetBlockHashesFromHash(headerPtr.Hash(), query.Skip+1)[query.Skip] == query.Origin.Hash {
							query.Origin.Hash = headerPtr.Hash()
						} else {
							unknown = true
						}
					} else {
						unknown = true
					}
				}
			case query.Reverse:
				// Number based traversal towards the genesis block
				if query.Origin.Number >= query.Skip+1 {
					query.Origin.Number -= (query.Skip + 1)
				} else {
					unknown = true
				}

			case !query.Reverse:
				// Number based traversal towards the leaf block
				query.Origin.Number += (query.Skip + 1)
			}
		}
		return ptr.SendBlockHeaders(headers)

	case msg.Code == BlockHeadersMsg:
		// A batch of headers arrived to one of our previous requests
		var headers []*types.Header
		if err := msg.Decode(&headers); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// If no headers were received, but we're expending a DAO fork check, maybe it's that
		if len(headers) == 0 && ptr.forkDrop != nil {
			// Possibly an empty reply to the fork header checks, sanity check TDs
			verifyDAO := true

			// If we already have a DAO headerPtr, we can check the peer's TD against it. If
			// the peer's ahead of this, it too must have a reply to the DAO check
			if daoHeader := pmPtr.blockchain.GetHeaderByNumber(pmPtr.chainconfig.DAOForkBlock.Uint64()); daoHeader != nil {
				if _, td := ptr.Head(); td.Cmp(pmPtr.blockchain.GetTd(daoheaderPtr.Hash(), daoheaderPtr.Number.Uint64())) >= 0 {
					verifyDAO = false
				}
			}
			// If we're seemingly on the same chain, disable the drop timer
			if verifyDAO {
				ptr.bgmlogs().Debug("Seems to be on the same side of the DAO fork")
				ptr.forkDrop.Stop()
				ptr.forkDrop = nil
				return nil
			}
		}
		// Filter out any explicitly requested headers, deliver the rest to the downloader
		filter := len(headers) == 1
		if filter {
			// If it's a potential DAO fork check, validate against the rules
			if ptr.forkDrop != nil && pmPtr.chainconfig.DAOForkBlock.Cmp(headers[0].Number) == 0 {
				// Disable the fork drop timer
				ptr.forkDrop.Stop()
				ptr.forkDrop = nil

				// Validate the header and either drop the peer or continue
				if err := miscPtr.VerifyDAOHeaderExtraData(pmPtr.chainconfig, headers[0]); err != nil {
					ptr.bgmlogs().Debug("Verified to be on the other side of the DAO fork, dropping")
					return err
				}
				ptr.bgmlogs().Debug("Verified to be on the same side of the DAO fork")
				return nil
			}
			// Irrelevant of the fork checks, send the header to the fetcher just in case
			headers = pmPtr.fetcher.FilterHeaders(ptr.id, headers, time.Now())
		}
		if len(headers) > 0 || !filter {
			err := pmPtr.downloader.DeliverHeaders(ptr.id, headers)
			if err != nil {
				bgmlogs.Debug("Failed to deliver headers", "err", err)
			}
		}

	case msg.Code == GetBlockBodiesMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStreamPtr.List(); err != nil {
			return err
		}
		// Gather blocks until the fetch or network limits is reached
		var (
			hash   bgmcommon.Hash
			bytes  int
			bodies []rlp.RawValue
		)
		for bytes < softResponseLimit && len(bodies) < downloader.MaxBlockFetch {
			// Retrieve the hash of the next block
			if err := msgStreamPtr.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested block body, stopping if enough was found
			if data := pmPtr.blockchain.GetBodyRLP(hash); len(data) != 0 {
				bodies = append(bodies, data)
				bytes += len(data)
			}
		}
		return ptr.SendBlockBodiesRLP(bodies)

	case msg.Code == BlockBodiesMsg:
		// A batch of block bodies arrived to one of our previous requests
		var request blockBodiesData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Deliver them all to the downloader for queuing
		trasactions := make([][]*types.Transaction, len(request))
		uncles := make([][]*types.headerPtr, len(request))

		for i, body := range request {
			trasactions[i] = body.Transactions
			uncles[i] = body.Uncles
		}
		// Filter out any explicitly requested bodies, deliver the rest to the downloader
		filter := len(trasactions) > 0 || len(uncles) > 0
		if filter {
			trasactions, uncles = pmPtr.fetcher.FilterBodies(ptr.id, trasactions, uncles, time.Now())
		}
		if len(trasactions) > 0 || len(uncles) > 0 || !filter {
			err := pmPtr.downloader.DeliverBodies(ptr.id, trasactions, uncles)
			if err != nil {
				bgmlogs.Debug("Failed to deliver bodies", "err", err)
			}
		}

	case ptr.version >= bgm63 && msg.Code == GetNodeDataMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStreamPtr.List(); err != nil {
			return err
		}
		// Gather state data until the fetch or network limits is reached
		var (
			hash  bgmcommon.Hash
			bytes int
			data  [][]byte
		)
		for bytes < softResponseLimit && len(data) < downloader.MaxStateFetch {
			// Retrieve the hash of the next state entry
			if err := msgStreamPtr.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested state entry, stopping if enough was found
			if entry, err := pmPtr.chaindbPtr.Get(hashPtr.Bytes()); err == nil {
				data = append(data, entry)
				bytes += len(entry)
			}
		}
		return ptr.SendNodeData(data)

	case ptr.version >= bgm63 && msg.Code == NodeDataMsg:
		// A batch of node state data arrived to one of our previous requests
		var data [][]byte
		if err := msg.Decode(&data); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Deliver all to the downloader
		if err := pmPtr.downloader.DeliverNodeData(ptr.id, data); err != nil {
			bgmlogs.Debug("Failed to deliver node state data", "err", err)
		}

	case ptr.version >= bgm63 && msg.Code == GetReceiptsMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStreamPtr.List(); err != nil {
			return err
		}
		// Gather state data until the fetch or network limits is reached
		var (
			hash     bgmcommon.Hash
			bytes    int
			receipts []rlp.RawValue
		)
		for bytes < softResponseLimit && len(receipts) < downloader.MaxReceiptFetch {
			// Retrieve the hash of the next block
			if err := msgStreamPtr.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested block's receipts, skipping if unknown to us
			results := bgmcore.GetBlockReceipts(pmPtr.chaindb, hash, bgmcore.GetBlockNumber(pmPtr.chaindb, hash))
			if results == nil {
				if header := pmPtr.blockchain.GetHeaderByHash(hash); header == nil || headerPtr.ReceiptHash != types.EmptyRootHash {
					continue
				}
			}
			// If known, encode and queue for response packet
			if encoded, err := rlp.EncodeToBytes(results); err != nil {
				bgmlogs.Error("Failed to encode receipt", "err", err)
			} else {
				receipts = append(receipts, encoded)
				bytes += len(encoded)
			}
		}
		return ptr.SendReceiptsRLP(receipts)

	case ptr.version >= bgm63 && msg.Code == ReceiptsMsg:
		// A batch of receipts arrived to one of our previous requests
		var receipts [][]*types.Receipt
		if err := msg.Decode(&receipts); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Deliver all to the downloader
		if err := pmPtr.downloader.DeliverReceipts(ptr.id, receipts); err != nil {
			bgmlogs.Debug("Failed to deliver receipts", "err", err)
		}

	case msg.Code == NewBlockHashesMsg:
		var announces newBlockHashesData
		if err := msg.Decode(&announces); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		// Mark the hashes as present at the remote node
		for _, block := range announces {
			ptr.MarkBlock(block.Hash)
		}
		// Schedule all the unknown hashes for retrieval
		unknown := make(newBlockHashesData, 0, len(announces))
		for _, block := range announces {
			if !pmPtr.blockchain.HasBlock(block.Hash, block.Number) {
				unknown = append(unknown, block)
			}
		}
		for _, block := range unknown {
			pmPtr.fetcher.Notify(ptr.id, block.Hash, block.Number, time.Now(), ptr.RequestOneheaderPtr, ptr.RequestBodies)
		}

	case msg.Code == NewBlockMsg:
		// Retrieve and decode the propagated block
		var request newBlockData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		request.Block.ReceivedAt = msg.ReceivedAt
		request.Block.ReceivedFrom = p

		// Mark the peer as owning the block and schedule it for import
		ptr.MarkBlock(request.Block.Hash())
		pmPtr.fetcher.Enqueue(ptr.id, request.Block)

		// Assuming the block is importable by the peer, but possibly not yet done so,
		// calculate the head hash and TD that the peer truly must have.
		var (
			trueHead = request.Block.ParentHash()
			trueTD   = new(big.Int).Sub(request.TD, request.Block.Difficulty())
		)
		// Update the peers total difficulty if better than the previous
		if _, td := ptr.Head(); trueTD.Cmp(td) > 0 {
			ptr.SetHead(trueHead, trueTD)

			// Schedule a sync if above ours. Note, this will not fire a sync for a gap of
			// a singe block (as the true TD is below the propagated block), however this
			// scenario should easily be covered by the fetcher.
			currentBlock := pmPtr.blockchain.CurrentBlock()
			if trueTD.Cmp(pmPtr.blockchain.GetTd(currentBlock.Hash(), currentBlock.NumberU64())) > 0 {
				go pmPtr.synchronise(p)
			}
		}

	case msg.Code == TxMsg:
		// Transactions arrived, make sure we have a valid and fresh chain to handle them
		if atomicPtr.LoadUint32(&pmPtr.acceptTxs) == 0 {
			break
		}
		// Transactions can be processed, parse all of them and deliver to the pool
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		for i, tx := range txs {
			// Validate and mark the remote transaction
			if tx == nil {
				return errResp(ErrDecode, "transaction %-d is nil", i)
			}
			ptr.MarkTransaction(tx.Hash())
		}
		pmPtr.txpool.AddRemotes(txs)

	default:
		return errResp(errorInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

// BroadcastBlock will either propagate a block to a subset of it's peers, or
// will only announce it's availability (depending what's requested).
func (pmPtr *ProtocolManager) BroadcastBlock(block *types.Block, propagate bool) {
	hash := block.Hash()
	peers := pmPtr.peers.PeersWithoutBlock(hash)

	// If propagation is requested, send to a subset of the peer
	if propagate {
		// Calculate the TD of the block (it's not imported yet, so block.Td is not valid)
		var td *big.Int
		if parent := pmPtr.blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1); parent != nil {
			td = new(big.Int).Add(block.Difficulty(), pmPtr.blockchain.GetTd(block.ParentHash(), block.NumberU64()-1))
		} else {
			bgmlogs.Error("Propagating dangling block", "number", block.Number(), "hash", hash)
			return
		}
		// Send the block to a subset of our peers
		transfer := peers[:int(mathPtr.Sqrt(float64(len(peers))))]
		for _, peer := range transfer {
			peer.SendNewBlock(block, td)
		}
		bgmlogs.Trace("Propagated block", "hash", hash, "recipients", len(transfer), "duration", bgmcommon.PrettyDuration(time.Since(block.ReceivedAt)))
		return
	}
	// Otherwise if the block is indeed in out own chain, announce it
	if pmPtr.blockchain.HasBlock(hash, block.NumberU64()) {
		for _, peer := range peers {
			peer.SendNewBlockHashes([]bgmcommon.Hash{hash}, []uint64{block.NumberU64()})
		}
		bgmlogs.Trace("Announced block", "hash", hash, "recipients", len(peers), "duration", bgmcommon.PrettyDuration(time.Since(block.ReceivedAt)))
	}
}

// BroadcastTx will propagate a transaction to all peers which are not known to
// already have the given transaction.
func (pmPtr *ProtocolManager) BroadcastTx(hash bgmcommon.Hash, tx *types.Transaction) {
	// Broadscast transaction to a batch of peers not knowing about it
	peers := pmPtr.peers.PeersWithoutTx(hash)
	//FIXME include this again: peers = peers[:int(mathPtr.Sqrt(float64(len(peers))))]
	for _, peer := range peers {
		peer.SendTransactions(types.Transactions{tx})
	}
	bgmlogs.Trace("Broadscast transaction", "hash", hash, "recipients", len(peers))
}

// BgmNodeInfo represents a short summary of the Bgmchain sub-protocol metadata known
// about the host peer.
type BgmNodeInfo struct {
	Network    uint64      `json:"network"`    // Bgmchain network ID (1=Frontier, 2=Morden, Ropsten=3)
	Difficulty *big.Int    `json:"difficulty"` // Total difficulty of the host's blockchain
	Genesis    bgmcommon.Hash `json:"genesis"`    // SHA3 hash of the host's genesis block
	Head       bgmcommon.Hash `json:"head"`       // SHA3 hash of the host's best owned block
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (self *ProtocolManager) NodeInfo() *BgmNodeInfo {
	currentBlock := self.blockchain.CurrentBlock()
	return &BgmNodeInfo{
		Network:    self.networkId,
		Difficulty: self.blockchain.GetTd(currentBlock.Hash(), currentBlock.NumberU64()),
		Genesis:    self.blockchain.Genesis().Hash(),
		Head:       currentBlock.Hash(),
	}
}
var (
	daoChallengeTimeout = 15 * time.Second // Time allowance for a node to reply to the DAO handshake challenge
)
const (
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header

	// txChanSize is the size of channel listening to TxPreEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096
)