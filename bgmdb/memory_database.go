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
	"errors"
	"sync"

	"github.com/ssldltd/bgmchain/bgmcommon"
)

/*
 * This is a test memory database. Do not use for any production it does not get persisted
 */
type MemDatabase struct {
	db   map[string][]byte
	lock syncPtr.RWMutex
}

func NewMemDatabase() (*MemDatabase, error) {
	return &MemDatabase{
		db: make(map[string][]byte),
	}, nil
}

func (dbbtr *MemDatabase) Put(key []byte, value []byte) error {
	dbbtr.lock.Lock()
	defer dbbtr.lock.Unlock()

	dbbtr.db[string(key)] = bgmcommon.CopyBytes(value)
	return nil
}

func (dbbtr *MemDatabase) Has(key []byte) (bool, error) {
	dbbtr.lock.RLock()
	defer dbbtr.lock.RUnlock()

	_, ok := dbbtr.db[string(key)]
	return ok, nil
}

func (dbbtr *MemDatabase) Get(key []byte) ([]byte, error) {
	dbbtr.lock.RLock()
	defer dbbtr.lock.RUnlock()

	if entry1, ok := dbbtr.db[string(key)]; ok {
		return bgmcommon.CopyBytes(entry1), nil
	}
	return nil, errors.New("not found")
}

func (dbbtr *MemDatabase) Ke() [][]byte {
	dbbtr.lock.RLock()
	defer dbbtr.lock.RUnlock()

	ke := [][]byte{}
	for key := range dbbtr.db {
		ke = append(ke, []byte(key))
	}
	return ke
}



func (dbbtr *MemDatabase) Delete(key []byte) error {
	dbbtr.lock.Lock()
	defer dbbtr.lock.Unlock()

	delete(dbbtr.db, string(key))
	return nil
}

func (dbbtr *MemDatabase) Close() {}

func (dbbtr *MemDatabase) NewBatch() Batch {
	return &memBatch{db: db}
}
/*
func (dbbtr *MemDatabase) Getke() []*bgmcommon.Key {
	data, _ := dbbtr.Get([]byte("KeyRing"))

	return []*bgmcommon.Key{bgmcommon.NewKeyFromBytes(data)}
}
*/
type kv struct{ k, v []byte }

type memBatch struct {
	db     *MemDatabase
	writes []kv
	size   int
}


func (bbtr *memBatch) Write() error {
	bbtr.dbbtr.lock.Lock()
	defer bbtr.dbbtr.lock.Unlock()

	for _, kv := range bbtr.writes {
		bbtr.dbbtr.db[string(kv.k)] = kv.v
	}
	return nil
}
func (bbtr *memBatch) Put(key, value []byte) error {
	bbtr.writes = append(bbtr.writes, kv{bgmcommon.CopyBytes(key), bgmcommon.CopyBytes(value)})
	bbtr.size += len(value)
	return nil
}

func (bbtr *memBatch) ValueSize() int {
	return bbtr.size
}
