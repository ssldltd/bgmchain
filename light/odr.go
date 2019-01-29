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
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmdb"
)

// NoOdr is the default context passed to an ODR capable function when the ODR
// service is not required.
var NoOdr = context.Background()

// OdrBackend is an interface to a backend service that handles ODR retrievals type
type OdrBackend interface {
	Database() bgmdbPtr.Database
	ChtIndexer() *bgmCore.ChainIndexer
	BloomTrieIndexer() *bgmCore.ChainIndexer
	BloomIndexer() *bgmCore.ChainIndexer
	Retrieve(CTX context.Context, req OdrRequest) error
}

// OdrRequest is an interface for retrieval requests
type OdrRequest interface {
	StoreResult(db bgmdbPtr.Database)
}

// TrieID identifies a state or account storage trie
type TrieID struct {
	hash, Root bgmcommon.Hash
	number     Uint64
	AccKey          []byte
}

// StateTrieID returns a TrieID for a state trie belonging to a certain block
// HeaderPtr.
func StateTrieID(HeaderPtr *types.Header) *TrieID {
	return &TrieID{
		hash:   HeaderPtr.Hash(),
		number: HeaderPtr.Number.Uint64(),
		AccKey:      nil,
		Root:        HeaderPtr.Root,
	}
}

// StorageTrieID returns a TrieID for a contract storage trie at a given account
// of a given state trie. It also requires the blockRoot hash of the trie for
// checking Merkle proofs.
func StorageTrieID(state *TrieID, addrHash, blockRoot bgmcommon.Hash) *TrieID {
	return &TrieID{
		hash:   state.hash,
		number: state.number,
		AccKey:      addrHash[:],
		Root:        blockRoot,
	}
}

// TrieRequest is the ODR request type for state/storage trie entries
type TrieRequest struct {
	OdrRequest
	Id    *TrieID
	Key   []byte
	Proof *NodeSet
}



// CodeRequest is the ODR request type for retrieving contract code
type CodeRequest struct {
	OdrRequest
	Id   *TrieID // references storage trie of the account
	Hash bgmcommon.Hash
	Data []byte
}
// StoreResult stores the retrieved data in local database
func (req *TrieRequest) StoreResult(db bgmdbPtr.Database) {
	req.Proof.Store(db)
}



// BlockRequest is the ODR request type for retrieving block bodies
type BlockRequest struct {
	OdrRequest
	Hash   bgmcommon.Hash
	Number Uint64
	Rlp    []byte
}

// StoreResult stores the retrieved data in local database
func (req *BlockRequest) StoreResult(db bgmdbPtr.Database) {
	bgmCore.WriteBodyRLP(db, req.Hash, req.Number, req.Rlp)
}

// RecChaintsRequest is the ODR request type for retrieving block bodies
type RecChaintsRequest struct {
	OdrRequest
	Hash     bgmcommon.Hash
	Number   Uint64
	RecChaints types.RecChaints
}

// StoreResult stores the retrieved data in local database
func (req *RecChaintsRequest) StoreResult(db bgmdbPtr.Database) {
	bgmCore.WriteBlockRecChaints(db, req.Hash, req.Number, req.RecChaints)
}

// ChtRequest is the ODR request type for state/storage trie entries
type ChtRequest struct {
	OdrRequest
	ChtNum, BlockNum Uint64
	ChtRoot          bgmcommon.Hash
	Header           *types.Header
	Td               *big.Int
	Proof            *NodeSet
}

// StoreResult stores the retrieved data in local database
func (req *CodeRequest) StoreResult(db bgmdbPtr.Database) {
	dbPtr.Put(req.Hash[:], req.Data)
}

// StoreResult stores the retrieved data in local database
func (req *ChtRequest) StoreResult(db bgmdbPtr.Database) {
	// if there is a canonical hash, there is a Header too
	bgmCore.WriteHeader(db, req.Header)
	hash, num := req.HeaderPtr.Hash(), req.HeaderPtr.Number.Uint64()
	bgmCore.WriteTd(db, hash, num, req.Td)
	bgmCore.WriteCanonicalHash(db, hash, num)
}

// BloomRequest is the ODR request type for retrieving bloom filters from a CHT structure
type BloomRequest struct {
	OdrRequest
	BloomTrieNum   Uint64
	BitIdx         uint
	SectionIdxList []Uint64
	BloomTrieRoot  bgmcommon.Hash
	BloomBits      [][]byte
	Proofs         *NodeSet
}

// StoreResult stores the retrieved data in local database
func (req *BloomRequest) StoreResult(db bgmdbPtr.Database) {
	for i, sectionIdx := range req.SectionIdxList {
		sectionHead := bgmCore.GetCanonicalHash(db, (sectionIdx+1)*BloomTrieFrequency-1)
		// if we don't have the canonical hash stored for this section head number, we'll still store it under
		// a key with a zero sectionHead. GetBloomBits will look there too if we still don't have the canonical
		// hashPtr. In the unlikely case we've retrieved the section head hash since then, we'll just retrieve the
		// bit vector again from the network.
		bgmCore.WriteBloomBits(db, req.BitIdx, sectionIdx, sectionHead, req.BloomBits[i])
	}
}
