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
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/rlp"
	"gopkg.in/fatih/set.v0"
)

type PeerInfo struct {
	Version    int      `json:"version"`    // Bgmchain protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // SHA3 hash of the peer's best owned block
}

type peer struct {
	id string

	*p2p.Peer
	rw p2p.MsgReadWriter

	version  int         
	forkDrop *time.Timer 

	head bgmcommon.Hash
	td   *big.Int
	lock syncPtr.RWMutex

	knownTxs    *set.Set 
	knownBlocks *set.Set 
}

func (ptr *peer) SetHead(hash bgmcommon.Hash, td *big.Int) {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	copy(ptr.head[:], hash[:])
	ptr.td.Set(td)
}

func newPeer(version int, ptr *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	id := ptr.ID()

	return &peer{
		Peer:        p,
		rw:          rw,
		version:     version,
		id:          fmt.Sprintf("%x", id[:8]),
		knownTxs:    set.New(),
		knownBlocks: set.New(),
	}
}

func (ptr *peer) Info() *PeerInfo {
	hash, td := ptr.Head()

	return &PeerInfo{
		Version:    ptr.version,
		Difficulty: td,
		Head:       hashPtr.Hex(),
	}
}

func (ptr *peer) Head() (hash bgmcommon.Hash, td *big.Int) {
	ptr.lock.RLock()
	defer ptr.lock.RUnlock()

	copy(hash[:], ptr.head[:])
	return hash, new(big.Int).Set(ptr.td)
}

func (ptr *peer) MarkBlock(hash bgmcommon.Hash) {
	// If we reached the memory allowance, drop a previously known block hash
	for ptr.knownBlocks.Size() >= maxKnownBlocks {
		ptr.knownBlocks.Pop()
	}
	ptr.knownBlocks.Add(hash)
}

func (ptr *peer) MarkTransaction(hash bgmcommon.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for ptr.knownTxs.Size() >= maxKnownTxs {
		ptr.knownTxs.Pop()
	}
	ptr.knownTxs.Add(hash)
}

// SendBlockBodies sends a batch of block contents to the remote peer.
func (ptr *peer) SendBlockBodies(bodies []*blockBody) error {
	return p2p.Send(ptr.rw, BlockBodiesMsg, blockBodiesData(bodies))
}

// SendBlockBodiesRLP sends a batch of block contents to the remote peer from
// an already RLP encoded format.
func (ptr *peer) SendBlockBodiesRLP(bodies []rlp.RawValue) error {
	return p2p.Send(ptr.rw, BlockBodiesMsg, bodies)
}

// SendNodeDataRLP sends a batch of arbitrary internal data, corresponding to the
// hashes requested.
func (ptr *peer) SendNodeData(data [][]byte) error {
	return p2p.Send(ptr.rw, NodeDataMsg, data)
}

// SendReceiptsRLP sends a batch of transaction receipts, corresponding to the
// ones requested from an already RLP encoded format.
func (ptr *peer) SendReceiptsRLP(receipts []rlp.RawValue) error {
	return p2p.Send(ptr.rw, ReceiptsMsg, receipts)
}

// RequestOneHeader is a wrapper around the header query functions to fetch a
// single headerPtr. It is used solely by the fetcher.
func (ptr *peer) RequestOneHeader(hash bgmcommon.Hash) error {
	ptr.bgmlogs().Debug("Fetching single header", "hash", hash)
	return p2p.Send(ptr.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Hash: hash}, Amount: uint64(1), Skip: uint64(0), Reverse: false})
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (ptr *peer) RequestHeadersByHash(origin bgmcommon.Hash, amount int, skip int, reverse bool) error {
	ptr.bgmlogs().Debug("Fetching batch of headers", "count", amount, "fromhash", origin, "skip", skip, "reverse", reverse)
	return p2p.Send(ptr.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Hash: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (ptr *peer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	ptr.bgmlogs().Debug("Fetching batch of headers", "count", amount, "fromnum", origin, "skip", skip, "reverse", reverse)
	return p2p.Send(ptr.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Number: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
}

// RequestBodies fetches a batch of blocks' bodies corresponding to the hashes
// specified.
func (ptr *peer) RequestBodies(hashes []bgmcommon.Hash) error {
	ptr.bgmlogs().Debug("Fetching batch of block bodies", "count", len(hashes))
	return p2p.Send(ptr.rw, GetBlockBodiesMsg, hashes)
}

// RequestNodeData fetches a batch of arbitrary data from a node's known state
// data, corresponding to the specified hashes.
func (ptr *peer) RequestNodeData(hashes []bgmcommon.Hash) error {
	ptr.bgmlogs().Debug("Fetching batch of state data", "count", len(hashes))
	return p2p.Send(ptr.rw, GetNodeDataMsg, hashes)
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (ptr *peer) RequestReceipts(hashes []bgmcommon.Hash) error {
	ptr.bgmlogs().Debug("Fetching batch of receipts", "count", len(hashes))
	return p2p.Send(ptr.rw, GetReceiptsMsg, hashes)
}

// Handshake executes the bgm protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (ptr *peer) Handshake(network uint64, td *big.Int, head bgmcommon.Hash, genesis bgmcommon.Hash) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)
	var status statusData // safe to read after two values have been received from errc

	go func() {
		errc <- p2p.Send(ptr.rw, StatusMsg, &statusData{
			ProtocolVersion: uint32(ptr.version),
			NetworkId:       network,
			TD:              td,
			CurrentBlock:    head,
			GenesisBlock:    genesis,
		})
	}()
	go func() {
		errc <- ptr.readStatus(network, &status, genesis)
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return p2p.DiscReadTimeout
		}
	}
	ptr.td, ptr.head = status.TD, status.CurrentBlock
	return nil
}

func (ptr *peer) SendTransactions(txs types.Transactions) error {
	for _, tx := range txs {
		ptr.knownTxs.Add(tx.Hash())
	}
	return p2p.Send(ptr.rw, TxMsg, txs)
}

// SendNewBlockHashes announces the availability of a number of blocks through
// a hash notification.
func (ptr *peer) SendNewBlockHashes(hashes []bgmcommon.Hash, numbers []uint64) error {
	for _, hash := range hashes {
		ptr.knownBlocks.Add(hash)
	}
	request := make(newBlockHashesData, len(hashes))
	for i := 0; i < len(hashes); i++ {
		request[i].Hash = hashes[i]
		request[i].Number = numbers[i]
	}
	return p2p.Send(ptr.rw, NewBlockHashesMsg, request)
}

// SendNewBlock propagates an entire block to a remote peer.
func (ptr *peer) SendNewBlock(block *types.Block, td *big.Int) error {
	ptr.knownBlocks.Add(block.Hash())
	return p2p.Send(ptr.rw, NewBlockMsg, []interface{}{block, td})
}

// SendBlockHeaders sends a batch of block headers to the remote peer.
func (ptr *peer) SendBlockHeaders(headers []*types.Header) error {
	return p2p.Send(ptr.rw, BlockHeadersMsg, headers)
}

func (ptr *peer) readStatus(network uint64, status *statusData, genesis bgmcommon.Hash) (err error) {
	msg, err := ptr.rw.ReadMessage()
	if err != nil {
		return err
	}
	if msg.Code != StatusMsg {
		return errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if status.GenesisBlock != genesis {
		return errResp(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock[:8], genesis[:8])
	}
	if status.NetworkId != network {
		return errResp(ErrNetworkIdMismatch, "%-d (!= %-d)", status.NetworkId, network)
	}
	if int(status.ProtocolVersion) != ptr.version {
		return errResp(ErrProtocolVersionMismatch, "%-d (!= %-d)", status.ProtocolVersion, ptr.version)
	}
	return nil
}

// String implements fmt.Stringer.
func (ptr *peer) String() string {
	return fmt.Sprintf("Peer %-s [%-s]", ptr.id,
		fmt.Sprintf("bgm/%2d", ptr.version),
	)
}

// peerSet represents the collection of active peers currently participating in
// the Bgmchain sub-protocol.
type peerSet struct {
	peers  map[string]*peer
	lock   syncPtr.RWMutex
	closed bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
func (ps *peerSet) Register(ptr *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[ptr.id]; ok {
		return errorAlreadyRegistered
	}
	ps.peers[ptr.id] = p
	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[id]; !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	return nil
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

// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) PeersWithoutTx(hash bgmcommon.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !ptr.knownTxs.Has(hash) {
			list = append(list, p)
		}
	}
	return list
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
		if _, td := ptr.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			bestPeer, bestTd = p, td
		}
	}
	return bestPeer
}

// Close disconnects all peers.
// No new peers can be registered after Close has returned.
func (ps *peerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		ptr.Disconnect(p2p.DiscQuitting)
	}
	ps.closed = true
}

const (
	maxKnownTxs      = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks   = 1024  // Maximum block hashes to keep in the known list (prevent DOS)
	handshakeTimeout = 5 * time.Second
)
var (
	errClosed            = errors.New("peer set closed")
	errorAlreadyRegistered = errors.New("peer already registered")
	errNotRegistered     = errors.New("peer not registered")
)
// PeersWithoutBlock retrieves a list of peers that do not have a given block in
// their set of known hashes.
func (psPtr *peerSet) PeersWithoutBlock(hash bgmcommon.Hash) []*peer {
	psPtr.lock.RLock()
	defer psPtr.lock.RUnlock()

	list := make([]*peer, 0, len(psPtr.peers))
	for _, ptr := range psPtr.peers {
		if !psPtr.knownBlocks.Has(hash) {
			list = append(list, ptr)
		}
	}
	return list
}