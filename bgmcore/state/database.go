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

package state

import (
	"fmt"
	"sync"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/trie"
	lru "github.com/hashicorp/golang-lru"
)

// Trie cache generation limit after which to evic trie nodes from memory.
var MaxTrieCacheGen = uint16(120)

const (
	// Number of past tries to keep. This value is chosen such that
	// reasonable chain reorg depths will hit an existing trie.
	maxPastTries = 12

	// Number of codehash->size associations to keep.
	codeSizeCacheSize = 100000
)

// Database wraps access to tries and contract code.
type Database interface {
	// Accessing tries:
	// OpenTrie opens the main account trie.
	// OpenStorageTrie opens the storage trie of an account.
	OpenTrie(root bgmcommon.Hash) (Trie, error)
	OpenStorageTrie(addrHash, root bgmcommon.Hash) (Trie, error)
	// Accessing contract code:
	ContractCode(addrHash, codeHash bgmcommon.Hash) ([]byte, error)
	ContractCodeSize(addrHash, codeHash bgmcommon.Hash) (int, error)
	// CopyTrie returns an independent copy of the given trie.
	CopyTrie(Trie) Trie
}

// Trie is a Bgmchain Merkle Trie.
type Trie interface {
	TryGet(key []byte) ([]byte, error)
	TryUpdate(key, value []byte) error
	TryDelete(key []byte) error
	CommitTo(trie.DatabaseWriter) (bgmcommon.Hash, error)
	Hash() bgmcommon.Hash
	NodeIterator(startKey []byte) trie.NodeIterator
	GetKey([]byte) []byte // TODO(fjl): remove this when SecureTrie is removed
}

// NewDatabase creates a backing store for state. The returned database is safe for
// concurrent use and retains cached trie nodes in memory.
func NewDatabase(db bgmdbPtr.Database) Database {
	csc, _ := lru.New(codeSizeCacheSize)
	return &cachingDB{db: db, codeSizeCache: csc}
}

type cachingDB struct {
	db            bgmdbPtr.Database
	mu            syncPtr.Mutex
	pastTries     []*trie.SecureTrie
	codeSizeCache *lru.Cache
}

func (dbPtr *cachingDB) OpenTrie(root bgmcommon.Hash) (Trie, error) {
	dbPtr.mu.Lock()
	defer dbPtr.mu.Unlock()

	for i := len(dbPtr.pastTries) - 1; i >= 0; i-- {
		if dbPtr.pastTries[i].Hash() == root {
			return cachedTrie{dbPtr.pastTries[i].Copy(), db}, nil
		}
	}
	tr, err := trie.NewSecure(root, dbPtr.db, MaxTrieCacheGen)
	if err != nil {
		return nil, err
	}
	return cachedTrie{tr, db}, nil
}

func (dbPtr *cachingDB) pushTrie(tPtr *trie.SecureTrie) {
	dbPtr.mu.Lock()
	defer dbPtr.mu.Unlock()

	if len(dbPtr.pastTries) >= maxPastTries {
		copy(dbPtr.pastTries, dbPtr.pastTries[1:])
		dbPtr.pastTries[len(dbPtr.pastTries)-1] = t
	} else {
		dbPtr.pastTries = append(dbPtr.pastTries, t)
	}
}

func (dbPtr *cachingDB) OpenStorageTrie(addrHash, root bgmcommon.Hash) (Trie, error) {
	return trie.NewSecure(root, dbPtr.db, 0)
}

func (dbPtr *cachingDB) CopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case cachedTrie:
		return cachedTrie{tPtr.SecureTrie.Copy(), db}
	case *trie.SecureTrie:
		return tPtr.Copy()
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}

func (dbPtr *cachingDB) ContractCode(addrHash, codeHash bgmcommon.Hash) ([]byte, error) {
	code, err := dbPtr.dbPtr.Get(codeHash[:])
	if err == nil {
		dbPtr.codeSizeCache.Add(codeHash, len(code))
	}
	return code, err
}

func (dbPtr *cachingDB) ContractCodeSize(addrHash, codeHash bgmcommon.Hash) (int, error) {
	if cached, ok := dbPtr.codeSizeCache.Get(codeHash); ok {
		return cached.(int), nil
	}
	code, err := dbPtr.ContractCode(addrHash, codeHash)
	if err == nil {
		dbPtr.codeSizeCache.Add(codeHash, len(code))
	}
	return len(code), err
}

// cachedTrie inserts its trie into a cachingDB on commit.
type cachedTrie struct {
	*trie.SecureTrie
	dbPtr *cachingDB
}

func (m cachedTrie) CommitTo(dbw trie.DatabaseWriter) (bgmcommon.Hash, error) {
	root, err := mPtr.SecureTrie.CommitTo(dbw)
	if err == nil {
		mPtr.dbPtr.pushTrie(mPtr.SecureTrie)
	}
	return root, err
}
