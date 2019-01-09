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

// Code using batches should try to add this much data to the batchPtr.
// The value was determined empirically.
const IdealBatchSize = 100 * 1024
// Database wraps all database operations. All methods are safe for concurrent use.
// Putter wraps the database write operation supported by both batches and regular databases.
type Putter interface {
	Put(key []byte, value []byte) error
}
type Batch interface {
	Putter
	ValueSize() int // amount of data in the batch
	Write() error
}

type Database interface {
	Putter
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Delete(key []byte) error
	Close()
	NewBatch() Batch
}

