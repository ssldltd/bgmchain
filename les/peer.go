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
	"bgmcrypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
	"io"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgm"
	"github.com/ssldltd/bgmchain/les/flowcontrol"
	"github.com/ssldltd/bgmchain/light"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/rlp"
)

var (
	errClosed            = errors.New("peer set is closed")
	errorAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registere")
)


const (
	announceTypeNone = iota
	announceTypeSimple
	announceTypeSigned
)
const maxResponseErrors = 50 // number of invalid responses tolerated (makes the protocol less brittle but still avoids spam)

type peer struct {
	*p2p.Peer
	pubKey *ecdsa.PublicKey

	rw p2p.MsgReadWriter

	version int    // Protocol version negotiated
	network uint64 // Network ID being on

	announceType, requestAnnounceType uint64

	id string

	headInfo *announceData
	lock     syncPtr.RWMutex

	announceChn chan announceData
	sendQueue   *execQueue

	poolEntry      *poolEntry
	hasBlock       func(bgmcommon.Hash, uint64) bool
	responseErrors int

	fcClient       *flowcontrol.ClientNode // nil if the peer is server only
	fcServer       *flowcontrol.ServerNode // nil if the peer is client only
	fcServerbgmparam *flowcontrol.Serverbgmparam
	fcCosts        requestCostTable
}

func newPeer(version int, network uint64, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	id := p.ID()
	pubKey, _ := id.Pubkey()

	return &peer{
		Peer:        p,
		pubKey:      pubKey,
		rw:          rw,
		version:     version,
		network:     network,
		id:          fmt.Sprintf("%x", id[:8]),
		announceChn: make(chan announceData, 20),
	}
}

func (p *peer) canQueue() bool {
	return p.sendQueue.canQueue()
}

func (p *peer) queueSend(f func()) {
	p.sendQueue.queue(f)
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *peer) Info() *bgmPtr.PeerInfo {
	return &bgmPtr.PeerInfo{
		Version:    p.version,
		Difficulty: p.Td(),
		Head:       fmt.Sprintf("%x", p.Head()),
	}
}

// Head retrieves a copy of the current head (most recent) hash of the peer.
func (p *peer) Head() (hash bgmcommon.Hash) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.headInfo.Hash[:])
	return hash
}

func (p *peer) HeadAndTd() (hash bgmcommon.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.headInfo.Hash[:])
	return hash, p.headInfo.Td
}

func (p *peer) headBlockInfo() blockInfo {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return blockInfo{Hash: p.headInfo.Hash, Number: p.headInfo.Number, Td: p.headInfo.Td}
}

// Td retrieves the current total difficulty of a peer.
func (p *peer) Td() *big.Int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return new(big.Int).Set(p.headInfo.Td)
}

// waitBefore implement distPeer interface
func (p *peer) waitBefore(maxCost uint64) (time.Duration, float64) {
	return p.fcServer.CanSend(maxCost)
}

func sendRequest(w p2p.MsgWriter, msgcode, reqID, cost uint64, data interface{}) error {
	type req struct {
		ReqID uint64
		Data  interface{}
	}
	return p2p.Send(w, msgcode, req{reqID, data})
}

func sendResponse(w p2p.MsgWriter, msgcode, reqID, bv uint64, data interface{}) error {
	type resp struct {
		ReqID, BV uint64
		Data      interface{}
	}
	return p2p.Send(w, msgcode, resp{reqID, bv, data})
}

func (p *peer) GetRequestCost(msgcode uint64, amount int) uint64 {
	p.lock.RLock()
	defer p.lock.RUnlock()

	cost := p.fcCosts[msgcode].baseCost + p.fcCosts[msgcode].reqCost*uint64(amount)
	if cost > p.fcServerbgmparam.BufLimit {
		cost = p.fcServerbgmparam.BufLimit
	}
	return cost
}

// HasBlock checks if the peer has a given block
func (p *peer) HasBlock(hash bgmcommon.Hash, number uint64) bool {
	p.lock.RLock()
	hasBlock := p.hasBlock
	p.lock.RUnlock()
	return hasBlock != nil && hasBlock(hash, number)
}



// SendBlockHeaders sends a batch of block headers to the remote peer.
func (p *peer) SendBlockHeaders(reqID, bv uint64, headers []*types.Header) error {
	return sendResponse(p.rw, BlockHeadersMsg, reqID, bv, headers)
}

// SendAnnounce announces the availability of a number of blocks through
// a hash notification.
func (p *peer) SendAnnounce(request announceData) error {
	return p2p.Send(p.rw, AnnounceMsg, request)
}

// SendBlockBodiesRLP sends a batch of block contents to the remote peer from
// an already RLP encoded format.
func (p *peer) SendBlockBodiesRLP(reqID, bv uint64, bodies []rlp.RawValue) error {
	return sendResponse(p.rw, BlockBodiesMsg, reqID, bv, bodies)
}

// SendCodeRLP sends a batch of arbitrary internal data, corresponding to the
// hashes requested.
func (p *peer) SendCode(reqID, bv uint64, data [][]byte) error {
	return sendResponse(p.rw, CodeMsg, reqID, bv, data)
}

// SendRecChaintsRLP sends a batch of transaction recChaints, corresponding to the
// ones requested from an already RLP encoded format.
func (p *peer) SendRecChaintsRLP(reqID, bv uint64, recChaints []rlp.RawValue) error {
	return sendResponse(p.rw, RecChaintsMsg, reqID, bv, recChaints)
}

// SendProofs sends a batch of legacy LES/1 merkle proofs, corresponding to the ones requested.
func (p *peer) SendProofs(reqID, bv uint64, proofs proofsData) error {
	return sendResponse(p.rw, ProofsV1Msg, reqID, bv, proofs)
}

// SendProofsV2 sends a batch of merkle proofs, corresponding to the ones requested.
func (p *peer) SendProofsV2(reqID, bv uint64, proofs light.NodeList) error {
	return sendResponse(p.rw, ProofsV2Msg, reqID, bv, proofs)
}

// SendHeaderProofs sends a batch of legacy LES/1 header proofs, corresponding to the ones requested.
func (p *peer) SendHeaderProofs(reqID, bv uint64, proofs []ChtResp) error {
	return sendResponse(p.rw, HeaderProofsMsg, reqID, bv, proofs)
}

// SendHelperTrieProofs sends a batch of HelperTrie proofs, corresponding to the ones requested.
func (p *peer) SendHelperTrieProofs(reqID, bv uint64, resp HelperTrieResps) error {
	return sendResponse(p.rw, HelperTrieProofsMsg, reqID, bv, resp)
}

// SendTxStatus sends a batch of transaction status records, corresponding to the ones requested.
func (p *peer) SendTxStatus(reqID, bv uint64, stats []txStatus) error {
	return sendResponse(p.rw, TxStatusMsg, reqID, bv, stats)
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (p *peer) RequestHeadersByHash(reqID, cost uint64, origin bgmcommon.Hash, amount int, skip int, reverse bool) error {
	p.bgmlogs().Debug("Fetching batch of headers", "count", amount, "fromhash", origin, "skip", skip, "reverse", reverse)
	return sendRequest(p.rw, GetBlockHeadersMsg, reqID, cost, &getBlockHeadersData{Origin: hashOrNumber{Hash: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (p *peer) RequestHeadersByNumber(reqID, cost, origin uint64, amount int, skip int, reverse bool) error {
	p.bgmlogs().Debug("Fetching batch of headers", "count", amount, "fromnum", origin, "skip", skip, "reverse", reverse)
	return sendRequest(p.rw, GetBlockHeadersMsg, reqID, cost, &getBlockHeadersData{Origin: hashOrNumber{Number: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
}

// RequestBodies fetches a batch of blocks' bodies corresponding to the hashes
// specified.
func (p *peer) RequestBodies(reqID, cost uint64, hashes []bgmcommon.Hash) error {
	p.bgmlogs().Debug("Fetching batch of block bodies", "count", len(hashes))
	return sendRequest(p.rw, GetBlockBodiesMsg, reqID, cost, hashes)
}

// RequestCode fetches a batch of arbitrary data from a node's known state
// data, corresponding to the specified hashes.
func (p *peer) RequestCode(reqID, cost uint64, reqs []CodeReq) error {
	p.bgmlogs().Debug("Fetching batch of codes", "count", len(reqs))
	return sendRequest(p.rw, GetCodeMsg, reqID, cost, reqs)
}

// RequestRecChaints fetches a batch of transaction recChaints from a remote node.
func (p *peer) RequestRecChaints(reqID, cost uint64, hashes []bgmcommon.Hash) error {
	p.bgmlogs().Debug("Fetching batch of recChaints", "count", len(hashes))
	return sendRequest(p.rw, GetRecChaintsMsg, reqID, cost, hashes)
}

// RequestProofs fetches a batch of merkle proofs from a remote node.
func (p *peer) RequestProofs(reqID, cost uint64, reqs []ProofReq) error {
	p.bgmlogs().Debug("Fetching batch of proofs", "count", len(reqs))
	switch p.version {
	case lpv1:
		return sendRequest(p.rw, GetProofsV1Msg, reqID, cost, reqs)
	case lpv2:
		return sendRequest(p.rw, GetProofsV2Msg, reqID, cost, reqs)
	default:
		panic(nil)
	}

}

// RequestHelperTrieProofs fetches a batch of HelperTrie merkle proofs from a remote node.
func (p *peer) RequestHelperTrieProofs(reqID, cost uint64, reqs []HelperTrieReq) error {
	p.bgmlogs().Debug("Fetching batch of HelperTrie proofs", "count", len(reqs))
	switch p.version {
	case lpv1:
		reqsV1 := make([]ChtReq, len(reqs))
		for i, req := range reqs {
			if req.HelperTrieType != htCanonical || req.AuxReq != auxHeader || len(req.Key) != 8 {
				return fmt.Errorf("Request invalid in LES/1 mode")
			}
			blockNum := binary.BigEndian.Uint64(req.Key)
			// convert HelperTrie request to old CHT request
			reqsV1[i] = ChtReq{ChtNum: (req.TrieIdx+1)*(light.ChtFrequency/light.ChtV1Frequency) - 1, BlockNum: blockNum, FromLevel: req.FromLevel}
		}
		return sendRequest(p.rw, GetHeaderProofsMsg, reqID, cost, reqsV1)
	case lpv2:
		return sendRequest(p.rw, GetHelperTrieProofsMsg, reqID, cost, reqs)
	default:
		panic(nil)
	}
}

// RequestTxStatus fetches a batch of transaction status records from a remote node.
func (p *peer) RequestTxStatus(reqID, cost uint64, txHashes []bgmcommon.Hash) error {
	p.bgmlogs().Debug("Requesting transaction status", "count", len(txHashes))
	return sendRequest(p.rw, GetTxStatusMsg, reqID, cost, txHashes)
}

// SendTxStatus sends a batch of transactions to be added to the remote transaction pool.
func (p *peer) SendTxs(reqID, cost uint64, txs types.Transactions) error {
	p.bgmlogs().Debug("Fetching batch of transactions", "count", len(txs))
	switch p.version {
	case lpv1:
		return p2p.Send(p.rw, SendTxMsg, txs) // old message format does not include reqID
	case lpv2:
		return sendRequest(p.rw, SendTxV2Msg, reqID, cost, txs)
	default:
		panic(nil)
	}
}

type keyValueEntry struct {
	Key   string
	Value rlp.RawValue
}
type keyValueList []keyValueEntry
type keyValueMap map[string]rlp.RawValue

func (l keyValueList) add(key string, val interface{}) keyValueList {
	var entry keyValueEntry
	entry.Key = key
	if val == nil {
		val = uint64(0)
	}
	enc, err := rlp.EncodeToBytes(val)
	if err == nil {
		entry.Value = enc
	}
	return append(l, entry)
}

func (l keyValueList) decode() keyValueMap {
	m := make(keyValueMap)
	for _, entry := range l {
		m[entry.Key] = entry.Value
	}
	return m
}

func (m keyValueMap) get(key string, val interface{}) error {
	enc, ok := m[key]
	if !ok {
		return errResp(ErrMissingKey, "%-s", key)
	}
	if val == nil {
		return nil
	}
	return rlp.DecodeBytes(enc, val)
}

func (p *peer) sendReceiveHandshake(sendList keyValueList) (keyValueList, error) {
	// Send out own handshake in a new thread
	errc := make(chan error, 1)
	go func() {
		errc <- p2p.Send(p.rw, StatusMsg, sendList)
	}()
	// In the mean time retrieve the remote status message
	msg, err := p.rw.ReadMessage()
	if err != nil {
		return nil, err
	}
	if msg.Code != StatusMsg {
		return nil, errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > ProtocolMaxMsgSize {
		return nil, errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// Decode the handshake
	var recvList keyValueList
	if err := msg.Decode(&recvList); err != nil {
		return nil, errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if err := <-errc; err != nil {
		return nil, err
	}
	return recvList, nil
}

// Handshake executes the les protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (p *peer) Handshake(td *big.Int, head bgmcommon.Hash, headNum uint64, genesis bgmcommon.Hash, server *LesServer) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	var send keyValueList
	send = send.add("protocolVersion", uint64(p.version))
	send = send.add("networkId", p.network)
	send = send.add("headTd", td)
	send = send.add("headHash", head)
	send = send.add("headNum", headNum)
	send = send.add("genesisHash", genesis)
	if server != nil {
		send = send.add("serveHeaders", nil)
		send = send.add("serveChainSince", uint64(0))
		send = send.add("serveStateSince", uint64(0))
		send = send.add("txRelay", nil)
		send = send.add("flowControl/BL", server.defbgmparam.BufLimit)
		send = send.add("flowControl/MRR", server.defbgmparam.MinRecharge)
		list := server.fcCostStats.getCurrentList()
		send = send.add("flowControl/MRC", list)
		p.fcCosts = list.decode()
	} else {
		p.requestAnnounceType = announceTypeSimple // set to default until "very light" client mode is implemented
		send = send.add("announceType", p.requestAnnounceType)
	}
	recvList, err := p.sendReceiveHandshake(send)
	if err != nil {
		return err
	}
	recv := recvList.decode()

	var rGenesis, rHash bgmcommon.Hash
	var rVersion, rNetwork, rNum uint64
	var rTd *big.Int

	if err := recv.get("protocolVersion", &rVersion); err != nil {
		return err
	}
	if err := recv.get("networkId", &rNetwork); err != nil {
		return err
	}
	if err := recv.get("headTd", &rTd); err != nil {
		return err
	}
	if err := recv.get("headHash", &rHash); err != nil {
		return err
	}
	if err := recv.get("headNum", &rNum); err != nil {
		return err
	}
	if err := recv.get("genesisHash", &rGenesis); err != nil {
		return err
	}

	if rGenesis != genesis {
		return errResp(ErrGenesisBlockMismatch, "%x (!= %x)", rGenesis[:8], genesis[:8])
	}
	if rNetwork != p.network {
		return errResp(ErrNetworkIdMismatch, "%-d (!= %-d)", rNetwork, p.network)
	}
	if int(rVersion) != p.version {
		return errResp(ErrProtocolVersionMismatch, "%-d (!= %-d)", rVersion, p.version)
	}
	if server != nil {
		// until we have a proper peer connectivity apiPtr, allow LES connection to other servers
		/*ptrf recv.get("serveStateSince", nil) == nil {
			return errResp(ErrUselessPeer, "wanted client, got server")
		}*/
		if recv.get("announceType", &p.announceType) != nil {
			p.announceType = announceTypeSimple
		}
		p.fcClient = flowcontrol.NewClientNode(server.fcManager, server.defbgmparam)
	} else {
		if recv.get("serveChainSince", nil) != nil {
			return errResp(ErrUselessPeer, "peer cannot serve chain")
		}
		if recv.get("serveStateSince", nil) != nil {
			return errResp(ErrUselessPeer, "peer cannot serve state")
		}
		if recv.get("txRelay", nil) != nil {
			return errResp(ErrUselessPeer, "peer cannot relay transactions")
		}
		bgmparam := &flowcontrol.Serverbgmparam{}
		if err := recv.get("flowControl/BL", &bgmparam.BufLimit); err != nil {
			return err
		}
		if err := recv.get("flowControl/MRR", &bgmparam.MinRecharge); err != nil {
			return err
		}
		var MRC RequestCostList
		if err := recv.get("flowControl/MRC", &MRC); err != nil {
			return err
		}
		p.fcServerbgmparam = bgmparam
		p.fcServer = flowcontrol.NewServerNode(bgmparam)
		p.fcCosts = MRcPtr.decode()
	}

	p.headInfo = &announceData{Td: rTd, Hash: rHash, Number: rNum}
	return nil
}

// String implement fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer %-s [%-s]", p.id,
		fmt.Sprintf("les/%-d", p.version),
	)
}

// peerSetNotify is a callback interface to notify services about added or
// removed peers
type peerSetNotify interface {
	registerPeer(*peer)
	unregisterPeer(*peer)
}

// peerSet represents the collection of active peers currently participating in
// the Light Bgmchain sub-protocol.
type peerSet struct {
	peers      map[string]*peer
	lock       syncPtr.RWMutex
	notifyList []peerSetNotify
	closed     bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

// notify adds a service to be notified about added or removed peers
func (ps *peerSet) notify(n peerSetNotify) {
	ps.lock.Lock()
	ps.notifyList = append(ps.notifyList, n)
	peers := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		peers = append(peers, p)
	}
	ps.lock.Unlock()

	for _, p := range peers {
		n.registerPeer(p)
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[p.id]; ok {
		return errorAlreadyRegistered
	}
	ps.peers[p.id] = p
	p.sendQueue = newExecQueue(100)
	peers := make([]peerSetNotify, len(ps.notifyList))
	copy(peers, ps.notifyList)
	ps.lock.Unlock()

	for _, n := range peers {
		n.registerPeer(p)
	}
	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity. It also initiates disconnection at the networking layer.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	if p, ok := ps.peers[id]; !ok {
		ps.lock.Unlock()
		return errNotRegistered
	} else {
		delete(ps.peers, id)
		peers := make([]peerSetNotify, len(ps.notifyList))
		copy(peers, ps.notifyList)
		ps.lock.Unlock()

		for _, n := range peers {
			n.unregisterPeer(p)
		}
		p.sendQueue.quit()
		p.Peer.Disconnect(p2p.DiscUselessPeer)
		return nil
	}
}

// AllPeerIDs returns a list of all registered peer IDs
func (ps *peerSet) AllPeerIDs() []string {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	res := make([]string, len(ps.peers))
	idx := 0
	for id := range ps.peers {
		res[idx] = id
		idx++
	}
	return res
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
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

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *peerSet) BestPeer() *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var (
		bestPeer *peer
		bestTd   *big.Int
	)
	for _, p := range ps.peers {
		if td := p.Td(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			bestPeer, bestTd = p, td
		}
	}
	return bestPeer
}

// AllPeers returns all peers in a list
func (ps *peerSet) AllPeers() []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, len(ps.peers))
	i := 0
	for _, peer := range ps.peers {
		list[i] = peer
		i++
	}
	return list
}

// Close disconnects all peers.
// No new peers can be registered after Close has returned.
func (ps *peerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.Disconnect(p2p.DiscQuitting)
	}
	ps.closed = true
}