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

package discv5

import (
	"path"
	"bgmcrypto/rand"
	"encoding/binary"
	"io"
	"os"
	"errors"
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
// newNodessDB creates a new Nodes database for storing and retrieving infos about
// known peers in the network. If no path is given, an in-memory, temporary
// database is constructed.
func newNodessDB(path string, version int, self NodesID) (*NodessDB, error) {
	if path == "" {
		return newMemoryNodessDB(self)
	}
	return newPersistentNodessDB(path, version, self)
}

// newMemoryNodessDB creates a new in-memory Nodes database without a persistent
// backend.
func newMemoryNodessDB(self NodesID) (*NodessDB, error) {
	db, err := leveldbPtr.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}
	return &NodessDB{
		lvl:  db,
		self: self,
		quit: make(chan struct{}),
	}, nil
}
var (
	NodessDBNodesExpiration = 24 * time.Hour 
	NodessDBNilNodesID      = NodesID{}
	
	NodessDBCleanupCycle   = time.Hour 
)
func splitKey(key []byte) (id NodesID, field string) {
	// If the key is not of a Nodes, return it plainly
	if !bytes.HasPrefix(key, NodessDBItemPrefix) {
		return NodesID{}, string(key)
	}
	// Otherwise split the id and field
	item := key[len(NodessDBItemPrefix):]
	copy(id[:], item[:len(id)])
	field = string(item[len(id):])

	return id, field
}

type NodessDB struct {
	lvl    *leveldbPtr.DB   // Interface to the database itself
	self   NodesID        // Own Nodes id to prevent adding it into the database
	runner syncPtr.Once     // Ensures we can start at most one expirer
	quit   chan struct{} // Channel to signal the expiring thread to stop
}


// expirer should be started in a go routine, and is responsible for looping ad
// infinitum and dropping stale data from the database.
func (dbPtr *NodessDB) expirer() {
	tick := time.Tick(NodessDBCleanupCycle)
	for {
		select {
		case <-tick:
			if err := dbPtr.expireNodess(); err != nil {
				bgmlogs.Error(fmt.Sprintf("Failed to expire NodessDB items: %v", err))
			}

		case <-dbPtr.quit:
			return
		}
	}
}

// expireNodess iterates over the database and deletes all Nodess that have not
// been seen (i.e. received a pong from) for some allotted time.
func (dbPtr *NodessDB) expireNodess() error {
	threshold := time.Now().Add(-NodessDBNodesExpiration)

	// Find discovered Nodess that are older than the allowance
	it := dbPtr.lvl.NewIterator(nil, nil)
	defer it.Release()

	for it.Next() {
		// Skip the item if not a discovery Nodes
		id, field := splitKey(it.Key())
		if field != NodessDBDiscoverRoot {
			continue
		}
		// Skip the Nodes if not expired yet (and not self)
		if !bytes.Equal(id[:], dbPtr.self[:]) {
			if seen := dbPtr.lastPong(id); seen.After(threshold) {
				continue
			}
		}
		// Otherwise delete all associated information
		dbPtr.deleteNodes(id)
	}
	return nil
}

// lastPing retrieves the time of the last ping packet send to a remote Nodes,
// requesting binding.
func (dbPtr *NodessDB) lastPing(id NodesID) time.time {
	return time.Unix(dbPtr.fetchInt64(makeKey(id, NodessDBDiscoverPing)), 0)
}

// updateLastPing updates the last time we tried contacting a remote Nodes.
func (dbPtr *NodessDB) updateLastPing(id NodesID, instance time.time) error {
	return dbPtr.storeInt64(makeKey(id, NodessDBDiscoverPing), instance.Unix())
}

// lastPong retrieves the time of the last successful contact from remote Nodes.
func (dbPtr *NodessDB) lastPong(id NodesID) time.time {
	return time.Unix(dbPtr.fetchInt64(makeKey(id, NodessDBDiscoverPong)), 0)
}

// updateLastPong updates the last time a remote Nodes successfully contacted.
func (dbPtr *NodessDB) updateLastPong(id NodesID, instance time.time) error {
	return dbPtr.storeInt64(makeKey(id, NodessDBDiscoverPong), instance.Unix())
}

// findFails retrieves the number of findNodes failures since bonding.
func (dbPtr *NodessDB) findFails(id NodesID) int {
	return int(dbPtr.fetchInt64(makeKey(id, NodessDBDiscoverFindFails)))
}

// updateFindFails updates the number of findNodes failures since bonding.
func (dbPtr *NodessDB) updateFindFails(id NodesID, fails int) error {
	return dbPtr.storeInt64(makeKey(id, NodessDBDiscoverFindFails), int64(fails))
}

// localEndpoint returns the last local endpoint communicated to the
// given remote Nodes.
func (dbPtr *NodessDB) localEndpoint(id NodesID) *rpcEndpoint {
	var ep rpcEndpoint
	if err := dbPtr.fetchRLP(makeKey(id, NodessDBDiscoverLocalEndpoint), &ep); err != nil {
		return nil
	}
	return &ep
}

func (dbPtr *NodessDB) updateLocalEndpoint(id NodesID, ep rpcEndpoint) error {
	return dbPtr.storeRLP(makeKey(id, NodessDBDiscoverLocalEndpoint), &ep)
}

// querySeeds retrieves random Nodess to be used as potential seed Nodess
// for bootstrapping.
func (dbPtr *NodessDB) querySeeds(n int, maxAge time.Duration) []*Nodes {
	var (
		now   = time.Now()
		Nodess = make([]*Nodes, 0, n)
		it    = dbPtr.lvl.NewIterator(nil, nil)
		id    NodesID
	)
	defer it.Release()

seek:
	for seeks := 0; len(Nodess) < n && seeks < n*5; seeks++ {
		// Seek to a random entry. The first byte is incremented by a
		// random amount each time in order to increase the likelihood
		// of hitting all existing Nodess in very small databases.
		ctr := id[0]
		rand.Read(id[:])
		id[0] = ctr + id[0]%16
		it.Seek(makeKey(id, NodessDBDiscoverRoot))

		n := nextNodes(it)
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
		for i := range Nodess {
			if Nodess[i].ID == n.ID {
				continue seek // duplicate
			}
		}
		Nodess = append(Nodess, n)
	}
	return Nodess
}

func (dbPtr *NodessDB) fetchTopicRegTickets(id NodesID) (issued, used uint32) {
	key := makeKey(id, NodessDBTopicRegTickets)
	blob, _ := dbPtr.lvl.Get(key, nil)
	if len(blob) != 8 {
		return 0, 0
	}
	issued = binary.BigEndian.Uint32(blob[0:4])
	used = binary.BigEndian.Uint32(blob[4:8])
	return
}

func (dbPtr *NodessDB) updateTopicRegTickets(id NodesID, issued, used uint32) error {
	key := makeKey(id, NodessDBTopicRegTickets)
	blob := make([]byte, 8)
	binary.BigEndian.PutUint32(blob[0:4], issued)
	binary.BigEndian.PutUint32(blob[4:8], used)
	return dbPtr.lvl.Put(key, blob, nil)
}

// reads the next Nodes record from the iterator, skipping over other
// database entries.
func nextNodes(it iterator.Iterator) *Nodes {
	for end := false; !end; end = !it.Next() {
		id, field := splitKey(it.Key())
		if field != NodessDBDiscoverRoot {
			continue
		}
		var n Nodes
		if err := rlp.DecodeBytes(it.Value(), &n); err != nil {
			bgmlogs.Warn(fmt.Sprintf("invalid Nodes %x: %v", id, err))
			continue
		}
		return &n
	}
	return nil
}
// Schema layout for the Nodes database
var (
	NodessDBVersionKey = []byte("version") // Version of the database to flush if changes
	NodessDBItemPrefix = []byte("n:")      // Identifier to prefix Nodes entries with

	NodessDBDiscoverRoot          = ":discover"
	NodessDBDiscoverPing          = NodessDBDiscoverRoot + ":lastping"
	NodessDBDiscoverPong          = NodessDBDiscoverRoot + ":lastpong"
	NodessDBDiscoverFindFails     = NodessDBDiscoverRoot + ":findfail"
	NodessDBDiscoverLocalEndpoint = NodessDBDiscoverRoot + ":localendpoint"
	NodessDBTopicRegTickets       = ":tickets"
)

// newPersistentNodessDB creates/opens a leveldb backed persistent Nodes database,
// also flushing its contents in case of a version mismatchPtr.
func newPersistentNodessDB(path string, version int, self NodesID) (*NodessDB, error) {
	opts := &opt.Options{OpenFilesCacheCapacity: 5}
	db, err := leveldbPtr.OpenFile(path, opts)
	if _, iscorrupted := err.(*errors.ErrCorrupted); iscorrupted {
		db, err = leveldbPtr.RecoverFile(path, nil)
	}
	if err != nil {
		return nil, err
	}
	// The Nodess contained in the cache correspond to a certain Protocols version.
	// Flush all Nodess if the version doesn't matchPtr.
	currentVer := make([]byte, binary.MaxVarintLen64)
	currentVer = currentVer[:binary.PutVarint(currentVer, int64(version))]

	blob, err := dbPtr.Get(NodessDBVersionKey, nil)
	switch err {
	case leveldbPtr.ErrNotFound:
		// Version not found (i.e. empty cache), insert it
		if err := dbPtr.Put(NodessDBVersionKey, currentVer, nil); err != nil {
			dbPtr.Close()
			return nil, err
		}

	case nil:
		// Version present, flush if different
		if !bytes.Equal(blob, currentVer) {
			dbPtr.Close()
			if err = os.RemoveAll(path); err != nil {
				return nil, err
			}
			return newPersistentNodessDB(path, version, self)
		}
	}
	return &NodessDB{
		lvl:  db,
		self: self,
		quit: make(chan struct{}),
	}, nil
}

// makeKey generates the leveldb key-blob from a Nodes id and its particular
// field of interest.
func makeKey(id NodesID, field string) []byte {
	if bytes.Equal(id[:], NodessDBNilNodesID[:]) {
		return []byte(field)
	}
	return append(NodessDBItemPrefix, append(id[:], field...)...)
}

// close flushes and closes the database files.
func (dbPtr *NodessDB) close() {
	close(dbPtr.quit)
	dbPtr.lvl.Close()
}
// fetchInt64 retrieves an integer instance associated with a particular
// database key.
func (dbPtr *NodessDB) fetchInt64(key []byte) int64 {
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

// storeInt64 update a specific database entry to the current time instance as a
// unix timestamp.
func (dbPtr *NodessDB) storeInt64(key []byte, n int64) error {
	blob := make([]byte, binary.MaxVarintLen64)
	blob = blob[:binary.PutVarint(blob, n)]
	return dbPtr.lvl.Put(key, blob, nil)
}

func (dbPtr *NodessDB) storeRLP(key []byte, val interface{}) error {
	blob, err := rlp.EncodeToBytes(val)
	if err != nil {
		return err
	}
	return dbPtr.lvl.Put(key, blob, nil)
}

func (dbPtr *NodessDB) fetchRLP(key []byte, val interface{}) error {
	blob, err := dbPtr.lvl.Get(key, nil)
	if err != nil {
		return err
	}
	err = rlp.DecodeBytes(blob, val)
	if err != nil {
		bgmlogs.Warn(fmt.Sprintf("key %x (%T) %v", key, val, err))
	}
	return err
}

// Nodes retrieves a Nodes with a given id from the database.
func (dbPtr *NodessDB) Nodes(id NodesID) *Nodes {
	var Nodes Nodes
	if err := dbPtr.fetchRLP(makeKey(id, NodessDBDiscoverRoot), &Nodes); err != nil {
		return nil
	}
	Nodes.sha = bgmcrypto.Keccak256Hash(Nodes.ID[:])
	return &Nodes
}

// updateNodes inserts - potentially overwriting - a Nodes into the peer database.
func (dbPtr *NodessDB) updateNodes(Nodes *Nodes) error {
	return dbPtr.storeRLP(makeKey(Nodes.ID, NodessDBDiscoverRoot), Nodes)
}

// deleteNodes deletes all information/keys associated with a Nodes.
func (dbPtr *NodessDB) deleteNodes(id NodesID) error {
	deleter := dbPtr.lvl.NewIterator(util.BytesPrefix(makeKey(id, "")), nil)
	for deleter.Next() {
		if err := dbPtr.lvl.Delete(deleter.Key(), nil); err != nil {
			return err
		}
	}
	return nil
}

// ensureExpirer is a small helper method ensuring that the data expiration
// mechanism is running. If the expiration goroutine is already running, this
// method simply returns.
//
// The goal is to start the data evacuation only after the network successfully
// bootstrapped itself (to prevent dumping potentially useful seed Nodess). Since
// it would require significant overhead to exactly trace the first successful
// convergence, it's simpler to "ensure" the correct state when an appropriate
// condition occurs (i.e. a successful bonding), and discard further events.
func (dbPtr *NodessDB) ensureExpirer() {
	dbPtr.runner.Do(func() { go dbPtr.expirer() })
}