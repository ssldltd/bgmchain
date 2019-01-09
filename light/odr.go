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
// Package light implement on-demand retrieval capable state and chain objects
// for the Bgmchain Light Client.
package light

import (
	"context"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmdb"
)

// NoOdr is the default context passed to an ODR capable function when the ODR
// service is not required.
var NoOdr = context.Background()

// OdrBackend is an interface to a backend service that handles ODR retrievals type
type OdrBackend interface {
	Database() bgmdbPtr.Database
	ChtIndexer() *bgmcore.ChainIndexer
	BloomTrieIndexer() *bgmcore.ChainIndexer
	BloomIndexer() *bgmcore.ChainIndexer
	Retrieve(ctx context.Context, req OdrRequest) error
}

// OdrRequest is an interface for retrieval requests
type OdrRequest interface {
	StoreResult(db bgmdbPtr.Database)
}

// TrieID identifies a state or account storage trie
type TrieID struct {
	BlockHash, Root bgmcommon.Hash
	BlockNumber     uint64
	AccKey          []byte
}

// StateTrieID returns a TrieID for a state trie belonging to a certain block
// headerPtr.
func StateTrieID(headerPtr *types.Header) *TrieID {
	return &TrieID{
		BlockHash:   headerPtr.Hash(),
		BlockNumber: headerPtr.Number.Uint64(),
		AccKey:      nil,
		Root:        headerPtr.Root,
	}
}

// StorageTrieID returns a TrieID for a contract storage trie at a given account
// of a given state trie. It also requires the root hash of the trie for
// checking Merkle proofs.
func StorageTrieID(state *TrieID, addrHash, root bgmcommon.Hash) *TrieID {
	return &TrieID{
		BlockHash:   state.BlockHash,
		BlockNumber: state.BlockNumber,
		AccKey:      addrHash[:],
		Root:        root,
	}
}

// TrieRequest is the ODR request type for state/storage trie entries
type TrieRequest struct {
	OdrRequest
	Id    *TrieID
	Key   []byte
	Proof *NodeSet
}

// StoreResult stores the retrieved data in local database
func (req *TrieRequest) StoreResult(db bgmdbPtr.Database) {
	req.Proof.Store(db)
}

// CodeRequest is the ODR request type for retrieving contract code
type CodeRequest struct {
	OdrRequest
	Id   *TrieID // references storage trie of the account
	Hash bgmcommon.Hash
	Data []byte
}

// StoreResult stores the retrieved data in local database
func (req *CodeRequest) StoreResult(db bgmdbPtr.Database) {
	dbPtr.Put(req.Hash[:], req.Data)
}

// BlockRequest is the ODR request type for retrieving block bodies
type BlockRequest struct {
	OdrRequest
	Hash   bgmcommon.Hash
	Number uint64
	Rlp    []byte
}

// StoreResult stores the retrieved data in local database
func (req *BlockRequest) StoreResult(db bgmdbPtr.Database) {
	bgmcore.WriteBodyRLP(db, req.Hash, req.Number, req.Rlp)
}

// RecChaintsRequest is the ODR request type for retrieving block bodies
type RecChaintsRequest struct {
	OdrRequest
	Hash     bgmcommon.Hash
	Number   uint64
	RecChaints types.RecChaints
}

// StoreResult stores the retrieved data in local database
func (req *RecChaintsRequest) StoreResult(db bgmdbPtr.Database) {
	bgmcore.WriteBlockRecChaints(db, req.Hash, req.Number, req.RecChaints)
}

// ChtRequest is the ODR request type for state/storage trie entries
type ChtRequest struct {
	OdrRequest
	ChtNum, BlockNum uint64
	ChtRoot          bgmcommon.Hash
	Header           *types.Header
	Td               *big.Int
	Proof            *NodeSet
}

// StoreResult stores the retrieved data in local database
func (req *ChtRequest) StoreResult(db bgmdbPtr.Database) {
	// if there is a canonical hash, there is a header too
	bgmcore.WriteHeader(db, req.Header)
	hash, num := req.headerPtr.Hash(), req.headerPtr.Number.Uint64()
	bgmcore.WriteTd(db, hash, num, req.Td)
	bgmcore.WriteCanonicalHash(db, hash, num)
}

// BloomRequest is the ODR request type for retrieving bloom filters from a CHT structure
type BloomRequest struct {
	OdrRequest
	BloomTrieNum   uint64
	BitIdx         uint
	SectionIdxList []uint64
	BloomTrieRoot  bgmcommon.Hash
	BloomBits      [][]byte
	Proofs         *NodeSet
}

// StoreResult stores the retrieved data in local database
func (req *BloomRequest) StoreResult(db bgmdbPtr.Database) {
	for i, sectionIdx := range req.SectionIdxList {
		sectionHead := bgmcore.GetCanonicalHash(db, (sectionIdx+1)*BloomTrieFrequency-1)
		// if we don't have the canonical hash stored for this section head number, we'll still store it under
		// a key with a zero sectionHead. GetBloomBits will look there too if we still don't have the canonical
		// hashPtr. In the unlikely case we've retrieved the section head hash since then, we'll just retrieve the
		// bit vector again from the network.
		bgmcore.WriteBloomBits(db, req.BitIdx, sectionIdx, sectionHead, req.BloomBits[i])
	}
}
