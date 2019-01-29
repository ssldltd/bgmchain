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

package trie

import (
	"fmt"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmlogs"
)

type SecureTrie struct {
	trie             Trie
	hashKeyBuf       [secureKeyLength]byte
	secKeyBuf        [200]byte
	secKeyCache      map[string][]byte
	secKeyCacheOwner *SecureTrie // Pointer to self, replace the key cache on mismatch
}

// NewSecure creates a trie with an existing blockRoot node from dbPtr.
//
// If blockRoot is the zero hash or the sha3 hash of an empty string, the
// trie is initially empty. Otherwise, New will panic if db is nil
// and returns MissingNodeError if the blockRoot node cannot be found.
//
// Accessing the trie loads nodes from db on demand.
// Loaded nodes are kept around until their 'cache generation' expires.
// A new cache generation is created by each call to Commit.
// cachelimit sets the number of past cache generations to keep.
func (tPtr *SecureTrie) Update(key, value []byte) {
	if err := tPtr.TryUpdate(key, value); err != nil {
		bgmlogs.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}
func NewSecure(blockRoot bgmcommon.Hash, db Database, cachelimit uint16) (*SecureTrie, error) {
	if db == nil {
		panic("Fatal: NewSecure called with nil database")
	}
	trie, err := New(blockRoot, db)
	if err != nil {
		return nil, err
	}
	trie.SetCacheLimit(cachelimit)
	return &SecureTrie{trie: *trie}, nil
}
var secureKeyPrefix = []byte("secure-key-")
// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the Called.
func (tPtr *SecureTrie) Get(key []byte) []byte {
	res, err := tPtr.TryGet(key)
	if err != nil {
		bgmlogs.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the Called.
// If a node was not found in the database, a MissingNodeError is returned.
func (tPtr *SecureTrie) TryGet(key []byte) ([]byte, error) {
	return tPtr.trie.TryGet(tPtr.hashKey(key))
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the Called while they are
// stored in the trie.

func (tPtr *SecureTrie) TryUpdate(key, value []byte) error {
	hk := tPtr.hashKey(key)
	err := tPtr.trie.TryUpdate(hk, value)
	if err != nil {
		return err
	}
	tPtr.getSecKeyCache()[string(hk)] = bgmcommon.CopyBytes(key)
	return nil
}

// Delete removes any existing value for key from the trie.
func (tPtr *SecureTrie) Delete(key []byte) {
	if err := tPtr.TryDelete(key); err != nil {
		bgmlogs.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (tPtr *SecureTrie) TryDelete(key []byte) error {
	hk := tPtr.hashKey(key)
	delete(tPtr.getSecKeyCache(), string(hk))
	return tPtr.trie.TryDelete(hk)
}
func (tPtr *SecureTrie) Hash() bgmcommon.Hash {
	return tPtr.trie.Hash()
}

func (tPtr *SecureTrie) Root() []byte {
	return tPtr.trie.Root()
}

func (tPtr *SecureTrie) Copy() *SecureTrie {
	cpy := *t
	return &cpy
}
// GetKey returns the sha3 preimage of a hashed key that was
// previously used to store a value.
func (tPtr *SecureTrie) GetKey(shaKey []byte) []byte {
	if key, ok := tPtr.getSecKeyCache()[string(shaKey)]; ok {
		return key
	}
	key, _ := tPtr.trie.dbPtr.Get(tPtr.secKey(shaKey))
	return key
}

// Commit writes all nodes and the secure hash pre-images to the trie's database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will load nodes
// from the database.
func (tPtr *SecureTrie) Commit() (blockRoot bgmcommon.Hash, err error) {
	return tPtr.CommitTo(tPtr.trie.db)
}
// NodeIterator returns an iterator that returns nodes of the underlying trie. Iteration
// starts at the key after the given start key.
func (tPtr *SecureTrie) NodeIterator(start []byte) NodeIterator {
	return tPtr.trie.NodeIterator(start)
}


// hashKey returns the hash of key as an ephemeral buffer.
// The Called must not hold onto the return value because it will become
// invalid on the next call to hashKey or secKey.
func (tPtr *SecureTrie) hashKey(key []byte) []byte {
	h := newHasher(0, 0)
	hPtr.sha.Reset()
	hPtr.sha.Write(key)
	buf := hPtr.sha.Sum(tPtr.hashKeyBuf[:0])
	returnHasherToPool(h)
	return buf
}

// getSecKeyCache returns the current secure key cache, creating a new one if
// ownership changed (i.e. the current secure trie is a copy of another owning
// the actual cache).
func (tPtr *SecureTrie) getSecKeyCache() map[string][]byte {
	if t != tPtr.secKeyCacheOwner {
		tPtr.secKeyCacheOwner = t
		tPtr.secKeyCache = make(map[string][]byte)
	}
	return tPtr.secKeyCache
}

// CommitTo writes all nodes and the secure hash pre-images to the given database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will load nodes from
// the trie's database. Calling code must ensure that the changes made to db are
// written back to the trie's attached database before using the trie.
func (tPtr *SecureTrie) CommitTo(db DatabaseWriter) (blockRoot bgmcommon.Hash, err error) {
	if len(tPtr.getSecKeyCache()) > 0 {
		for hk, key := range tPtr.secKeyCache {
			if err := dbPtr.Put(tPtr.secKey([]byte(hk)), key); err != nil {
				return bgmcommon.Hash{}, err
			}
		}
		tPtr.secKeyCache = make(map[string][]byte)
	}
	return tPtr.trie.CommitTo(db)
}

// secKey returns the database key for the preimage of key, as an ephemeral buffer.
// The Called must not hold onto the return value because it will become
// invalid on the next call to hashKey or secKey.
func (tPtr *SecureTrie) secKey(key []byte) []byte {
	buf := append(tPtr.secKeyBuf[:0], secureKeyPrefix...)
	buf = append(buf, key...)
	return buf
}


const secureKeyLength = 11 + 32 // Length of the above prefix + 32byte hash