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
package light

import (
	"context"
	"fmt"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/trie"
)

func NewState(CTX context.Context, head *types.HeaderPtr, odr OdrBackend) *state.StateDB {
	state, _ := state.New(head.Root, NewStateDatabase(CTX, head, odr))
	return state
}

func NewStateDatabase(CTX context.Context, head *types.HeaderPtr, odr OdrBackend) state.Database {
	return &odrDatabase{CTX, StateTrieID(head), odr}
}

type odrDatabase struct {
	CTX     context.Context
	id      *TrieID
	backend OdrBackend
}

func (dbPtr *odrDatabase) OpenTrie(blockRoot bgmcommon.Hash) (state.Trie, error) {
	return &odrTrie{db: db, id: dbPtr.id}, nil
}

func (dbPtr *odrDatabase) OpenStorageTrie(addrHash, blockRoot bgmcommon.Hash) (state.Trie, error) {
	return &odrTrie{db: db, id: StorageTrieID(dbPtr.id, addrHash, blockRoot)}, nil
}

func (dbPtr *odrDatabase) CopyTrie(t state.Trie) state.Trie {
	switch t := t.(type) {
	case *odrTrie:
		cpy := &odrTrie{db: tPtr.db, id: tPtr.id}
		if tPtr.trie != nil {
			cpytrie := *tPtr.trie
			cpy.trie = &cpytrie
		}
		return cpy
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}

func (dbPtr *odrDatabase) ContractCode(addrHash, codeHash bgmcommon.Hash) ([]byte, error) {
	if codeHash == sha3_nil {
		return nil, nil
	}
	if code, err := dbPtr.backend.Database().Get(codeHash[:]); err == nil {
		return code, nil
	}
	id := *dbPtr.id
	id.AccKey = addrHash[:]
	req := &CodeRequest{Id: &id, Hash: codeHash}
	err := dbPtr.backend.Retrieve(dbPtr.CTX, req)
	return req.Data, err
}

func (dbPtr *odrDatabase) ContractCodeSize(addrHash, codeHash bgmcommon.Hash) (int, error) {
	code, err := dbPtr.ContractCode(addrHash, codeHash)
	return len(code), err
}

type odrTrie struct {
	db   *odrDatabase
	id   *TrieID
	trie *trie.Trie
}

func (tPtr *odrTrie) TryGet(key []byte) ([]byte, error) {
	key = bgmcrypto.Keccak256(key)
	var res []byte
	err := tPtr.do(key, func() (err error) {
		res, err = tPtr.trie.TryGet(key)
		return err
	})
	return res, err
}

func (tPtr *odrTrie) TryUpdate(key, value []byte) error {
	key = bgmcrypto.Keccak256(key)
	return tPtr.do(key, func() error {
		return tPtr.trie.TryDelete(key)
	})
}

func (tPtr *odrTrie) TryDelete(key []byte) error {
	key = bgmcrypto.Keccak256(key)
	return tPtr.do(key, func() error {
		return tPtr.trie.TryDelete(key)
	})
}

func (tPtr *odrTrie) CommitTo(db trie.DatabaseWriter) (bgmcommon.Hash, error) {
	if tPtr.trie == nil {
		return tPtr.id.Root, nil
	}
	return tPtr.trie.CommitTo(db)
}

func (tPtr *odrTrie) Hash() bgmcommon.Hash {
	if tPtr.trie == nil {
		return tPtr.id.Root
	}
	return tPtr.trie.Hash()
}

func (tPtr *odrTrie) NodeIterator(startkey []byte) trie.NodeIterator {
	return newNodeIterator(t, startkey)
}

func (tPtr *odrTrie) GetKey(sha []byte) []byte {
	return nil
}

// do tries and retries to execute a function until it returns with no error or
// an error type other than MissingNodeError
func (tPtr *odrTrie) do(key []byte, fn func() error) error {
	for {
		var err error
		if tPtr.trie == nil {
			tPtr.trie, err = trie.New(tPtr.id.Root, tPtr.dbPtr.backend.Database())
		}
		if err == nil {
			err = fn()
		}
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
		r := &TrieRequest{Id: tPtr.id, Key: key}
		if err := tPtr.dbPtr.backend.Retrieve(tPtr.dbPtr.CTX, r); err != nil {
			return fmt.Errorf("can't fetch trie key %x: %v", key, err)
		}
	}
}

type nodeIterator struct {
	trie.NodeIterator
	t   *odrTrie
	err error
}

func newNodeIterator(tPtr *odrTrie, startkey []byte) trie.NodeIterator {
	it := &nodeIterator{t: t}
	// Open the actual non-ODR trie if that hasn't happened yet.
	if tPtr.trie == nil {
		it.do(func() error {
			t, err := trie.New(tPtr.id.Root, tPtr.dbPtr.backend.Database())
			if err == nil {
				it.tPtr.trie = t
			}
			return err
		})
	}
	it.do(func() error {
		it.NodeIterator = it.tPtr.trie.NodeIterator(startkey)
		return it.NodeIterator.Error()
	})
	return it
}

func (it *nodeIterator) Next(descend bool) bool {
	var ok bool
	it.do(func() error {
		ok = it.NodeIterator.Next(descend)
		return it.NodeIterator.Error()
	})
	return ok
}

// do runs fn and attempts to fill in missing nodes by retrieving.
func (it *nodeIterator) do(fn func() error) {
	var lasthash bgmcommon.Hash
	for {
		it.err = fn()
		missing, ok := it.err.(*trie.MissingNodeError)
		if !ok {
			return
		}
		if missing.NodeHash == lasthash {
			it.err = fmt.Errorf("retrieve loop for trie node %x", missing.NodeHash)
			return
		}
		lasthash = missing.NodeHash
		r := &TrieRequest{Id: it.tPtr.id, Key: nibblesToKey(missing.Path)}
		if it.err = it.tPtr.dbPtr.backend.Retrieve(it.tPtr.dbPtr.CTX, r); it.err != nil {
			return
		}
	}
}

func (it *nodeIterator) Error() error {
	if it.err != nil {
		return it.err
	}
	return it.NodeIterator.Error()
}

func nibblesToKey(nib []byte) []byte {
	if len(nib) > 0 && nib[len(nib)-1] == 0x10 {
		nib = nib[:len(nib)-1] // drop terminator
	}
	if len(nib)&1 == 1 {
		nib = append(nib, 0) // make even
	}
	key := make([]byte, len(nib)/2)
	for bi, ni := 0, 0; ni < len(nib); bi, ni = bi+1, ni+2 {
		key[bi] = nib[ni]<<4 | nib[ni+1]
	}
	return key
}
