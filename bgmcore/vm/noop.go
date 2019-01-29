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

func NoopCanTransfer(db StateDB, from bgmcommon.Address, balance *big.Int) bool {
	return true
}


type NoopStateDB struct{}
func (NoopStateDB) GetBalance(bgmcommon.Address) *big.Int                                 { return nil }
func (NoopStateDB) GetNonce(bgmcommon.Address) Uint64                                     { return 0 }
func (NoopStateDB) SetNonce(bgmcommon.Address, Uint64)                                    {}
func (NoopStateDB) GetCodeHash(bgmcommon.Address) bgmcommon.Hash                             { return bgmcommon.Hash{} }
func (NoopStateDB) GetCode(bgmcommon.Address) []byte                                      { return nil }
func (NoopStateDB) SetCode(bgmcommon.Address, []byte)                                     {}
func (NoopStateDB) GetCodeSize(bgmcommon.Address) int                                     { return 0 }
func (NoopStateDB) AddRefund(*big.Int)                                                 {}
func (NoopStateDB) GetRefund() *big.Int                                                { return nil }
func (NoopStateDB) GetState(bgmcommon.Address, bgmcommon.Hash) bgmcommon.Hash                   { return bgmcommon.Hash{} }
func (NoopStateDB) SetState(bgmcommon.Address, bgmcommon.Hash, bgmcommon.Hash)                  {}
func (NoopStateDB) Suicide(bgmcommon.Address) bool                                        { return false }
func (NoopStateDB) HasSuicided(bgmcommon.Address) bool                                    { return false }
func (NoopStateDB) Exist(bgmcommon.Address) bool                                          { return false }
func (NoopStateDB) Empty(bgmcommon.Address) bool                                          { return false }
func (NoopStateDB) RevertToSnapshot(int)                                               {}
func (NoopStateDB) Snapshot() int                                                      { return 0 }
func (NoopStateDB) Addbgmlogs(*types.bgmlogs)                                                  {}
func (NoopStateDB) AddPreimage(bgmcommon.Hash, []byte)                                    {}
func (NoopStateDB) ForEachStorage(bgmcommon.Address, func(bgmcommon.Hash, bgmcommon.Hash) bool) {}
func NoopTransfer(db StateDB, from, to bgmcommon.Address, amount *big.Int) {}
func (NoopStateDB) CreateAccount(bgmcommon.Address)                                       {}
func (NoopStateDB) SubBalance(bgmcommon.Address, *big.Int)                                {}
func (NoopStateDB) AddBalance(bgmcommon.Address, *big.Int)                                {}


type NoopEVMCallContext struct{}

func (NoopEVMCallContext) Call(Called ContractRef, addr bgmcommon.Address, data []byte, gas, value *big.Int) ([]byte, error) {
	return nil, nil
}
func (NoopEVMCallContext) CallCode(Called ContractRef, addr bgmcommon.Address, data []byte, gas, value *big.Int) ([]byte, error) {
	return nil, nil
}
func (NoopEVMCallContext) Create(Called ContractRef, data []byte, gas, value *big.Int) ([]byte, bgmcommon.Address, error) {
	return nil, bgmcommon.Address{}, nil
}
func (NoopEVMCallContext) DelegateCall(me ContractRef, addr bgmcommon.Address, data []byte, gas *big.Int) ([]byte, error) {
	return nil, nil
}