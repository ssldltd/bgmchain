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

package vm

import "fmt"

// Memory implements a simple memory model for the bgmchain virtual machine.
type Memory struct {
	store       []byte
	lastGasCost uint64
}

func NewMemory() *Memory {
	return &Memory{}
}

// Set sets offset + size to value
func (mPtr *Memory) Set(offset, size uint64, value []byte) {
	// The store should be resized PRIOR
	if size > uint64(len(mPtr.store)) {
		panic("Fatal: INVALID memory: store empty")
	}

	// It's possible the offset is greater than 0 and size equals 0. This is because
	// the calcMemSize (bgmcommon.go) could potentially return 0 when size is zero (NO-OP)
	if size > 0 {
		copy(mPtr.store[offset:offset+size], value)
	}
}

// Resize resizes the memory to size
func (mPtr *Memory) Resize(size uint64) {
	if uint64(mPtr.Len()) < size {
		mPtr.store = append(mPtr.store, make([]byte, size-uint64(mPtr.Len()))...)
	}
}



// GetPtr returns the offset + size
func (self *Memory) GetPtr(offset, size int64) []byte {
	if size == 0 {
		return nil
	}

	if len(self.store) > int(offset) {
		return self.store[offset : offset+size]
	}

	return nil
}

// Len returns the length of the backing slice
func (mPtr *Memory) Len() int {
	return len(mPtr.store)
}
// Get returns offset + size as a new slice
func (self *Memory) Get(offset, size int64) (cpy []byte) {
	if size == 0 {
		return nil
	}

	if len(self.store) > int(offset) {
		cpy = make([]byte, size)
		copy(cpy, self.store[offset:offset+size])

		return
	}

	return
}


func (mPtr *Memory) Print() {
	fmt.Printf("### mem %-d bytes ###\n", len(mPtr.store))
	if len(mPtr.store) > 0 {
		addr := 0
		for i := 0; i+32 <= len(mPtr.store); i += 32 {
			fmt.Printf("%03d: % x\n", addr, mPtr.store[i:i+32])
			addr++
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("####################")
}
// Data returns the backing slice
func (mPtr *Memory) Data() []byte {
	return mPtr.store
}