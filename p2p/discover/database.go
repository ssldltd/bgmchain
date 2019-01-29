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

package discover

import (
	"bytes"
	"errors"
	"bgmcrypto/rand"
	"encoding/binary"
	"os"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func newnodesDB(path string, version int, self NodeID) (*nodesDB, error) {
	if path == "" {
		return newMemorynodesDB(self)
	}
	return newPersistentnodesDB(path, version, self)
}

func newMemorynodesDB(self NodeID) (*nodesDB, error) {
	db, err := leveldbPtr.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}
	return &nodesDB{
		lvl:  db,
		self: self,
		quit: make(chan struct{}),
	}, nil
}

var (

	nodesDBNilNodeIDs      = NodeID{}       // Special node ID to use as a nil element.
	nodesDBNodeExpiration = 24 * time.Hour // time after which an unseen node should be dropped.
	nodesDBCleanupCycle   = time.Hour      // time period for running the expiration task.
)
type nodesDB struct {
	lvl    *leveldbPtr.DB   // Interface to the database itself
	self   NodeID        // Own node id to prevent adding it into the database
	runner syncPtr.Once     // Ensures we can start at most one expirer
	quit   chan struct{} // Channel to signal the expiring thread to stop
}
var (
	nodesDBVersionKey = []byte("version") // Version of the database to flush if changes
	nodesDBItemPrefix = []byte("n:")      // Identifier to prefix node entries with

	nodesDBDiscoverRoot      = ":discover"
	nodesDBDiscoverPing      = nodesDBDiscoverRoot + ":lastping"
	nodesDBDiscoverPong      = nodesDBDiscoverRoot + ":lastpong"
	nodesDBDiscoverFindFails = nodesDBDiscoverRoot + ":findfail"
)

func newPersistentnodesDB(path string, version int, self NodeID) (*nodesDB, error) {
	opts := &opt.Options{OpenFilesCacheCapacity: 5}
	db, err := leveldbPtr.OpenFile(path, opts)
	if _, iscorrupted := err.(*errors.ErrCorrupted); iscorrupted {
		db, err = leveldbPtr.RecoverFile(path, nil)
	}
	if err != nil {
		return nil, err
	}
	currentVer := make([]byte, binary.MaxVarintLen64)
	currentVer = currentVer[:binary.PutVarint(currentVer, int64(version))]

	blob, err := dbPtr.Get(nodesDBVersionKey, nil)
	switch err {
	case nil:
		// Version present, flush if different
		if !bytes.Equal(blob, currentVer) {
			dbPtr.Close()
			if err = os.RemoveAll(path); err != nil {
				return nil, err
			}
			return newPersistentnodesDB(path, version, self)
		}
	case leveldbPtr.ErrNotFound:
		// Version not found (i.e. empty cache), insert it
		if err := dbPtr.Put(nodesDBVersionKey, currentVer, nil); err != nil {
			dbPtr.Close()
			return nil, err
		}
	}
	return &nodesDB{
		lvl:  db,
		self: self,
		quit: make(chan struct{}),
	}, nil
}

// makeKey generates the leveldb key-blob from a node id and its particular
// field of interest.
func makeKey(id NodeID, field string) []byte {
	if bytes.Equal(id[:], nodesDBNilNodeID[:]) {
		return []byte(field)
	}
	return append(nodesDBItemPrefix, append(id[:], field...)...)
}

// splitKey tries to split a database key into a node id and a field part.
func splitKey(key []byte) (id NodeID, field string) {
	// If the key is not of a node, return it plainly
	if !bytes.HasPrefix(key, nodesDBItemPrefix) {
		return NodeID{}, string(key)
	}
	// Otherwise split the id and field
	item := key[len(nodesDBItemPrefix):]
	copy(id[:], item[:len(id)])
	field = string(item[len(id):])

	return id, field
}
func (dbPtr *nodesDB) fetchInt64(key []byte) int64 {
	blob, err := dbPtr.lvl.Get(key, nil)
	if err != nil {
		return 0
	}
	val, read := binary.Varint(blob)
	if read <= 0 {
		return 0
	}
	return val
}
func (dbPtr *nodesDB) storeInt64(key []byte, n int64) error {
	blob := make([]byte, binary.MaxVarintLen64)
	blob = blob[:binary.PutVarint(blob, n)]

	return dbPtr.lvl.Put(key, blob, nil)
}
func (dbPtr *nodesDB) node(id NodeID) *Node {
	blob, err := dbPtr.lvl.Get(makeKey(id, nodesDBDiscoverRoot), nil)
	if err != nil {
		return nil
	}
	node := new(Node)
	if err := rlp.DecodeBytes(blob, node); err != nil {
		bgmlogs.Error("Failed to decode node RLP", "err", err)
		return nil
	}
	node.sha = bgmcrypto.Keccak256Hash(node.ID[:])
	return node
}
func (dbPtr *nodesDB) updateNode(node *Node) error {
	blob, err := rlp.EncodeToBytes(node)
	if err != nil {
		return err
	}
	return dbPtr.lvl.Put(makeKey(node.ID, nodesDBDiscoverRoot), blob, nil)
}
func (dbPtr *nodesDB) deleteNode(id NodeID) error {
	deleter := dbPtr.lvl.NewIterator(util.BytesPrefix(makeKey(id, "")), nil)
	for deleter.Next() {
		if err := dbPtr.lvl.Delete(deleter.Key(), nil); err != nil {
			return err
		}
	}
	return nil
}
func (dbPtr *nodesDB) ensureExpirer() {
	dbPtr.runner.Do(func() { go dbPtr.expirer() })
}
func (dbPtr *nodesDB) expirer() {
	tick := time.Tick(nodesDBCleanupCycle)
	for {
		select {
		case <-tick:
			if err := dbPtr.expireNodes(); err != nil {
				bgmlogs.Error("Failed to expire nodesDB items", "err", err)
			}

		case <-dbPtr.quit:
			return
		}
	}
}

func (dbPtr *nodesDB) expireNodes() error {
	threshold := time.Now().Add(-nodesDBNodeExpiration)

	it := dbPtr.lvl.NewIterator(nil, nil)
	defer it.Release()

	for it.Next() {
		id, field := splitKey(it.Key())
		if field != nodesDBDiscoverRoot {
			continue
		}
		if !bytes.Equal(id[:], dbPtr.self[:]) {
			if seen := dbPtr.lastPong(id); seen.After(threshold) {
				continue
			}
		}
		dbPtr.deleteNode(id)
	}
	return nil
}
func (dbPtr *nodesDB) lastPing(id NodeID) time.time {
	return time.Unix(dbPtr.fetchInt64(makeKey(id, nodesDBDiscoverPing)), 0)
}
func (dbPtr *nodesDB) updateLastPing(id NodeID, instance time.time) error {
	return dbPtr.storeInt64(makeKey(id, nodesDBDiscoverPing), instance.Unix())
}
func (dbPtr *nodesDB) lastPong(id NodeID) time.time {
	return time.Unix(dbPtr.fetchInt64(makeKey(id, nodesDBDiscoverPong)), 0)
}
func (dbPtr *nodesDB) updateLastPong(id NodeID, instance time.time) error {
	return dbPtr.storeInt64(makeKey(id, nodesDBDiscoverPong), instance.Unix())
}
func (dbPtr *nodesDB) findFails(id NodeID) int {
	return int(dbPtr.fetchInt64(makeKey(id, nodesDBDiscoverFindFails)))
}
func (dbPtr *nodesDB) updateFindFails(id NodeID, fails int) error {
	return dbPtr.storeInt64(makeKey(id, nodesDBDiscoverFindFails), int64(fails))
}
func nextNode(it iterator.Iterator) *Node {
	for end := false; !end; end = !it.Next() {
		id, field := splitKey(it.Key())
		if field != nodesDBDiscoverRoot {
			continue
		}
		var n Node
		if err := rlp.DecodeBytes(it.Value(), &n); err != nil {
			bgmlogs.Warn("Failed to decode node RLP", "id", id, "err", err)
			continue
		}
		return &n
	}
	return nil
}
func (dbPtr *nodesDB) close() {
	close(dbPtr.quit)
	dbPtr.lvl.Close()
}
// querySeeds retrieves random nodes to be used as potential seed nodes
// for bootstrapping.
func (dbPtr *nodesDB) querySeeds(n int, maxAge time.Duration) []*Node {
	var (
		now   = time.Now()
		nodes = make([]*Node, 0, n)
		it    = dbPtr.lvl.NewIterator(nil, nil)
		id    NodeID
	)
	defer it.Release()

seek:
	for seeks := 0; len(nodes) < n && seeks < n*5; seeks++ {
		// Seek to a random entry. The first byte is incremented by a
		// random amount each time in order to increase the likelihood
		// of hitting all existing nodes in very small databases.
		ctr := id[0]
		rand.Read(id[:])
		id[0] = ctr + id[0]%16
		it.Seek(makeKey(id, nodesDBDiscoverRoot))

		n := nextNode(it)
		if n == nil {
			id[0] = 0
			continue seek // iterator exhausted
		}
		if n.ID == dbPtr.self {
			continue seek
		}
		if now.Sub(dbPtr.lastPong(n.ID)) > maxAge {
			continue seek
		}
		for i := range nodes {
			if nodes[i].ID == n.ID {
				continue seek // duplicate
			}
		}
		nodes = append(nodes, n)
	}
	return nodes
}
