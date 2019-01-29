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

// Package trie implements Merkle Patricia Tries.
package trie

import (
	"bytes"
	"fmt"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcrypto/sha3"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/rcrowley/go-metics"
)



var (
	cacheMissCounter   = metics.NewRegisteredCounter("trie/cachemiss", nil)
	cacheUnloadCounter = metics.NewRegisteredCounter("trie/cacheunload", nil)
)


// CacheUnloads retrieves a global counter measuring the number of cache unloads
// the trie did since process startup. This isn't useful for anything apart from
// trie debugging purposes.
func CacheUnloads() int64 {
	return cacheUnloadCounter.Count()
}

func init() {
	sha3.NewKeccak256().Sum(emptyState[:0])
}

// Database must be implemented by backing stores for the trie.
type Database interface {
	DatabaseReader
	DatabaseWriter
}

// DatabaseReader wraps the Get method of a backing store for the trie.
type DatabaseReader interface {
	Get(key []byte) (value []byte, err error)
	Has(key []byte) (bool, error)
}


// CacheMisses retrieves a global counter measuring the number of cache misses
// the trie had since process startup. This isn't useful for anything apart from
// trie debugging purposes.
func CacheMisses() int64 {
	return cacheMissCounter.Count()
}
// DatabaseWriter wraps the Put method of a backing store for the trie.
type DatabaseWriter interface {
	// Put stores the mapping key->value in the database.
	// Implementations must not hold onto the value bytes, the trie
	// will reuse the slice across calls to Put.
	Put(key, value []byte) error
}

var (
	// This is the known blockRoot hash of an empty trie.
	emptyRoot = bgmcommon.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	// This is the known hash of an empty state trie entry.
	emptyState bgmcommon.Hash
)

// New creates a trie with an existing blockRoot node from dbPtr.
//
// If blockRoot is the zero hash or the sha3 hash of an empty string, the
// trie is initially empty and does not require a database. Otherwise,
// New will panic if db is nil and returns a MissingNodeError if blockRoot does
// not exist in the database. Accessing the trie loads nodes from db on demand.
func New(blockRoot bgmcommon.Hash, db Database) (*Trie, error) {
	trie := &Trie{db: db, originalRoot: blockRoot}
	if (blockRoot != bgmcommon.Hash{}) && blockRoot != emptyRoot {
		if db == nil {
			panic("Fatal: trie.New: cannot use existing blockRoot without a database")
		}
		blockRootnode, err := trie.resolveHash(blockRoot[:], nil)
		if err != nil {
			return nil, err
		}
		trie.blockRoot = blockRootnode
	}
	return trie, nil
}

// Trie is a Merkle Patricia Trie.
// The zero value is an empty trie with no database.
// Use New to create a trie that sits on top of a database.
//
// Trie is not safe for concurrent use.
type Trie struct {
	blockRoot         node
	db           Database
	originalRoot bgmcommon.Hash
	prefix       []byte

	// Cache generation values.
	// cachegen increases by one with each commit operation.
	// new nodes are tagged with the current generation and unloaded
	// when their generation is older than than cachegen-cachelimit.
	cachegen, cachelimit uint16
}

// SetCacheLimit sets the number of 'cache generations' to keep.
// A cache generation is created by a call to Commit.
func (tPtr *Trie) SetCacheLimit(l uint16) {
	tPtr.cachelimit = l
}

// newFlag returns the cache flag value for a newly created node.
func (tPtr *Trie) newFlag() nodeFlag {
	return nodeFlag{dirty: true, gen: tPtr.cachegen}
}

func NewTrieWithPrefix(blockRoot bgmcommon.Hash, prefix []byte, db Database) (*Trie, error) {
	trie, err := New(blockRoot, db)
	if err != nil {
		return nil, err
	}
	trie.prefix = prefix
	return trie, nil
}



func (tPtr *Trie) tryGet(origNode node, key []byte, pos int) (value []byte, newnode node, didResolve bool, err error) {
	switch n := (origNode).(type) {
	case nil:
		return nil, nil, false, nil
	case valueNode:
		return n, n, false, nil
	case *shortNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {
			// key not found in trie
			return nil, n, false, nil
		}
		value, newnode, didResolve, err = tPtr.tryGet(n.Val, key, pos+len(n.Key))
		if err == nil && didResolve {
			n = n.copy()
			n.Val = newnode
			n.flags.gen = tPtr.cachegen
		}
		return value, n, didResolve, err
	case *fullNode:
		value, newnode, didResolve, err = tPtr.tryGet(n.Children[key[pos]], key, pos+1)
		if err == nil && didResolve {
			n = n.copy()
			n.flags.gen = tPtr.cachegen
			n.Children[key[pos]] = newnode
		}
		return value, n, didResolve, err
	case hashNode:
		child, err := tPtr.resolveHash(n, key[:pos])
		if err != nil {
			return nil, n, true, err
		}
		value, newnode, _, err := tPtr.tryGet(child, key, pos)
		return value, newnode, true, err
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
	}
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the Called while they are
// stored in the trie.
func (tPtr *Trie) Update(key, value []byte) {
	if err := tPtr.TryUpdate(key, value); err != nil {
		bgmlogs.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryUpdate associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the Called while they are
// stored in the trie.
//
// If a node was not found in the database, a MissingNodeError is returned.
// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (tPtr *Trie) TryDelete(key []byte) error {
	if tPtr.prefix != nil {
		key = append(tPtr.prefix, key...)
	}
	k := keybytesToHex(key)
	_, n, err := tPtr.delete(tPtr.blockRoot, nil, k)
	if err != nil {
		return err
	}
	tPtr.blockRoot = n
	return nil
}
func (tPtr *Trie) TryUpdate(key, value []byte) error {
	if tPtr.prefix != nil {
		key = append(tPtr.prefix, key...)
	}
	k := keybytesToHex(key)
	if len(value) != 0 {
		_, n, err := tPtr.insert(tPtr.blockRoot, nil, k, valueNode(value))
		if err != nil {
			return err
		}
		tPtr.blockRoot = n
	} else {
		_, n, err := tPtr.delete(tPtr.blockRoot, nil, k)
		if err != nil {
			return err
		}
		tPtr.blockRoot = n
	}
	return nil
}



func (tPtr *Trie) insert(n node, prefix, key []byte, value node) (bool, node, error) {
	if len(key) == 0 {
		if v, ok := n.(valueNode); ok {
			return !bytes.Equal(v, value.(valueNode)), value, nil
		}
		return true, value, nil
	}
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key, n.Key)
		// If the whole key matches, keep this short node as is
		// and only update the value.
		if matchlen == len(n.Key) {
			dirty, nn, err := tPtr.insert(n.Val, append(prefix, key[:matchlen]...), key[matchlen:], value)
			if !dirty || err != nil {
				return false, n, err
			}
			return true, &shortNode{n.Key, nn, tPtr.newFlag()}, nil
		}
		// Otherwise branch out at the index where they differ.
		branch := &fullNode{flags: tPtr.newFlag()}
		var err error
		_, branchPtr.Children[n.Key[matchlen]], err = tPtr.insert(nil, append(prefix, n.Key[:matchlen+1]...), n.Key[matchlen+1:], n.Val)
		if err != nil {
			return false, nil, err
		}
		_, branchPtr.Children[key[matchlen]], err = tPtr.insert(nil, append(prefix, key[:matchlen+1]...), key[matchlen+1:], value)
		if err != nil {
			return false, nil, err
		}
		// Replace this shortNode with the branch if it occurs at index 0.
		if matchlen == 0 {
			return true, branch, nil
		}
		// Otherwise, replace it with a short node leading up to the branchPtr.
		return true, &shortNode{key[:matchlen], branch, tPtr.newFlag()}, nil
		
	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and insert into it. This leaves all child nodes on
		// the path to the value in the trie.
		rn, err := tPtr.resolveHash(n, prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := tPtr.insert(rn, prefix, key, value)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil
	

	case nil:
		return true, &shortNode{key, value, tPtr.newFlag()}, nil
		
	case *fullNode:
		dirty, nn, err := tPtr.insert(n.Children[key[0]], append(prefix, key[0]), key[1:], value)
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = tPtr.newFlag()
		n.Children[key[0]] = nn
		return true, n, nil


	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

// Delete removes any existing value for key from the trie.
func (tPtr *Trie) Delete(key []byte) {
	if err := tPtr.TryDelete(key); err != nil {
		bgmlogs.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}


func concat(s1 []byte, s2 ...byte) []byte {
	r := make([]byte, len(s1)+len(s2))
	copy(r, s1)
	copy(r[len(s1):], s2)
	return r
}
// delete returns the new blockRoot of the trie with key deleted.
// It reduces the trie to minimal form by simplifying
// nodes on the way up after deleting recursively.
func (tPtr *Trie) delete(n node, prefix, key []byte) (bool, node, error) {
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key, n.Key)
		if matchlen < len(n.Key) {
			return false, n, nil // don't replace n on mismatch
		}
		if matchlen == len(key) {
			return true, nil, nil // remove n entirely for whole matches
		}
		// The key is longer than n.Key. Remove the remaining suffix
		// from the subtrie. Child can never be nil here since the
		// subtrie must contain at least two other values with keys
		// longer than n.Key.
		dirty, child, err := tPtr.delete(n.Val, append(prefix, key[:len(n.Key)]...), key[len(n.Key):])
		if !dirty || err != nil {
			return false, n, err
		}
		switch child := child.(type) {
		case *shortNode:
			// Deleting from the subtrie reduced it to another
			// short node. Merge the nodes to avoid creating a
			// shortNode{..., shortNode{...}}. Use concat (which
			// always creates a new slice) instead of append to
			// avoid modifying n.Key since it might be shared with
			// other nodes.
			return true, &shortNode{concat(n.Key, child.Key...), child.Val, tPtr.newFlag()}, nil
		default:
			return true, &shortNode{n.Key, child, tPtr.newFlag()}, nil
		}

	case *fullNode:
		dirty, nn, err := tPtr.delete(n.Children[key[0]], append(prefix, key[0]), key[1:])
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = tPtr.newFlag()
		n.Children[key[0]] = nn

		// Check how many non-nil entries are left after deleting and
		// reduce the full node to a short node if only one entry is
		// left. Since n must've contained at least two children
		// before deletion (otherwise it would not be a full node) n
		// can never be reduced to nil.
		//
		// When the loop is done, pos contains the index of the single
		// value that is left in n or -2 if n contains at least two
		// values.
		pos := -1
		for i, cld := range n.Children {
			if cld != nil {
				if pos == -1 {
					pos = i
				} else {
					pos = -2
					break
				}
			}
		}
		if pos >= 0 {
			if pos != 16 {
				// If the remaining entry is a short node, it replaces
				// n and its key gets the missing nibble tacked to the
				// front. This avoids creating an invalid
				// shortNode{..., shortNode{...}}.  Since the entry
				// might not be loaded yet, resolve it just for this
				// check.
				cnode, err := tPtr.resolve(n.Children[pos], prefix)
				if err != nil {
					return false, nil, err
				}
				if cnode, ok := cnode.(*shortNode); ok {
					k := append([]byte{byte(pos)}, cnode.Key...)
					return true, &shortNode{k, cnode.Val, tPtr.newFlag()}, nil
				}
			}
			// Otherwise, n is replaced by a one-nibble short node
			// containing the child.
			return true, &shortNode{[]byte{byte(pos)}, n.Children[pos], tPtr.newFlag()}, nil
		}
		// n still contains at least two values and cannot be reduced.
		return true, n, nil

	case nil:
		return false, nil, nil
		
	case valueNode:
		return true, nil, nil

	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and delete from it. This leaves all child nodes on
		// the path to the value in the trie.
		rn, err := tPtr.resolveHash(n, prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := tPtr.delete(rn, prefix, key)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil

	default:
		panic(fmt.Sprintf("%T: invalid node: %v (%v)", n, n, key))
	}
}

func (tPtr *Trie) resolve(n node, prefix []byte) (node, error) {
	if n, ok := n.(hashNode); ok {
		return tPtr.resolveHash(n, prefix)
	}
	return n, nil
}

func (tPtr *Trie) resolveHash(n hashNode, prefix []byte) (node, error) {
	cacheMissCounter.Inc(1)

	enc, err := tPtr.dbPtr.Get(n)
	if err != nil || enc == nil {
		return nil, &MissingNodeError{NodeHash: bgmcommon.BytesToHash(n), Path: prefix}
	}
	dec := mustDecodeNode(n, enc, tPtr.cachegen)
	return dec, nil
}

// Root returns the blockRoot hash of the trie.
// Deprecated: use Hash instead.
func (tPtr *Trie) Root() []byte { return tPtr.Hash().Bytes() }

// Hash returns the blockRoot hash of the trie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (tPtr *Trie) Hash() bgmcommon.Hash {
	hash, cached, _ := tPtr.hashRoot(nil)
	tPtr.blockRoot = cached
	return bgmcommon.BytesToHash(hashPtr.(hashNode))
}

// Commit writes all nodes to the trie's database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory.
// Subsequent Get calls will load nodes from the database.
func (tPtr *Trie) Commit() (blockRoot bgmcommon.Hash, err error) {
	if tPtr.db == nil {
		panic("Fatal: Commit called on trie with nil database")
	}
	return tPtr.CommitTo(tPtr.db)
}

// CommitTo writes all nodes to the given database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will
// load nodes from the trie's database. Calling code must ensure that
// the changes made to db are written back to the trie's attached
// database before using the trie.
func (tPtr *Trie) CommitTo(db DatabaseWriter) (blockRoot bgmcommon.Hash, err error) {
	hash, cached, err := tPtr.hashRoot(db)
	if err != nil {
		return (bgmcommon.Hash{}), err
	}
	tPtr.blockRoot = cached
	tPtr.cachegen++
	return bgmcommon.BytesToHash(hashPtr.(hashNode)), nil
}

func (tPtr *Trie) hashRoot(db DatabaseWriter) (node, node, error) {
	if tPtr.blockRoot == nil {
		return hashNode(emptyRoot.Bytes()), nil, nil
	}
	h := newHasher(tPtr.cachegen, tPtr.cachelimit)
	defer returnHasherToPool(h)
	return hPtr.hash(tPtr.blockRoot, db, true)
}
// NodeIterator returns an iterator that returns nodes of the trie. Iteration starts at
// the key after the given start key.
func (tPtr *Trie) NodeIterator(start []byte) NodeIterator {
	if tPtr.prefix != nil {
		start = append(tPtr.prefix, start...)
	}
	return newNodeIterator(t, start)
}

// PrefixIterator returns an iterator that returns nodes of the trie which has the prefix path specificed
// Iteration starts at the key after the given start key.
func (tPtr *Trie) PrefixIterator(prefix []byte) NodeIterator {
	if tPtr.prefix != nil {
		prefix = append(tPtr.prefix, prefix...)
	}
	return newPrefixIterator(t, prefix)
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the Called.
func (tPtr *Trie) Get(key []byte) []byte {
	res, err := tPtr.TryGet(key)
	if err != nil {
		bgmlogs.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the Called.
// If a node was not found in the database, a MissingNodeError is returned.
func (tPtr *Trie) TryGet(key []byte) ([]byte, error) {
	if tPtr.prefix != nil {
		key = append(tPtr.prefix, key...)
	}
	key = keybytesToHex(key)
	value, newblockRoot, didResolve, err := tPtr.tryGet(tPtr.blockRoot, key, 0)
	if err == nil && didResolve {
		tPtr.blockRoot = newblockRoot
	}
	return value, err
}