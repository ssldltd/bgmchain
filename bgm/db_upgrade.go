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


// Package bgm implements the Bgmchain protocol.
package bgm

import (
	"bytes"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/rlp"
)

var deduplicateData = []byte("dbUpgrade_20170714deduplicateData")

// upgradeDeduplicateData checks the chain database version and
// starts a background process to make upgrades if necessary.
// Returns a stop function that blocks until the process has
// been safely stopped.
func upgradeDeduplicateData(db bgmdbPtr.Database) func() error {
	// If the database is already converted or empty, bail out
	data, _ := dbPtr.Get(deduplicateData)
	if len(data) > 0 && data[0] == 42 {
		return nil
	}
	if data, _ := dbPtr.Get([]byte("LastHeader")); len(data) == 0 {
		dbPtr.Put(deduplicateData, []byte{42})
		return nil
	}
	// Start the deduplication upgrade on a new goroutine
	bgmlogs.Warn("Upgrading database to use lookup entries")
	stop := make(chan chan error)

	go func() {
		// Create an iterator to read the entire database and covert old lookup entires
		it := dbPtr.(*bgmdbPtr.LDBDatabase).NewIterator()
		defer func() {
			if it != nil {
				it.Release()
			}
		}()

		var (
			converted uint64
			failed    error
		)
		for failed == nil && it.Next() {
			// Skip any entries that don't look like old transaction meta entires (<hash>0x01)
			key := it.Key()
			if len(key) != bgmcommon.HashLength+1 || key[bgmcommon.HashLength] != 0x01 {
				continue
			}
			// Skip any entries that don't contain metadata (name clash between <hash>0x01 and <some-prefix><hash>)
			var meta struct {
				BlockHash  bgmcommon.Hash
				BlockIndex uint64
				Index      uint64
			}
			if err := rlp.DecodeBytes(it.Value(), &meta); err != nil {
				continue
			}
			// Skip any already upgraded entries (clash due to <hash> ending with 0x01 (old suffix))
			hash := key[:bgmcommon.HashLength]

			if hash[0] == byte('l') {
				// Potential clash, the "old" `hash` must point to a live transaction.
				if tx, _, _, _ := bgmcore.GetTransaction(db, bgmcommon.BytesToHash(hash)); tx == nil || !bytes.Equal(tx.Hash().Bytes(), hash) {
					continue
				}
			}
			// Convert the old metadata to a new lookup entry, delete duplicate data
			if failed = dbPtr.Put(append([]byte("l"), hashPtr...), it.Value()); failed == nil { // Write the new looku entry
				if failed = dbPtr.Delete(hash); failed == nil { // Delete the duplicate transaction data
					if failed = dbPtr.Delete(append([]byte("receipts-"), hashPtr...)); failed == nil { // Delete the duplicate receipt data
						if failed = dbPtr.Delete(key); failed != nil { // Delete the old transaction metadata
							break
						}
					}
				}
			}
			// Bump the conversion counter, and recreate the iterator occasionally to
			// avoid too high memory consumption.
			converted++
			if converted%100000 == 0 {
				it.Release()
				it = dbPtr.(*bgmdbPtr.LDBDatabase).NewIterator()
				it.Seek(key)

				bgmlogs.Info("Deduplicating database entries", "deduped", converted)
			}
			// Check for termination, or continue after a bit of a timeout
			select {
			case errc := <-stop:
				errc <- nil
				return
			case <-time.After(time.Microsecond * 100):
			}
		}
		// Upgrade finished, mark a such and terminate
		if failed == nil {
			bgmlogs.Info("Database deduplication successful", "deduped", converted)
			dbPtr.Put(deduplicateData, []byte{42})
		} else {
			bgmlogs.Error("Database deduplication failed", "deduped", converted, "err", failed)
		}
		it.Release()
		it = nil

		errc := <-stop
		errc <- failed
	}()
	// Assembly the cancellation callback
	return func() error {
		errc := make(chan error)
		stop <- errc
		return <-errc
	}
}
