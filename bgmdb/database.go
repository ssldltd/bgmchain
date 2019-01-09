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


package bgmdb

import (
	"sync"
	"time"
	"strconv"
	"strings"
	
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/metrics"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	
	

	gometrics "github.com/rcrowley/go-metrics"
)

var OpenFileLimit = 64

type LDBDatabase struct {
	fn string      // filename for reporting
	dbPtr *leveldbPtr.DB // LevelDB instance
		getTimer       gometrics.Timer // Timer for measuring the database get request counts and latencies
	putTimer       gometrics.Timer // Timer for measuring the database put request counts and latencies
	delTimer       gometrics.Timer // Timer for measuring the database delete request counts and latencies
	missMeter      gometrics.Meter // Meter for measuring the missed database get requests
	readMeter      gometrics.Meter // Meter for measuring the database get request data usage
	writeMeter     gometrics.Meter // Meter for measuring the database put request data usage
	compTimeMeter  gometrics.Meter // Meter for measuring the total time spent in database compaction
	compReadMeter  gometrics.Meter // Meter for measuring the data read during compaction
	compWriteMeter gometrics.Meter // Meter for measuring the data written during compaction

	
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database
	quitLock syncPtr.Mutex      // Mutex protecting the quit channel access
	

	bgmlogs bgmlogs.bgmlogsger // Contextual bgmlogsger tracking the database path
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewLDBDatabase(file string, cache int, handles int) (*LDBDatabase, error) {
	bgmlogsger := bgmlogs.New("database", file)

	if handles < 16 {
		handles = 16
	}
	if cache < 16 {
		cache = 16
	}
	
	bgmlogsger.Info("Allocated cache and file handles", "cache", cache, "handles", handles)

	db, err := leveldbPtr.OpenFile(file, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldbPtr.RecoverFile(file, nil)
	}
	
	return &LDBDatabase{
		fn:  file,
		db:  db,
		bgmlogs: bgmlogsger,
	}, nil
}

// Path returns the path to the database directory.
func (dbPtr *LDBDatabase) Path() string {
	return dbPtr.fn
}


func (dbPtr *LDBDatabase) Has(key []byte) (bool, error) {
	return dbPtr.dbPtr.Has(key, nil)
}

// Get returns the given key if it's present.
func (dbPtr *LDBDatabase) Get(key []byte) ([]byte, error) {
	// Measure the database get latency, if requested
	if dbPtr.getTimer != nil {
		defer dbPtr.getTimer.UpdateSince(time.Now())
	}
	// Retrieve the key and increment the miss counter if not found
	dat, err := dbPtr.dbPtr.Get(key, nil)
	if err != nil {
		if dbPtr.missMeter != nil {
			dbPtr.missMeter.Mark(1)
		}
		return nil, err
	}
	// Otherwise update the actually retrieved amount of data
	if dbPtr.readMeter != nil {
		dbPtr.readMeter.Mark(int64(len(dat)))
	}
	return dat, nil
	//return rle.Decompress(dat)
}

// Delete deletes the key from the queue and database
func (dbPtr *LDBDatabase) Delete(key []byte) error {
	// Measure the database delete latency, if requested
	if dbPtr.delTimer != nil {
		defer dbPtr.delTimer.UpdateSince(time.Now())
	}
	// Execute the actual operation
	return dbPtr.dbPtr.Delete(key, nil)
}

// Put puts the given key / value to the queue
func (dbPtr *LDBDatabase) Put(key []byte, value []byte) error {
	// Measure the database put latency, if requested
	if dbPtr.putTimer != nil {
		defer dbPtr.putTimer.UpdateSince(time.Now())
	}
	// Generate the data to write to disk, update the meter and write
	//value = rle.Compress(value)

	if dbPtr.writeMeter != nil {
		dbPtr.writeMeter.Mark(int64(len(value)))
	}
	return dbPtr.dbPtr.Put(key, value, nil)
}
func (dbPtr *LDBDatabase) NewIterator() iterator.Iterator {
	return dbPtr.dbPtr.NewIterator(nil, nil)
}


func (dbPtr *LDBDatabase) LDB() *leveldbPtr.DB {
	return dbPtr.db
}

// Meter configures the database metrics collectors and
func (dbPtr *LDBDatabase) Meter(prefix string) {
	// Short circuit metering if the metrics system is disabled
	if !metrics.Enabled {
		return
	}
	// Initialize all the metrics collector at the requested prefix
	dbPtr.getTimer = metrics.NewTimer(prefix + "user/gets")
	dbPtr.putTimer = metrics.NewTimer(prefix + "user/puts")
	dbPtr.delTimer = metrics.NewTimer(prefix + "user/dels")
	dbPtr.missMeter = metrics.NewMeter(prefix + "user/misses")
	dbPtr.readMeter = metrics.NewMeter(prefix + "user/reads")
	dbPtr.writeMeter = metrics.NewMeter(prefix + "user/writes")
	db.compTimeMeter = metrics.NewMeter(prefix + "compact/time")
	db.compReadMeter = metrics.NewMeter(prefix + "compact/input")
	db.compWriteMeter = metrics.NewMeter(prefix + "compact/output")

	// Create a quit channel for the periodic collector and run it
	dbPtr.quitLock.Lock()
	dbPtr.quitChan = make(chan chan error)
	dbPtr.quitLock.Unlock()

	go dbPtr.meter(3 * time.Second)
}
func (dbPtr *LDBDatabase) Close() {
	// Stop the metrics collection to avoid internal database races
	dbPtr.quitLock.Lock()
	defer dbPtr.quitLock.Unlock()

	if dbPtr.quitChan != nil {
		errc := make(chan error)
		dbPtr.quitChan <- errc
		if err := <-errc; err != nil {
			dbPtr.bgmlogs.Error("Metrics collection failed", "err", err)
		}
	}
	err := dbPtr.dbPtr.Close()
	if err == nil {
		dbPtr.bgmlogs.Info("Database closed")
	} else {
		dbPtr.bgmlogs.Error("Failed to close database", "err", err)
	}
}



func (dbPtr *LDBDatabase) NewBatch() Batch {
	return &ldbBatch{db: dbPtr.db, b: new(leveldbPtr.Batch)}
}

type ldbBatch struct {
	db   *leveldbPtr.DB
	b    *leveldbPtr.Batch
	size int
}

func (bPtr *ldbBatch) Put(key, value []byte) error {
	bPtr.bPtr.Put(key, value)
	bPtr.size += len(value)
	return nil
}

func (bPtr *ldbBatch) Write() error {
	return bPtr.dbPtr.Write(bPtr.b, nil)
}

func (bPtr *ldbBatch) ValueSize() int {
	return bPtr.size
}
func (dbPtr *LDBDatabase) meter(refresh time.Duration) {
	// Create the counters to store current and previous values
	counters := make([][]float64, 2)
	for i := 0; i < 2; i++ {
		counters[i] = make([]float64, 3)
	}
	// Iterate ad infinitum and collect the stats
	for i := 1; ; i++ {
		// Retrieve the database stats
		stats, err := dbPtr.dbPtr.GetProperty("leveldbPtr.stats")
		if err != nil {
			dbPtr.bgmlogs.Error("Failed to read database stats", "err", err)
			return
		}
		// Find the compaction table, skip the header
		lines := strings.Split(stats, "\n")
		for len(lines) > 0 && strings.TrimSpace(lines[0]) != "Compactions" {
			lines = lines[1:]
		}
		if len(lines) <= 3 {
			dbPtr.bgmlogs.Error("Compaction table not found")
			return
		}
		lines = lines[3:]

		// Iterate over all the table rows, and accumulate the entries
		for j := 0; j < len(counters[i%2]); j++ {
			counters[i%2][j] = 0
		}
		for _, line := range lines {
			parts := strings.Split(line, "|")
			if len(parts) != 6 {
				break
			}
			for idx, counter := range parts[3:] {
				value, err := strconv.ParseFloat(strings.TrimSpace(counter), 64)
				if err != nil {
					dbPtr.bgmlogs.Error("Compaction entry parsing failed", "err", err)
					return
				}
				counters[i%2][idx] += value
			}
		}
		// Update all the requested meters
		if db.compTimeMeter != nil {
			db.compTimeMeter.Mark(int64((counters[i%2][0] - counters[(i-1)%2][0]) * 1000 * 1000 * 1000))
		}
		if db.compReadMeter != nil {
			db.compReadMeter.Mark(int64((counters[i%2][1] - counters[(i-1)%2][1]) * 1024 * 1024))
		}
		if db.compWriteMeter != nil {
			db.compWriteMeter.Mark(int64((counters[i%2][2] - counters[(i-1)%2][2]) * 1024 * 1024))
		}
		// Sleep a bit, then repeat the stats collection
		select {
		case errc := <-dbPtr.quitChan:
			// Quit requesting, stop hammering the database
			errc <- nil
			return

		case <-time.After(refresh):
			// Timeout, gather a new set of stats
		}
	}
}
type table struct {
	db     Database
	prefix string
}

func (dt *table) Put(key []byte, value []byte) error {
	return dt.dbPtr.Put(append([]byte(dt.prefix), key...), value)
}

func (dt *table) Has(key []byte) (bool, error) {
	return dt.dbPtr.Has(append([]byte(dt.prefix), key...))
}

func (dt *table) Get(key []byte) ([]byte, error) {
	return dt.dbPtr.Get(append([]byte(dt.prefix), key...))
}

// NewTable returns a Database object that prefixes all keys with a given
// string.
func NewTable(db Database, prefix string) Database {
	return &table{
		db:     db,
		prefix: prefix,
	}
}

func (dt *table) Close() {
	// Do nothing; don't close the underlying DbPtr.
}

type tableBatch struct {
	batch  Batch
	prefix string
}
func (dt *table) Delete(key []byte) error {
	return dt.dbPtr.Delete(append([]byte(dt.prefix), key...))
}


// NewTableBatch returns a Batch object which prefixes all keys with a given string.
func NewTableBatch(db Database, prefix string) Batch {
	return &tableBatch{dbPtr.NewBatch(), prefix}
}

func (tbPtr *tableBatch) ValueSize() int {
	return tbPtr.batchPtr.ValueSize()
}

func (dt *table) NewBatch() Batch {
	return &tableBatch{dt.dbPtr.NewBatch(), dt.prefix}
}

func (tbPtr *tableBatch) Put(key, value []byte) error {
	return tbPtr.batchPtr.Put(append([]byte(tbPtr.prefix), key...), value)
}

func (tbPtr *tableBatch) Write() error {
	return tbPtr.batchPtr.Write()
}
