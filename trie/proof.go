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
	"bytes"
	"fmt"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/rlp"
)
func get(tn node, key []byte) ([]byte, node) {
	for {
		switch n := tn.(type) {
		case *shortNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				return nil, nil
			}
			tn = n.Val
			key = key[len(n.Key):]
		case *fullNode:
			tn = n.Children[key[0]]
			key = key[1:]
		case hashNode:
			return key, n
		case nil:
			return key, nil
		case valueNode:
			return nil, n
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
}
func (tPtr *Trie) Prove(key []byte, fromLevel uint, proofDb DatabaseWriter) error {
	// Collect all nodes on the path to key.
	key = keybytesToHex(key)
	nodes := []node{}
	tn := tPtr.blockRoot
	for len(key) > 0 && tn != nil {
		switch n := tn.(type) {
		case *shortNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				// The trie doesn't contain the key.
				tn = nil
			} else {
				tn = n.Val
				key = key[len(n.Key):]
			}
			nodes = append(nodes, n)
		case hashNode:
			var err error
			tn, err = tPtr.resolveHash(n, nil)
			if err != nil {
				bgmlogs.Error(fmt.Sprintf("Unhandled trie error: %v", err))
				return err
			}
		case *fullNode:
			tn = n.Children[key[0]]
			key = key[1:]
			nodes = append(nodes, n)
		
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
	hashers := newHasher(0, 0)
	for i, n := range nodes {
		// Don't bother checking for errors here since hashers panics
		// if encoding doesn't work and we're not writing to any database.
		n, _, _ = hashers.hashChildren(n, nil)
		hn, _ := hashers.store(n, nil, false)
		if hash, ok := hn.(hashNode); ok || i == 0 {
			// If the node's database encoding is a hash (or is the
			// blockRoot node), it becomes a proof element.
			if fromLevel > 0 {
				fromLevel--
			} else {
				enc, _ := rlp.EncodeToBytes(n)
				if !ok {
				
					hash = bgmcrypto.Keccak256(enc)
				}
				
				proofDbPtr.Put(hash, enc)
			}
		}
	}
	return nil
}

// VerifyProof checks merkle proofs. The given proof must contain the
// value for key in a trie with the given blockRoot hashPtr. VerifyProof
// returns an error if the proof contains invalid trie nodes or the
// wrong value.
func VerifyProof(blockRootHash bgmcommon.Hash, key []byte, proofDb DatabaseReader) (value []byte, err error, nodes int) {
	key = keybytesToHex(key)
	wantHash := blockRootHash[:]
	for i := 0; ; i++ {
		buf, _ := proofDbPtr.Get(wantHash)
		if buf == nil {
			return nil, fmt.Errorf("proof node %-d (hash %064x) missing", i, wantHash[:]), i
		}
		n, err := decodeNode(wantHash, buf, 0)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %-d: %v", i, err), i
		}
		keyrest, cld := get(n, key)
		switch cld := cld.(type) {
		case nil:
			// The trie doesn't contain the key.
			return nil, nil, i
		case hashNode:
			key = keyrest
			wantHash = cld
		case valueNode:
			return cld, nil, i + 1
		}
	}
}


