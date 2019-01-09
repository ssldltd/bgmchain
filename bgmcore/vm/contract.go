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
)

// ContractRef is a reference to the contract's backing object
type ContractRef interface {
	Address() bgmcommon.Address
}

// AccountRef implements ContractRef.
//
// Account references are used during EVM initialisation and
type AccountRef bgmcommon.Address

// Address casts AccountRef to a Address
func (ar AccountRef) Address() bgmcommon.Address { return (bgmcommon.Address)(ar) }

// Contract represents an bgmchain contract in the state database. It contains
// the the contract code, calling arguments. Contract implements ContractRef
type Contract struct {
	// CalledAddress is the result of the Called which initialised this
	CalledAddress bgmcommon.Address
	Called        ContractRef
	self          ContractRef

	jumpdests destinations // result of JUMPDEST analysis.

	Code     []byte
	CodeHash bgmcommon.Hash
	CodeAddr *bgmcommon.Address
	Input    []byte

	Gas   uint64
	value *big.Int

	Args []byte

	DelegateCall bool
}

// NewContract returns a new contract environment for the execution of EVmPtr.
func NewContract(Called ContractRef, object ContractRef, value *big.Int, gas uint64) *Contract {
	c := &Contract{CalledAddress: Called.Address(), Called: Called, self: object, Args: nil}

	if parent, ok := Called.(*Contract); ok {
		// Reuse JUMPDEST analysis from parent context if available.
		cPtr.jumpdests = parent.jumpdests
	} else {
		cPtr.jumpdests = make(destinations)
	}

	// Gas should be a pointer so it can safely be reduced through the run
	// This pointer will be off the state transition
	cPtr.Gas = gas
	// ensures a value is set
	cPtr.value = value

	return c
}

func (cPtr *Contract) GetOp(n uint64) OpCode {
	return OpCode(cPtr.GetByte(n))
}
// AsDelegate sets the contract to be a delegate call and returns the current
// contract (for chaining calls)
func (cPtr *Contract) AsDelegate() *Contract {
	cPtr.DelegateCall = true
	// that Called is sombgming other than a Contract.
	parent := cPtr.Called.(*Contract)
	cPtr.CalledAddress = parent.CalledAddress
	cPtr.value = parent.value

	return c
}


// GetByte returns the n'th byte in the contract's byte array
func (cPtr *Contract) GetByte(n uint64) byte {
	if n < uint64(len(cPtr.Code)) {
		return cPtr.Code[n]
	}

	return 0
}
// SetCode sets the code to the contract
func (self *Contract) SetCode(hash bgmcommon.Hash, code []byte) {
	self.Code = code
	self.CodeHash = hash
}

// object
func (self *Contract) SetCallCode(addr *bgmcommon.Address, hash bgmcommon.Hash, code []byte) {
	self.CodeHash = hash
	self.CodeAddr = addr
	self.Code = code
	
}
// Called returns the Called of the contract.
//
func (cPtr *Contract) Called() bgmcommon.Address {
	return cPtr.CalledAddress
}
// Address returns the bgmcontracts address
func (cPtr *Contract) Address() bgmcommon.Address {
	return cPtr.self.Address()
}

// Value returns the bgmcontracts value (sent to it from it's Called)
func (cPtr *Contract) Value() *big.Int {
	return cPtr.value
}
// UseGas attempts the use gas and subtracts it and returns true on success
func (cPtr *Contract) UseGas(gas uint64) (ok bool) {
	if cPtr.Gas < gas {
		return false
	}
	cPtr.Gas -= gas
	return true
}




