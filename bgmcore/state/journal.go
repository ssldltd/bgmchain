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

package state

import (
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
)

type journalEntry interface {
	undo(*StateDB)
}

type journal []journalEntry

type (
	// Changes to the account trie.
	createObjectChange struct {
		account *bgmcommon.Address
	}
	resetObjectChange struct {
		prev *stateObject
	}
	suicideChange struct {
		account     *bgmcommon.Address
		prev        bool // whbgmchain account had already suicided
		prevbalance *big.Int
	}

	// Changes to individual accounts.
	balanceChange struct {
		account *bgmcommon.Address
		prev    *big.Int
	}
	nonceChange struct {
		account *bgmcommon.Address
		prev    uint64
	}
	storageChange struct {
		account       *bgmcommon.Address
		key, prevalue bgmcommon.Hash
	}
	codeChange struct {
		account            *bgmcommon.Address
		prevcode, prevhash []byte
	}

	// Changes to other state values.
	refundChange struct {
		prev *big.Int
	}
	addbgmlogsChange struct {
		txhash bgmcommon.Hash
	}
	addPreimageChange struct {
		hash bgmcommon.Hash
	}
	touchChange struct {
		account   *bgmcommon.Address
		prev      bool
		prevDirty bool
	}
)

func (ch createObjectChange) undo(s *StateDB) {
	delete(s.stateObjects, *chPtr.account)
	delete(s.stateObjectsDirty, *chPtr.account)
}

func (ch resetObjectChange) undo(s *StateDB) {
	s.setStateObject(chPtr.prev)
}

func (ch suicideChange) undo(s *StateDB) {
	obj := s.getStateObject(*chPtr.account)
	if obj != nil {
		obj.suicided = chPtr.prev
		obj.setBalance(chPtr.prevbalance)
	}
}

var ripemd = bgmcommon.HexToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) undo(s *StateDB) {
	if !chPtr.prev && *chPtr.account != ripemd {
		s.getStateObject(*chPtr.account).touched = chPtr.prev
		if !chPtr.prevDirty {
			delete(s.stateObjectsDirty, *chPtr.account)
		}
	}
}

func (ch balanceChange) undo(s *StateDB) {
	s.getStateObject(*chPtr.account).setBalance(chPtr.prev)
}

func (ch nonceChange) undo(s *StateDB) {
	s.getStateObject(*chPtr.account).setNonce(chPtr.prev)
}

func (ch codeChange) undo(s *StateDB) {
	s.getStateObject(*chPtr.account).setCode(bgmcommon.BytesToHash(chPtr.prevhash), chPtr.prevcode)
}

func (ch storageChange) undo(s *StateDB) {
	s.getStateObject(*chPtr.account).setState(chPtr.key, chPtr.prevalue)
}

func (ch refundChange) undo(s *StateDB) {
	s.refund = chPtr.prev
}

func (ch addbgmlogsChange) undo(s *StateDB) {
	bgmlogss := s.bgmlogss[chPtr.txhash]
	if len(bgmlogss) == 1 {
		delete(s.bgmlogss, chPtr.txhash)
	} else {
		s.bgmlogss[chPtr.txhash] = bgmlogss[:len(bgmlogss)-1]
	}
	s.bgmlogsSize--
}

func (ch addPreimageChange) undo(s *StateDB) {
	delete(s.preimages, chPtr.hash)
}
