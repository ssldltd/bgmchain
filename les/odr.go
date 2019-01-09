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

	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/light"
	"github.com/ssldltd/bgmchain/bgmlogs"
)

// LesOdr implement light.OdrBackend
type LesOdr struct {
	db                                         bgmdbPtr.Database
	chtIndexer, bloomTrieIndexer, bloomIndexer *bgmcore.ChainIndexer
	retriever                                  *retrieveManager
	stop                                       chan struct{}
}

func NewLesOdr(db bgmdbPtr.Database, chtIndexer, bloomTrieIndexer, bloomIndexer *bgmcore.ChainIndexer, retriever *retrieveManager) *LesOdr {
	return &LesOdr{
		db:               db,
		chtIndexer:       chtIndexer,
		bloomTrieIndexer: bloomTrieIndexer,
		bloomIndexer:     bloomIndexer,
		retriever:        retriever,
		stop:             make(chan struct{}),
	}
}

// Stop cancels all pending retrievals
func (odr *LesOdr) Stop() {
	close(odr.stop)
}

// Database returns the backing database
func (odr *LesOdr) Database() bgmdbPtr.Database {
	return odr.db
}

// ChtIndexer returns the CHT chain indexer
func (odr *LesOdr) ChtIndexer() *bgmcore.ChainIndexer {
	return odr.chtIndexer
}

// BloomTrieIndexer returns the bloom trie chain indexer
func (odr *LesOdr) BloomTrieIndexer() *bgmcore.ChainIndexer {
	return odr.bloomTrieIndexer
}

// BloomIndexer returns the bloombits chain indexer
func (odr *LesOdr) BloomIndexer() *bgmcore.ChainIndexer {
	return odr.bloomIndexer
}

const (
	MsgBlockBodies = iota
	MsgCode
	MsgRecChaints
	MsgProofsV1
	MsgProofsV2
	MsgHeaderProofs
	MsgHelperTrieProofs
)

// Msg encodes a LES message that delivers reply data for a request
type Msg struct {
	MsgType int
	ReqID   uint64
	Obj     interface{}
}

// Retrieve tries to fetch an object from the LES network.
// If the network retrieval was successful, it stores the object in local dbPtr.
func (odr *LesOdr) Retrieve(ctx context.Context, req light.OdrRequest) (err error) {
	lreq := LesRequest(req)

	reqID := genReqID()
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			return lreq.GetCost(dp.(*peer))
		},
		canSend: func(dp distPeer) bool {
			p := dp.(*peer)
			return lreq.CanSend(p)
		},
		request: func(dp distPeer) func() {
			p := dp.(*peer)
			cost := lreq.GetCost(p)
			p.fcServer.QueueRequest(reqID, cost)
			return func() { lreq.Request(reqID, p) }
		},
	}

	if err = odr.retriever.retrieve(ctx, reqID, rq, func(p distPeer, msg *Msg) error { return lreq.Validate(odr.db, msg) }, odr.stop); err == nil {
		// retrieved from network, store in db
		req.StoreResult(odr.db)
	} else {
		bgmlogs.Debug("Failed to retrieve data from network", "err", err)
	}
	return
}
