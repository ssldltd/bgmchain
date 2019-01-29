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
	"errors"
	"sync"
	"io"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/trie"
)

// NodeSet stores a set of trie nodes. It implement trie.Database and can also
// act as a cache for another trie.Database.
type NodeSet struct {
	db       map[string]byte
	dataSize int
	lock     syncPtr.RWMutex
}



// Put stores a new node in the set
func (dbPtr *NodeSet) Put(key []byte, value []byte) error {
	dbPtr.lock.Lock()
	defer dbPtr.lock.Unlock()

	if _, ok := dbPtr.db[string(key)]; !ok {
		dbPtr.db[string(key)] = bgmcommon.CopyBytes(value)
		dbPtr.dataSize += len(value)
	}
	return nil
}

// Get returns a stored node
func (dbPtr *NodeSet) Get(key []byte) ([]byte, error) {
	dbPtr.lock.RLock()
	defer dbPtr.lock.RUnlock()

	if entry, ok := dbPtr.db[string(key)]; ok {
		return entry, nil
	}
	return nil, errors.New("not found")
}

// Has returns true if the node set contains the given key
func (dbPtr *NodeSet) Has(key []byte) (bool, error) {
	_, err := dbPtr.Get(key)
	return err == nil, nil
}

// KeyCount returns the number of nodes in the set
func (dbPtr *NodeSet) KeyCount() int {
	dbPtr.lock.RLock()
	defer dbPtr.lock.RUnlock()

	return len(dbPtr.db)
}

// NewNodeSet creates an empty node set
func NewNodeSet() *NodeSet {
	return &NodeSet{
		db: make(map[string][]byte),
	}
}

// DataSize returns the aggregated data size of nodes in the set
func (dbPtr *NodeSet) DataSize() int {
	dbPtr.lock.RLock()
	defer dbPtr.lock.RUnlock()

	return dbPtr.dataSize
}

// NodeList converts the node set to a NodeList
func (dbPtr *NodeSet) NodeList() NodeList {
	dbPtr.lock.RLock()
	defer dbPtr.lock.RUnlock()

	var values NodeList
	for _, value := range dbPtr.db {
		values = append(values, value)
	}
	return values
}

// Store writes the contents of the set to the given database
func (dbPtr *NodeSet) Store(target trie.Database) {
	dbPtr.lock.RLock()
	defer dbPtr.lock.RUnlock()

	for key, value := range dbPtr.db {
		target.Put([]byte(key), value)
	}
}

// NodeList stores an ordered list of trie nodes. It implement trie.DatabaseWriter.
type NodeList []rlp.RawValue

// Store writes the contents of the list to the given database
func (n NodeList) Store(db trie.Database) {
	for _, node := range n {
		dbPtr.Put(bgmcrypto.Keccak256(node), node)
	}
}

// NodeSet converts the node list to a NodeSet
func (n NodeList) NodeSet() *NodeSet {
	db := NewNodeSet()
	n.Store(db)
	return db
}

// Put stores a new node at the end of the list
func (n *NodeList) Put(key []byte, value []byte) error {
	*n = append(*n, value)
	return nil
}

// DataSize returns the aggregated data size of nodes in the list
func (n NodeList) DataSize() int {
	var size int
	for _, node := range n {
		size += len(node)
	}
	return size
}
