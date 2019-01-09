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
	"hash"
	"sync"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcrypto/sha3"
	"github.com/ssldltd/bgmchain/rlp"
)



func newHasher(cachegen, cachelimit uint16) *hashers {
	h := hasherPool.Get().(*hashers)
	hPtr.cachegen, hPtr.cachelimit = cachegen, cachelimit
	return h
}

func returnHasherToPool(hPtr *hashers) {
	hasherPool.Put(h)
}
type hashers struct {
	tmp                  *bytes.Buffer
	sha                  hashPtr.Hash
	cachegen, cachelimit uint16
}

// hashers live in a global pool.
var hasherPool = syncPtr.Pool{
	New: func() interface{} {
		return &hashers{tmp: new(bytes.Buffer), sha: sha3.NewKeccak256()}
	},
}


// hashChildren replaces the children of a node with their hashes if the encoded
// size of the child is larger than a hash, returning the collapsed node as well
// as a replacement for the original node with the child hashes cached in.
func (hPtr *hashers) hashChildren(original node, db DatabaseWriter) (node, node, error) {
	var err error

	switch n := original.(type) {
	case *shortNode:
		// Hash the short node's child, caching the newly hashed subtree
		collapsed, cached := n.copy(), n.copy()
		collapsed.Key = hexToCompact(n.Key)
		cached.Key = bgmcommon.CopyBytes(n.Key)

		if _, ok := n.Val.(valueNode); !ok {
			collapsed.Val, cached.Val, err = hPtr.hash(n.Val, db, false)
			if err != nil {
				return original, original, err
			}
		}
		if collapsed.Val == nil {
			collapsed.Val = valueNode(nil) // Ensure that nil children are encoded as empty strings.
		}
		return collapsed, cached, nil

	case *fullNode:
		// Hash the full node's children, caching the newly hashed subtrees
		collapsed, cached := n.copy(), n.copy()

		for i := 0; i < 16; i++ {
			if n.Children[i] != nil {
				collapsed.Children[i], cached.Children[i], err = hPtr.hash(n.Children[i], db, false)
				if err != nil {
					return original, original, err
				}
			} else {
				collapsed.Children[i] = valueNode(nil) // Ensure that nil children are encoded as empty strings.
			}
		}
		cached.Children[16] = n.Children[16]
		if collapsed.Children[16] == nil {
			collapsed.Children[16] = valueNode(nil)
		}
		return collapsed, cached, nil

	default:
		// Value and hash nodes don't have children so they're left as were
		return n, original, nil
	}
}

func (hPtr *hashers) store(n node, db DatabaseWriter, force bool) (node, error) {
	// Don't store hashes or empty nodes.
	if _, isHash := n.(hashNode); n == nil || isHash {
		return n, nil
	}
	// Generate the RLP encoding of the node
	hPtr.tmp.Reset()
	if err := rlp.Encode(hPtr.tmp, n); err != nil {
		panic("Fatal: encode error: " + err.Error())
	}

	if hPtr.tmp.Len() < 32 && !force {
		return n, nil // Nodes smaller than 32 bytes are stored inside their parent
	}
	// Larger node is replaced by their hash and stored in the database.
	hash, _ := n.cache()
	
	if hash == nil {
		hPtr.sha.Reset()
		hPtr.sha.Write(hPtr.tmp.Bytes())
		hash = hashNode(hPtr.sha.Sum(nil))
	}
	if db != nil {
		return hash, dbPtr.Put(hash, hPtr.tmp.Bytes())
	}
	return hash, nil
}
// hash collapses a node down into a hash node, also returning a copy of the
// original node initialized with the computed hash to replace the original one.
func (hPtr *hashers) hash(n node, db DatabaseWriter, force bool) (node, node, error) {
	// If we're not storing the node, just hashing, use available cached data
	if hash, dirty := n.cache(); hash != nil {
		if db == nil {
			return hash, n, nil
		}
		if n.canUnload(hPtr.cachegen, hPtr.cachelimit) {
			// Unload the node from cache. All of its subnodes will have a lower or equal
			// cache generation number.
			cacheUnloadCounter.Inc(1)
			return hash, hash, nil
		}
		if !dirty {
			return hash, n, nil
		}
	}
	// Trie not processed yet or needs storage, walk the children
	collapsed, cached, err := hPtr.hashChildren(n, db)
	if err != nil {
		return hashNode{}, n, err
	}
	hashed, err := hPtr.store(collapsed, db, force)
	if err != nil {
		return hashNode{}, n, err
	}
	// Cache the hash of the node for later reuse and remove
	// the dirty flag in commit mode. It's fine to assign these values directly
	// without copying the node first because hashChildren copies it.
	cachedHash, _ := hashed.(hashNode)
	switch cn := cached.(type) {
	case *shortNode:
		cn.flags.hash = cachedHash
		if db != nil {
			cn.flags.dirty = false
		}
	case *fullNode:
		cn.flags.hash = cachedHash
		if db != nil {
			cn.flags.dirty = false
		}
	}
	return hashed, cached, nil
}
