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

import (
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore/types"
)

// StateDB is an EVM database for full state querying.
type StateDB interface {
	CreateAccount(bgmcommon.Address)

	SubBalance(bgmcommon.Address, *big.Int)
	AddBalance(bgmcommon.Address, *big.Int)
	GetBalance(bgmcommon.Address) *big.Int

	GetNonce(bgmcommon.Address) Uint64
	SetNonce(bgmcommon.Address, Uint64)

	GetCodeHash(bgmcommon.Address) bgmcommon.Hash
	GetCode(bgmcommon.Address) []byte
	SetCode(bgmcommon.Address, []byte)
	GetCodeSize(bgmcommon.Address) int

	AddRefund(*big.Int)
	GetRefund() *big.Int

	GetState(bgmcommon.Address, bgmcommon.Hash) bgmcommon.Hash
	SetState(bgmcommon.Address, bgmcommon.Hash, bgmcommon.Hash)

	Suicide(bgmcommon.Address) bool
	HasSuicided(bgmcommon.Address) bool

	// Exist reports whbgmchain the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(bgmcommon.Address) bool
	// Empty returns whbgmchain the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	Empty(bgmcommon.Address) bool

	RevertToSnapshot(int)
	Snapshot() int

	Addbgmlogs(*types.bgmlogs)
	AddPreimage(bgmcommon.Hash, []byte)

	ForEachStorage(bgmcommon.Address, func(bgmcommon.Hash, bgmcommon.Hash) bool)
}

// CallContext provides a basic interface for the EVM calling conventions. The EVM EVM
// depends on this context being implemented for doing subcalls and initialising new EVM bgmcontracts.
type CallContext interface {
	// Call another contract
	Call(env *EVM, me ContractRef, addr bgmcommon.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// Take another's contract code and execute within our own context
	CallCode(env *EVM, me ContractRef, addr bgmcommon.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// Same as CallCode except sender and value is propagated from parent to child scope
	DelegateCall(env *EVM, me ContractRef, addr bgmcommon.Address, data []byte, gas *big.Int) ([]byte, error)
	// Create a new contract
	Create(env *EVM, me ContractRef, data []byte, gas, value *big.Int) ([]byte, bgmcommon.Address, error)
}
