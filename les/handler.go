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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgm"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/light"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/p2p/discv5"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/trie"
)

const (
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, Headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block Header

	bgmVersion = 63 // equivalent bgm version for the downloader

	MaxHeaderFetch           = 192 // Amount of block Headers to be fetched per retrieval request
	MaxBodyFetch             = 32  // Amount of block bodies to be fetched per retrieval request
	MaxRecChaintFetch          = 128 // Amount of transaction recChaints to allow fetching per request
	MaxCodeFetch             = 64  // Amount of contract codes to allow fetching per request
	MaxProofsFetch           = 64  // Amount of merkle proofs to be fetched per retrieval request
	MaxHelperTrieProofsFetch = 64  // Amount of merkle proofs to be fetched per retrieval request
	MaxTxSend                = 64  // Amount of transactions to be send per request
	MaxTxStatus              = 256 // Amount of transactions to queried per request

	disableClientRemovePeer = false
)

// errNotcompatibleConfig is returned if the requested protocols and configs are
// not compatible (low protocol version restrictions and high requirements).
var errNotcompatibleConfig = errors.New("Notcompatible configuration")

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type BlockChain interface {
	HasHeader(hash bgmcommon.Hash, number Uint64) bool
	GetHeader(hash bgmcommon.Hash, number Uint64) *types.Header
	GetHeaderByHash(hash bgmcommon.Hash) *types.Header
	CurrentHeader() *types.Header
	GetTdByHash(hash bgmcommon.Hash) *big.Int
	InsertHeaderChain(chain []*types.HeaderPtr, checkFreq int) (int, error)
	Rollback(chain []bgmcommon.Hash)
	Status() (td *big.Int, currentBlock bgmcommon.Hash, genesisBlock bgmcommon.Hash)
	GetHeaderByNumber(number Uint64) *types.Header
	GethashesFromHash(hash bgmcommon.Hash, max Uint64) []bgmcommon.Hash
	Lasthash() bgmcommon.Hash
	Genesis() *types.Block
	SubscribeChainHeadEvent(ch chan<- bgmCore.ChainHeadEvent) event.Subscription
}

type txPool interface {
	AddRemotes(txs []*types.Transaction) []error
	Status(hashes []bgmcommon.Hash) []bgmCore.TxStatus
}

type ProtocolManager struct {
	lightSync   bool
	txpool      txPool
	txrelay     *LesTxRelay
	networkId   Uint64
	chainConfig *bgmparam.ChainConfig
	blockchain  BlockChain
	chainDb     bgmdbPtr.Database
	odr         *LesOdr
	server      *LesServer
	serverPool  *serverPool
	lesTopic    discv5.Topic
	reqDist     *requestDistributor
	retriever   *retrieveManager

	downloader *downloader.Downloader
	fetcher    *lightFetcher
	peers      *peerSet

	SubProtocols []p2p.Protocol

	eventMux *event.TypeMux

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh   chan *peer
	quitSync    chan struct{}
	noMorePeers chan struct{}

	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg *syncPtr.WaitGroup
}

// NewProtocolManager returns a new bgmchain sub protocol manager. The Bgmchain sub protocol manages peers capable
// with the bgmchain network.
func NewProtocolManager(chainConfig *bgmparam.ChainConfig, lightSync bool, protocolVersions []uint, networkId Uint64, mux *event.TypeMux, engine consensus.Engine, peers *peerSet, blockchain BlockChain, txpool txPool, chainDb bgmdbPtr.Database, odr *LesOdr, txrelay *LesTxRelay, quitSync chan struct{}, wg *syncPtr.WaitGroup) (*ProtocolManager, error) {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		lightSync:   lightSync,
		eventMux:    mux,
		blockchain:  blockchain,
		chainConfig: chainConfig,
		chainDb:     chainDb,
		odr:         odr,
		networkId:   networkId,
		txpool:      txpool,
		txrelay:     txrelay,
		peers:       peers,
		newPeerCh:   make(chan *peer),
		quitSync:    quitSync,
		wg:          wg,
		noMorePeers: make(chan struct{}),
	}
	if odr != nil {
		manager.retriever = odr.retriever
		manager.reqDist = odr.retriever.dist
	}

	// Initiate a sub-protocol for every implemented version we can handle
	manager.SubProtocols = make([]p2p.Protocol, 0, len(protocolVersions))
	for _, version := range protocolVersions {
		// Compatible, initialize the sub-protocol
		version := version // Closure for the run
		manager.SubProtocols = append(manager.SubProtocols, p2p.Protocol{
			Name:    "les",
			Version: version,
			Length:  ProtocolLengths[version],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				var entry *poolEntry
				peer := manager.newPeer(int(version), networkId, p, rw)
				if manager.serverPool != nil {
					addr := p.RemoteAddr().(*net.TCPAddr)
					entry = manager.serverPool.connect(peer, addr.IP, uint16(addr.Port))
				}
				peer.poolEntry = entry
				select {
				case manager.newPeerCh <- peer:
					manager.wg.Add(1)
					defer manager.wg.Done()
					err := manager.handle(peer)
					if entry != nil {
						manager.serverPool.disconnect(entry)
					}
					return err
				case <-manager.quitSync:
					if entry != nil {
						manager.serverPool.disconnect(entry)
					}
					return p2p.DiscQuitting
				}
			},
			NodeInfo: func() interface{} {
				return manager.NodeInfo()
			},
			PeerInfo: func(id discover.NodeID) interface{} {
				if p := manager.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
					return p.Info()
				}
				return nil
			},
		})
	}
	if len(manager.SubProtocols) == 0 {
		return nil, errNotcompatibleConfig
	}

	removePeer := manager.removePeer
	if disableClientRemovePeer {
		removePeer = func(id string) {}
	}

	if lightSync {
		manager.downloader = downloader.New(downloader.LightSync, chainDb, manager.eventMux, nil, blockchain, removePeer)
		manager.peers.notify((*downloaderPeerNotify)(manager))
		manager.fetcher = newLightFetcher(manager)
	}

	return manager, nil
}

// removePeer initiates disconnection from a peer by removing it from the peer set
func (pmPtr *ProtocolManager) removePeer(id string) {
	pmPtr.peers.Unregister(id)
}

func (pmPtr *ProtocolManager) Start() {
	if pmPtr.lightSync {
		go pmPtr.syncer()
	} else {
		go func() {
			for range pmPtr.newPeerCh {
			}
		}()
	}
}

func (pmPtr *ProtocolManager) Stop() {
	// Showing a bgmlogs message. During download / process this could actually
	// take between 5 to 10 seconds and therefor feedback is required.
	bgmlogs.Info("Stopping light Bgmchain protocol")

	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	pmPtr.noMorePeers <- struct{}{}

	close(pmPtr.quitSync) // quits syncer, fetcher

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to pmPtr.peers yet
	// will exit when they try to register.
	pmPtr.peers.Close()

	// Wait for any process action
	pmPtr.wg.Wait()

	bgmlogs.Info("Light Bgmchain protocol stopped")
}

func (pmPtr *ProtocolManager) newPeer(pv int, nv Uint64, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, nv, p, newMeteredMsgWriter(rw))
}

// handle is the callback invoked to manage the life cycle of a les peer. When
// this function terminates, the peer is disconnected.
func (pmPtr *ProtocolManager) handle(p *peer) error {
	p.bgmlogs().Debug("Light Bgmchain peer connected", "name", p.Name())

	// Execute the LES handshake
	td, head, genesis := pmPtr.blockchain.Status()
	headNum := bgmCore.Getnumber(pmPtr.chainDb, head)
	if err := p.Handshake(td, head, headNum, genesis, pmPtr.server); err != nil {
		p.bgmlogs().Debug("Light Bgmchain handshake failed", "err", err)
		return err
	}
	if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	}
	// Register the peer locally
	if err := pmPtr.peers.Register(p); err != nil {
		p.bgmlogs().Error("Light Bgmchain peer registration failed", "err", err)
		return err
	}
	defer func() {
		if pmPtr.server != nil && pmPtr.server.fcManager != nil && p.fcClient != nil {
			p.fcClient.Remove(pmPtr.server.fcManager)
		}
		pmPtr.removePeer(p.id)
	}()
	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if pmPtr.lightSync {
		p.lock.Lock()
		head := p.headInfo
		p.lock.Unlock()
		if pmPtr.fetcher != nil {
			pmPtr.fetcher.announce(p, head)
		}

		if p.poolEntry != nil {
			pmPtr.serverPool.registered(p.poolEntry)
		}
	}

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		// new block announce loop
		for {
			select {
			case announce := <-p.announceChn:
				p.SendAnnounce(announce)
			case <-stop:
				return
			}
		}
	}()

	// main loop. handle incoming messages.
	for {
		if err := pmPtr.handleMessage(p); err != nil {
			p.bgmlogs().Debug("Light Bgmchain message handling failed", "err", err)
			return err
		}
	}
}

var reqList = []Uint64{GetBlockHeadersMsg, GetBlockBodiesMsg, GetCodeMsg, GetRecChaintsMsg, GetProofsV1Msg, SendTxMsg, SendTxV2Msg, GetTxStatusMsg, GetHeaderProofsMsg, GetProofsV2Msg, GetHelperTrieProofsMsg}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (pmPtr *ProtocolManager) handleMessage(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMessage()
	if err != nil {
		return err
	}
	p.bgmlogs().Trace("Light Bgmchain message arrived", "code", msg.Code, "bytes", msg.Size)

	costs := p.fcCosts[msg.Code]
	reject := func(reqCnt, maxCnt Uint64) bool {
		if p.fcClient == nil || reqCnt > maxCnt {
			return true
		}
		bufValue, _ := p.fcClient.AcceptRequest()
		cost := costs.baseCost + reqCnt*costs.reqCost
		if cost > pmPtr.server.defbgmparam.BufLimit {
			cost = pmPtr.server.defbgmparam.BufLimit
		}
		if cost > bufValue {
			recharge := time.Duration((cost - bufValue) * 1000000 / pmPtr.server.defbgmparam.MinRecharge)
			p.bgmlogs().Error("Request came too early", "recharge", bgmcommon.PrettyDuration(recharge))
			return true
		}
		return false
	}

	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	var deliverMsg *Msg

	// Handle the message depending on its contents
	switch msg.Code {
	case StatusMsg:
		p.bgmlogs().Trace("Received status message")
		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	// Block Header query, collect the requested Headers and reply
	case AnnounceMsg:
		p.bgmlogs().Trace("Received announce message")
		if p.requestAnnounceType == announceTypeNone {
			return errResp(ErrUnexpectedResponse, "")
		}

		var req announceData
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}

		if p.requestAnnounceType == announceTypeSigned {
			if err := req.checkSignature(p.pubKey); err != nil {
				p.bgmlogs().Trace("Invalid announcement signature", "err", err)
				return err
			}
			p.bgmlogs().Trace("Valid announcement signature")
		}

		p.bgmlogs().Trace("Announce message content", "number", req.Number, "hash", req.Hash, "td", req.Td, "reorg", req.ReorgDepth)
		if pmPtr.fetcher != nil {
			pmPtr.fetcher.announce(p, &req)
		}

	case GetBlockHeadersMsg:
		p.bgmlogs().Trace("Received block Header request")
		// Decode the complex Header query
		var req struct {
			ReqID Uint64
			Query getBlockHeadersData
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}

		query := req.Query
		if reject(query.Amount, MaxHeaderFetch) {
			return errResp(ErrRequestRejected, "")
		}

		hashMode := query.Origin.Hash != (bgmcommon.Hash{})

		// Gather Headers until the fetch or network limits is reached
		var (
			bytes   bgmcommon.StorageSize
			Headers []*types.Header
			unknown bool
		)
		for !unknown && len(Headers) < int(query.Amount) && bytes < softResponseLimit {
			// Retrieve the next Header satisfying the query
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
			Headers = append(Headers, origin)
			bytes += estHeaderRlpSize

			// Advance to the next Header of the query
			switch {
			case query.Origin.Hash != (bgmcommon.Hash{}) && query.Reverse:
				// Hash based traversal towards the genesis block
				for i := 0; i < int(query.Skip)+1; i++ {
					if Header := pmPtr.blockchain.GetHeader(query.Origin.Hash, number); Header != nil {
						query.Origin.Hash = HeaderPtr.ParentHash
						number--
					} else {
						unknown = true
						break
					}
				}
			case query.Origin.Hash != (bgmcommon.Hash{}) && !query.Reverse:
				// Hash based traversal towards the leaf block
				if Header := pmPtr.blockchain.GetHeaderByNumber(origin.Number.Uint64() + query.Skip + 1); Header != nil {
					if pmPtr.blockchain.GethashesFromHash(HeaderPtr.Hash(), query.Skip+1)[query.Skip] == query.Origin.Hash {
						query.Origin.Hash = HeaderPtr.Hash()
					} else {
						unknown = true
					}
				} else {
					unknown = true
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

		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + query.Amount*costs.reqCost)
		pmPtr.server.fcCostStats.update(msg.Code, query.Amount, rcost)
		return p.SendBlockHeaders(req.ReqID, bv, Headers)

	case BlockHeadersMsg:
		if pmPtr.downloader == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.bgmlogs().Trace("Received block Header response message")
		// A batch of Headers arrived to one of our previous requests
		var resp struct {
			ReqID, BV Uint64
			Headers   []*types.Header
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.GotReply(resp.ReqID, resp.BV)
		if pmPtr.fetcher != nil && pmPtr.fetcher.requestedID(resp.ReqID) {
			pmPtr.fetcher.deliverHeaders(p, resp.ReqID, resp.Headers)
		} else {
			err := pmPtr.downloader.DeliverHeaders(p.id, resp.Headers)
			if err != nil {
				bgmlogs.Debug(fmt.Sprint(err))
			}
		}

	case GetBlockBodiesMsg:
		p.bgmlogs().Trace("Received block bodies request")
		// Decode the retrieval message
		var req struct {
			ReqID  Uint64
			Hashes []bgmcommon.Hash
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather blocks until the fetch or network limits is reached
		var (
			bytes  int
			bodies []rlp.RawValue
		)
		reqCnt := len(req.Hashes)
		if reject(Uint64(reqCnt), MaxBodyFetch) {
			return errResp(ErrRequestRejected, "")
		}
		for _, hash := range req.Hashes {
			if bytes >= softResponseLimit {
				break
			}
			// Retrieve the requested block body, stopping if enough was found
			if data := bgmCore.GetBodyRLP(pmPtr.chainDb, hash, bgmCore.Getnumber(pmPtr.chainDb, hash)); len(data) != 0 {
				bodies = append(bodies, data)
				bytes += len(data)
			}
		}
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + Uint64(reqCnt)*costs.reqCost)
		pmPtr.server.fcCostStats.update(msg.Code, Uint64(reqCnt), rcost)
		return p.SendBlockBodiesRLP(req.ReqID, bv, bodies)

	case BlockBodiesMsg:
		if pmPtr.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.bgmlogs().Trace("Received block bodies response")
		// A batch of block bodies arrived to one of our previous requests
		var resp struct {
			ReqID, BV Uint64
			Data      []*types.Body
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgBlockBodies,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case GetCodeMsg:
		p.bgmlogs().Trace("Received code request")
		// Decode the retrieval message
		var req struct {
			ReqID Uint64
			Reqs  []CodeReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			bytes int
			data  [][]byte
		)
		reqCnt := len(req.Reqs)
		if reject(Uint64(reqCnt), MaxCodeFetch) {
			return errResp(ErrRequestRejected, "")
		}
		for _, req := range req.Reqs {
			// Retrieve the requested state entry, stopping if enough was found
			if Header := bgmCore.GetHeader(pmPtr.chainDb, req.BHash, bgmCore.Getnumber(pmPtr.chainDb, req.BHash)); Header != nil {
				if trie, _ := trie.New(HeaderPtr.Root, pmPtr.chainDb); trie != nil {
					sdata := trie.Get(req.AccKey)
					var acc state.Account
					if err := rlp.DecodeBytes(sdata, &acc); err == nil {
						entry, _ := pmPtr.chainDbPtr.Get(accPtr.CodeHash)
						if bytes+len(entry) >= softResponseLimit {
							break
						}
						data = append(data, entry)
						bytes += len(entry)
					}
				}
			}
		}
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + Uint64(reqCnt)*costs.reqCost)
		pmPtr.server.fcCostStats.update(msg.Code, Uint64(reqCnt), rcost)
		return p.SendCode(req.ReqID, bv, data)

	case CodeMsg:
		if pmPtr.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.bgmlogs().Trace("Received code response")
		// A batch of node state data arrived to one of our previous requests
		var resp struct {
			ReqID, BV Uint64
			Data      [][]byte
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgCode,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case GetRecChaintsMsg:
		p.bgmlogs().Trace("Received recChaints request")
		// Decode the retrieval message
		var req struct {
			ReqID  Uint64
			Hashes []bgmcommon.Hash
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			bytes    int
			recChaints []rlp.RawValue
		)
		reqCnt := len(req.Hashes)
		if reject(Uint64(reqCnt), MaxRecChaintFetch) {
			return errResp(ErrRequestRejected, "")
		}
		for _, hash := range req.Hashes {
			if bytes >= softResponseLimit {
				break
			}
			// Retrieve the requested block's recChaints, skipping if unknown to us
			results := bgmCore.GetBlockRecChaints(pmPtr.chainDb, hash, bgmCore.Getnumber(pmPtr.chainDb, hash))
			if results == nil {
				if Header := pmPtr.blockchain.GetHeaderByHash(hash); Header == nil || HeaderPtr.RecChaintHash != types.EmptyRootHash {
					continue
				}
			}
			// If known, encode and queue for response packet
			if encoded, err := rlp.EncodeToBytes(results); err != nil {
				bgmlogs.Error("Failed to encode recChaint", "err", err)
			} else {
				recChaints = append(recChaints, encoded)
				bytes += len(encoded)
			}
		}
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + Uint64(reqCnt)*costs.reqCost)
		pmPtr.server.fcCostStats.update(msg.Code, Uint64(reqCnt), rcost)
		return p.SendRecChaintsRLP(req.ReqID, bv, recChaints)

	case RecChaintsMsg:
		if pmPtr.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.bgmlogs().Trace("Received recChaints response")
		// A batch of recChaints arrived to one of our previous requests
		var resp struct {
			ReqID, BV Uint64
			RecChaints  []types.RecChaints
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgRecChaints,
			ReqID:   resp.ReqID,
			Obj:     resp.RecChaints,
		}

	case GetProofsV1Msg:
		p.bgmlogs().Trace("Received proofs request")
		// Decode the retrieval message
		var req struct {
			ReqID Uint64
			Reqs  []ProofReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			bytes  int
			proofs proofsData
		)
		reqCnt := len(req.Reqs)
		if reject(Uint64(reqCnt), MaxProofsFetch) {
			return errResp(ErrRequestRejected, "")
		}
		for _, req := range req.Reqs {
			if bytes >= softResponseLimit {
				break
			}
			// Retrieve the requested state entry, stopping if enough was found
			if Header := bgmCore.GetHeader(pmPtr.chainDb, req.BHash, bgmCore.Getnumber(pmPtr.chainDb, req.BHash)); Header != nil {
				if tr, _ := trie.New(HeaderPtr.Root, pmPtr.chainDb); tr != nil {
					if len(req.AccKey) > 0 {
						sdata := tr.Get(req.AccKey)
						tr = nil
						var acc state.Account
						if err := rlp.DecodeBytes(sdata, &acc); err == nil {
							tr, _ = trie.New(accPtr.Root, pmPtr.chainDb)
						}
					}
					if tr != nil {
						var proof light.NodeList
						tr.Prove(req.Key, 0, &proof)
						proofs = append(proofs, proof)
						bytes += proof.DataSize()
					}
				}
			}
		}
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + Uint64(reqCnt)*costs.reqCost)
		pmPtr.server.fcCostStats.update(msg.Code, Uint64(reqCnt), rcost)
		return p.SendProofs(req.ReqID, bv, proofs)

	case GetProofsV2Msg:
		p.bgmlogs().Trace("Received les/2 proofs request")
		// Decode the retrieval message
		var req struct {
			ReqID Uint64
			Reqs  []ProofReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			lastBHash  bgmcommon.Hash
			lastAccKey []byte
			tr, str    *trie.Trie
		)
		reqCnt := len(req.Reqs)
		if reject(Uint64(reqCnt), MaxProofsFetch) {
			return errResp(ErrRequestRejected, "")
		}

		nodes := light.NewNodeSet()

		for _, req := range req.Reqs {
			if nodes.DataSize() >= softResponseLimit {
				break
			}
			if tr == nil || req.BHash != lastBHash {
				if Header := bgmCore.GetHeader(pmPtr.chainDb, req.BHash, bgmCore.Getnumber(pmPtr.chainDb, req.BHash)); Header != nil {
					tr, _ = trie.New(HeaderPtr.Root, pmPtr.chainDb)
				} else {
					tr = nil
				}
				lastBHash = req.BHash
				str = nil
			}
			if tr != nil {
				if len(req.AccKey) > 0 {
					if str == nil || !bytes.Equal(req.AccKey, lastAccKey) {
						sdata := tr.Get(req.AccKey)
						str = nil
						var acc state.Account
						if err := rlp.DecodeBytes(sdata, &acc); err == nil {
							str, _ = trie.New(accPtr.Root, pmPtr.chainDb)
						}
						lastAccKey = bgmcommon.CopyBytes(req.AccKey)
					}
					if str != nil {
						str.Prove(req.Key, req.FromLevel, nodes)
					}
				} else {
					tr.Prove(req.Key, req.FromLevel, nodes)
				}
			}
		}
		proofs := nodes.NodeList()
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + Uint64(reqCnt)*costs.reqCost)
		pmPtr.server.fcCostStats.update(msg.Code, Uint64(reqCnt), rcost)
		return p.SendProofsV2(req.ReqID, bv, proofs)

	case ProofsV1Msg:
		if pmPtr.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.bgmlogs().Trace("Received proofs response")
		// A batch of merkle proofs arrived to one of our previous requests
		var resp struct {
			ReqID, BV Uint64
			Data      []light.NodeList
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgProofsV1,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case ProofsV2Msg:
		if pmPtr.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.bgmlogs().Trace("Received les/2 proofs response")
		// A batch of merkle proofs arrived to one of our previous requests
		var resp struct {
			ReqID, BV Uint64
			Data      light.NodeList
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgProofsV2,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case GetHeaderProofsMsg:
		p.bgmlogs().Trace("Received Headers proof request")
		// Decode the retrieval message
		var req struct {
			ReqID Uint64
			Reqs  []ChtReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			bytes  int
			proofs []ChtResp
		)
		reqCnt := len(req.Reqs)
		if reject(Uint64(reqCnt), MaxHelperTrieProofsFetch) {
			return errResp(ErrRequestRejected, "")
		}
		trieDb := bgmdbPtr.NewTable(pmPtr.chainDb, light.ChtTablePrefix)
		for _, req := range req.Reqs {
			if bytes >= softResponseLimit {
				break
			}

			if Header := pmPtr.blockchain.GetHeaderByNumber(req.BlockNum); Header != nil {
				sectionHead := bgmCore.GetCanonicalHash(pmPtr.chainDb, (req.ChtNum+1)*light.ChtV1Frequency-1)
				if blockRoot := light.GetChtRoot(pmPtr.chainDb, req.ChtNum, sectionHead); blockRoot != (bgmcommon.Hash{}) {
					if tr, _ := trie.New(blockRoot, trieDb); tr != nil {
						var encNumber [8]byte
						binary.BigEndian.PutUint64(encNumber[:], req.BlockNum)
						var proof light.NodeList
						tr.Prove(encNumber[:], 0, &proof)
						proofs = append(proofs, ChtResp{Header: HeaderPtr, Proof: proof})
						bytes += proof.DataSize() + estHeaderRlpSize
					}
				}
			}
		}
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + Uint64(reqCnt)*costs.reqCost)
		pmPtr.server.fcCostStats.update(msg.Code, Uint64(reqCnt), rcost)
		return p.SendHeaderProofs(req.ReqID, bv, proofs)

	case GetHelperTrieProofsMsg:
		p.bgmlogs().Trace("Received helper trie proof request")
		// Decode the retrieval message
		var req struct {
			ReqID Uint64
			Reqs  []HelperTrieReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			auxBytes int
			auxData  [][]byte
		)
		reqCnt := len(req.Reqs)
		if reject(Uint64(reqCnt), MaxHelperTrieProofsFetch) {
			return errResp(ErrRequestRejected, "")
		}

		var (
			lastIdx  Uint64
			lastType uint
			blockRoot     bgmcommon.Hash
			tr       *trie.Trie
		)

		nodes := light.NewNodeSet()

		for _, req := range req.Reqs {
			if nodes.DataSize()+auxBytes >= softResponseLimit {
				break
			}
			if tr == nil || req.HelperTrieType != lastType || req.TrieIdx != lastIdx {
				var prefix string
				blockRoot, prefix = pmPtr.getHelperTrie(req.HelperTrieType, req.TrieIdx)
				if blockRoot != (bgmcommon.Hash{}) {
					if t, err := trie.New(blockRoot, bgmdbPtr.NewTable(pmPtr.chainDb, prefix)); err == nil {
						tr = t
					}
				}
				lastType = req.HelperTrieType
				lastIdx = req.TrieIdx
			}
			if req.AuxReq == auxRoot {
				var data []byte
				if blockRoot != (bgmcommon.Hash{}) {
					data = blockRoot[:]
				}
				auxData = append(auxData, data)
				auxBytes += len(data)
			} else {
				if tr != nil {
					tr.Prove(req.Key, req.FromLevel, nodes)
				}
				if req.AuxReq != 0 {
					data := pmPtr.getHelperTrieAuxData(req)
					auxData = append(auxData, data)
					auxBytes += len(data)
				}
			}
		}
		proofs := nodes.NodeList()
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + Uint64(reqCnt)*costs.reqCost)
		pmPtr.server.fcCostStats.update(msg.Code, Uint64(reqCnt), rcost)
		return p.SendHelperTrieProofs(req.ReqID, bv, HelperTrieResps{Proofs: proofs, AuxData: auxData})

	case HeaderProofsMsg:
		if pmPtr.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.bgmlogs().Trace("Received Headers proof response")
		var resp struct {
			ReqID, BV Uint64
			Data      []ChtResp
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgHeaderProofs,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case HelperTrieProofsMsg:
		if pmPtr.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.bgmlogs().Trace("Received helper trie proof response")
		var resp struct {
			ReqID, BV Uint64
			Data      HelperTrieResps
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgHelperTrieProofs,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case SendTxMsg:
		if pmPtr.txpool == nil {
			return errResp(ErrRequestRejected, "")
		}
		// Transactions arrived, parse all of them and deliver to the pool
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		reqCnt := len(txs)
		if reject(Uint64(reqCnt), MaxTxSend) {
			return errResp(ErrRequestRejected, "")
		}
		pmPtr.txpool.AddRemotes(txs)

		_, rcost := p.fcClient.RequestProcessed(costs.baseCost + Uint64(reqCnt)*costs.reqCost)
		pmPtr.server.fcCostStats.update(msg.Code, Uint64(reqCnt), rcost)

	case SendTxV2Msg:
		if pmPtr.txpool == nil {
			return errResp(ErrRequestRejected, "")
		}
		// Transactions arrived, parse all of them and deliver to the pool
		var req struct {
			ReqID Uint64
			Txs   []*types.Transaction
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		reqCnt := len(req.Txs)
		if reject(Uint64(reqCnt), MaxTxSend) {
			return errResp(ErrRequestRejected, "")
		}

		hashes := make([]bgmcommon.Hash, len(req.Txs))
		for i, tx := range req.Txs {
			hashes[i] = tx.Hash()
		}
		stats := pmPtr.txStatus(hashes)
		for i, stat := range stats {
			if stat.Status == bgmCore.TxStatusUnknown {
				if errs := pmPtr.txpool.AddRemotes([]*types.Transaction{req.Txs[i]}); errs[0] != nil {
					stats[i].Error = errs[0]
					continue
				}
				stats[i] = pmPtr.txStatus([]bgmcommon.Hash{hashes[i]})[0]
			}
		}

		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + Uint64(reqCnt)*costs.reqCost)
		pmPtr.server.fcCostStats.update(msg.Code, Uint64(reqCnt), rcost)

		return p.SendTxStatus(req.ReqID, bv, stats)

	case GetTxStatusMsg:
		if pmPtr.txpool == nil {
			return errResp(ErrUnexpectedResponse, "")
		}
		// Transactions arrived, parse all of them and deliver to the pool
		var req struct {
			ReqID  Uint64
			Hashes []bgmcommon.Hash
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		reqCnt := len(req.Hashes)
		if reject(Uint64(reqCnt), MaxTxStatus) {
			return errResp(ErrRequestRejected, "")
		}
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + Uint64(reqCnt)*costs.reqCost)
		pmPtr.server.fcCostStats.update(msg.Code, Uint64(reqCnt), rcost)

		return p.SendTxStatus(req.ReqID, bv, pmPtr.txStatus(req.Hashes))

	case TxStatusMsg:
		if pmPtr.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.bgmlogs().Trace("Received tx status response")
		var resp struct {
			ReqID, BV Uint64
			Status    []bgmCore.TxStatus
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		p.fcServer.GotReply(resp.ReqID, resp.BV)

	default:
		p.bgmlogs().Trace("Received unknown message", "code", msg.Code)
		return errResp(errorInvalidMsgCode, "%v", msg.Code)
	}

	if deliverMsg != nil {
		err := pmPtr.retriever.deliver(p, deliverMsg)
		if err != nil {
			p.responseErrors++
			if p.responseErrors > maxResponseErrors {
				return err
			}
		}
	}
	return nil
}

// getHelperTrie returns the post-processed trie blockRoot for the given trie ID and section index
func (pmPtr *ProtocolManager) getHelperTrie(id uint, idx Uint64) (bgmcommon.Hash, string) {
	switch id {
	case htCanonical:
		sectionHead := bgmCore.GetCanonicalHash(pmPtr.chainDb, (idx+1)*light.ChtFrequency-1)
		return light.GetChtV2Root(pmPtr.chainDb, idx, sectionHead), light.ChtTablePrefix
	case htBloomBits:
		sectionHead := bgmCore.GetCanonicalHash(pmPtr.chainDb, (idx+1)*light.BloomTrieFrequency-1)
		return light.GetBloomTrieRoot(pmPtr.chainDb, idx, sectionHead), light.BloomTrieTablePrefix
	}
	return bgmcommon.Hash{}, ""
}

// getHelperTrieAuxData returns requested auxiliary data for the given HelperTrie request
func (pmPtr *ProtocolManager) getHelperTrieAuxData(req HelperTrieReq) []byte {
	if req.HelperTrieType == htCanonical && req.AuxReq == auxHeader {
		if len(req.Key) != 8 {
			return nil
		}
		blockNum := binary.BigEndian.Uint64(req.Key)
		hash := bgmCore.GetCanonicalHash(pmPtr.chainDb, blockNum)
		return bgmCore.GetHeaderRLP(pmPtr.chainDb, hash, blockNum)
	}
	return nil
}

func (pmPtr *ProtocolManager) txStatus(hashes []bgmcommon.Hash) []txStatus {
	stats := make([]txStatus, len(hashes))
	for i, stat := range pmPtr.txpool.Status(hashes) {
		// Save the status we've got from the transaction pool
		stats[i].Status = stat

		// If the transaction is unknown to the pool, try looking it up locally
		if stat == bgmCore.TxStatusUnknown {
			if block, number, index := bgmCore.GetTxLookupEntry(pmPtr.chainDb, hashes[i]); block != (bgmcommon.Hash{}) {
				stats[i].Status = bgmCore.TxStatusIncluded
				stats[i].Lookup = &bgmCore.TxLookupEntry{hash: block, BlockIndex: number, Index: index}
			}
		}
	}
	return stats
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (self *ProtocolManager) NodeInfo() *bgmPtr.BgmNodeInfo {
	return &bgmPtr.BgmNodeInfo{
		Network:    self.networkId,
		Difficulty: self.blockchain.GetTdByHash(self.blockchain.Lasthash()),
		Genesis:    self.blockchain.Genesis().Hash(),
		Head:       self.blockchain.Lasthash(),
	}
}

// downloaderPeerNotify implement peerSetNotify
type downloaderPeerNotify ProtocolManager

type peerConnection struct {
	manager *ProtocolManager
	peer    *peer
}

func (pcPtr *peerConnection) Head() (bgmcommon.Hash, *big.Int) {
	return pcPtr.peer.HeadAndTd()
}

func (pcPtr *peerConnection) RequestHeadersByHash(origin bgmcommon.Hash, amount int, skip int, reverse bool) error {
	reqID := genReqID()
	rq := &distReq{
		getCost: func(dp distPeer) Uint64 {
			peer := dp.(*peer)
			return peer.GetRequestCost(GetBlockHeadersMsg, amount)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*peer) == pcPtr.peer
		},
		request: func(dp distPeer) func() {
			peer := dp.(*peer)
			cost := peer.GetRequestCost(GetBlockHeadersMsg, amount)
			peer.fcServer.QueueRequest(reqID, cost)
			return func() { peer.RequestHeadersByHash(reqID, cost, origin, amount, skip, reverse) }
		},
	}
	_, ok := <-pcPtr.manager.reqDist.queue(rq)
	if !ok {
		return ErrNoPeers
	}
	return nil
}

func (pcPtr *peerConnection) RequestHeadersByNumber(origin Uint64, amount int, skip int, reverse bool) error {
	reqID := genReqID()
	rq := &distReq{
		getCost: func(dp distPeer) Uint64 {
			peer := dp.(*peer)
			return peer.GetRequestCost(GetBlockHeadersMsg, amount)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*peer) == pcPtr.peer
		},
		request: func(dp distPeer) func() {
			peer := dp.(*peer)
			cost := peer.GetRequestCost(GetBlockHeadersMsg, amount)
			peer.fcServer.QueueRequest(reqID, cost)
			return func() { peer.RequestHeadersByNumber(reqID, cost, origin, amount, skip, reverse) }
		},
	}
	_, ok := <-pcPtr.manager.reqDist.queue(rq)
	if !ok {
		return ErrNoPeers
	}
	return nil
}

func (d *downloaderPeerNotify) registerPeer(p *peer) {
	pm := (*ProtocolManager)(d)
	pc := &peerConnection{
		manager: pm,
		peer:    p,
	}
	pmPtr.downloader.RegisterLightPeer(p.id, bgmVersion, pc)
}

func (d *downloaderPeerNotify) unregisterPeer(p *peer) {
	pm := (*ProtocolManager)(d)
	pmPtr.downloader.UnregisterPeer(p.id)
}
