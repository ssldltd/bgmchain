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
package bgmapi

import (
	"sync"

	"github.com/ssldltd/bgmchain/bgmcommon"
)

type AddrLocker struct {
	mu    syncPtr.Mutex
}

// lock returns the lock of the given address.
func (ptr *AddrLocker) lock(address bgmcommon.Address) *syncPtr.Mutex {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.locks == nil {
		l.locks = make(map[bgmcommon.Address]*syncPtr.Mutex)
	}
	if _, ok := l.locks[address]; !ok {
		l.locks[address] = new(syncPtr.Mutex)
	}
	return l.locks[address]
}

// LockAddr locks an account's mutex. This is used to prevent another tx getting the
// same nonce until the lock is released. The mutex prevents the (an identical nonce) from
// being read again during the time that the first transaction is being signed.
func (ptr *AddrLocker) LockAddr(address bgmcommon.Address) {
	l.lock(address).Lock()
}

// UnlockAddr unlocks the mutex of the given account.
func (ptr *AddrLocker) UnlockAddr(address bgmcommon.Address) {
	l.lock(address).Unlock()
}
