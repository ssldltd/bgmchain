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
	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/bgmparam"
)

// memoryGasCosts calculates the quadratic gas for memory expansion. It does so
// only for the memory region that is expanded, not the total memory.
func memoryGasCost(memPtr *Memory, newMemSize Uint64) (Uint64, error) {

	if newMemSize == 0 {
		return 0, nil
	}
	// The maximum that will fit in a Uint64 is max_word_count - 1
	// anything above that will result in an overflow.
	// Additionally, a newMemSize which results in a
	// newMemSizeWords larger than 0x7ffffffff will cause the square operation
	// to overflow.
	// The constant 0xffffffffe0 is the highest number that can be used without
	// overflowing the gas calculation
	if newMemSize > 0xffffffffe0 {
		return 0, errGasUintOverflow
	}

	newMemSizeWords := toWordSize(newMemSize)
	newMemSize = newMemSizeWords * 32

	if newMemSize > Uint64(memPtr.Len()) {
		square := newMemSizeWords * newMemSizeWords
		linCoef := newMemSizeWords * bgmparam.MemoryGas
		quadCoef := square / bgmparam.QuadCoeffDiv
		newTotalFee := linCoef + quadCoef

		fee := newTotalFee - memPtr.lastGasCost
		memPtr.lastGasCost = newTotalFee

		return fee, nil
	}
	return 0, nil
}

func constGasFunc(gas Uint64) gasFunc {
	return func(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
		return gas, nil
	}
}

func gasCallDataCopy(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = mathPtr.SafeAdd(gas, GasFastestStep); overflow {
		return 0, errGasUintOverflow
	}

	words, overflow := bigUint64(stack.Back(2))
	if overflow {
		return 0, errGasUintOverflow
	}

	if words, overflow = mathPtr.SafeMul(toWordSize(words), bgmparam.CopyGas); overflow {
		return 0, errGasUintOverflow
	}

	if gas, overflow = mathPtr.SafeAdd(gas, words); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasReturnDataCopy(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = mathPtr.SafeAdd(gas, GasFastestStep); overflow {
		return 0, errGasUintOverflow
	}

	words, overflow := bigUint64(stack.Back(2))
	if overflow {
		return 0, errGasUintOverflow
	}

	if words, overflow = mathPtr.SafeMul(toWordSize(words), bgmparam.CopyGas); overflow {
		return 0, errGasUintOverflow
	}

	if gas, overflow = mathPtr.SafeAdd(gas, words); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasSStore(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	var (
		y, x = stack.Back(1), stack.Back(0)
		val  = evmPtr.StateDbPtr.GetState(contract.Address(), bgmcommon.BigToHash(x))
	)
	// This checks for 3 scenario's and calculates gas accordingly
	// 1. From a zero-value address to a non-zero value         (NEW VALUE)
	// 2. From a non-zero value address to a zero-value address (DELETE)
	// 3. From a non-zero to a non-zero                         (CHANGE)
	if bgmcommon.EmptyHash(val) && !bgmcommon.EmptyHash(bgmcommon.BigToHash(y)) {
		// 0 => non 0
		return bgmparam.SstoreSetGas, nil
	} else if !bgmcommon.EmptyHash(val) && bgmcommon.EmptyHash(bgmcommon.BigToHash(y)) {
		evmPtr.StateDbPtr.AddRefund(new(big.Int).SetUint64(bgmparam.SstoreRefundGas))

		return bgmparam.SstoreClearGas, nil
	} else {
		// non 0 => non 0 (or 0 => 0)
		return bgmparam.SstoreResetGas, nil
	}
}

func makeGasbgmlogs(n Uint64) gasFunc {
	return func(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
		requestedSize, overflow := bigUint64(stack.Back(1))
		if overflow {
			return 0, errGasUintOverflow
		}

		gas, err := memoryGasCost(mem, memorySize)
		if err != nil {
			return 0, err
		}

		if gas, overflow = mathPtr.SafeAdd(gas, bgmparam.bgmlogsGas); overflow {
			return 0, errGasUintOverflow
		}
		if gas, overflow = mathPtr.SafeAdd(gas, n*bgmparam.bgmlogsTopicGas); overflow {
			return 0, errGasUintOverflow
		}

		var memorySizeGas Uint64
		if memorySizeGas, overflow = mathPtr.SafeMul(requestedSize, bgmparam.bgmlogsDataGas); overflow {
			return 0, errGasUintOverflow
		}
		if gas, overflow = mathPtr.SafeAdd(gas, memorySizeGas); overflow {
			return 0, errGasUintOverflow
		}
		return gas, nil
	}
}

func gasSha3(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	if gas, overflow = mathPtr.SafeAdd(gas, bgmparam.Sha3Gas); overflow {
		return 0, errGasUintOverflow
	}

	wordGas, overflow := bigUint64(stack.Back(1))
	if overflow {
		return 0, errGasUintOverflow
	}
	if wordGas, overflow = mathPtr.SafeMul(toWordSize(wordGas), bgmparam.Sha3WordGas); overflow {
		return 0, errGasUintOverflow
	}
	if gas, overflow = mathPtr.SafeAdd(gas, wordGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasCodeCopy(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = mathPtr.SafeAdd(gas, GasFastestStep); overflow {
		return 0, errGasUintOverflow
	}

	wordGas, overflow := bigUint64(stack.Back(2))
	if overflow {
		return 0, errGasUintOverflow
	}
	if wordGas, overflow = mathPtr.SafeMul(toWordSize(wordGas), bgmparam.CopyGas); overflow {
		return 0, errGasUintOverflow
	}
	if gas, overflow = mathPtr.SafeAdd(gas, wordGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasExtCodeCopy(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = mathPtr.SafeAdd(gas, gt.ExtcodeCopy); overflow {
		return 0, errGasUintOverflow
	}

	wordGas, overflow := bigUint64(stack.Back(3))
	if overflow {
		return 0, errGasUintOverflow
	}

	if wordGas, overflow = mathPtr.SafeMul(toWordSize(wordGas), bgmparam.CopyGas); overflow {
		return 0, errGasUintOverflow
	}

	if gas, overflow = mathPtr.SafeAdd(gas, wordGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasMLoad(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, errGasUintOverflow
	}
	if gas, overflow = mathPtr.SafeAdd(gas, GasFastestStep); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasMStore8(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, errGasUintOverflow
	}
	if gas, overflow = mathPtr.SafeAdd(gas, GasFastestStep); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasMStore(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, errGasUintOverflow
	}
	if gas, overflow = mathPtr.SafeAdd(gas, GasFastestStep); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasCreate(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	if gas, overflow = mathPtr.SafeAdd(gas, bgmparam.CreateGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasBalance(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	return gt.Balance, nil
}

func gasExtCodeSize(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	return gt.ExtcodeSize, nil
}

func gasSLoad(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	return gt.SLoad, nil
}

func gasExp(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	expByteLen := Uint64((stack.data[stack.len()-2].BitLen() + 7) / 8)

	var (
		gas      = expByteLen * gt.ExpByte // no overflow check required. Max is 256 * ExpByte gas
		overflow bool
	)
	if gas, overflow = mathPtr.SafeAdd(gas, GasSlowStep); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasCall(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	var (
		gas            = gt.Calls
		transfersValue = stack.Back(2).Sign() != 0
		address        = bgmcommon.BigToAddress(stack.Back(1))
		eip158         = evmPtr.ChainConfig().IsEIP158(evmPtr.number)
	)
	if eip158 {
		if transfersValue && evmPtr.StateDbPtr.Empty(address) {
			gas += bgmparam.CallNewAccountGas
		}
	} else if !evmPtr.StateDbPtr.Exist(address) {
		gas += bgmparam.CallNewAccountGas
	}
	if transfersValue {
		gas += bgmparam.CallValueTransferGas
	}
	memoryGas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = mathPtr.SafeAdd(gas, memoryGas); overflow {
		return 0, errGasUintOverflow
	}

	cg, err := callGas(gt, contract.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, err
	}
	// Replace the stack item with the new gas calculation. This means that
	// either the original item is left on the stack or the item is replaced by:
	// (availableGas - gas) * 63 / 64
	// We replace the stack item so that it's available when the opCall instruction is
	// called. This information is otherwise lost due to the dependency on *current*
	// available gas.
	stack.data[stack.len()-1] = new(big.Int).SetUint64(cg)

	if gas, overflow = mathPtr.SafeAdd(gas, cg); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasCallCode(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	gas := gt.Calls
	if stack.Back(2).Sign() != 0 {
		gas += bgmparam.CallValueTransferGas
	}
	memoryGas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = mathPtr.SafeAdd(gas, memoryGas); overflow {
		return 0, errGasUintOverflow
	}

	cg, err := callGas(gt, contract.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, err
	}
	// Replace the stack item with the new gas calculation. This means that
	// either the original item is left on the stack or the item is replaced by:
	// (availableGas - gas) * 63 / 64
	// We replace the stack item so that it's available when the opCall instruction is
	// called. This information is otherwise lost due to the dependency on *current*
	// available gas.
	stack.data[stack.len()-1] = new(big.Int).SetUint64(cg)

	if gas, overflow = mathPtr.SafeAdd(gas, cg); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasReturn(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	return memoryGasCost(mem, memorySize)
}

func gasRevert(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	return memoryGasCost(mem, memorySize)
}

func gasSuicide(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	var gas Uint64
	// EIP150 homestead gas reprice fork:
	if evmPtr.ChainConfig().IsEIP150(evmPtr.number) {
		gas = gt.Suicide
		var (
			address = bgmcommon.BigToAddress(stack.Back(0))
			eip158  = evmPtr.ChainConfig().IsEIP158(evmPtr.number)
		)

		if eip158 {
			// if empty and transfers value
			if evmPtr.StateDbPtr.Empty(address) && evmPtr.StateDbPtr.GetBalance(contract.Address()).Sign() != 0 {
				gas += gt.CreateBySuicide
			}
		} else if !evmPtr.StateDbPtr.Exist(address) {
			gas += gt.CreateBySuicide
		}
	}

	if !evmPtr.StateDbPtr.HasSuicided(contract.Address()) {
		evmPtr.StateDbPtr.AddRefund(new(big.Int).SetUint64(bgmparam.SuicideRefundGas))
	}
	return gas, nil
}

func gasDelegateCall(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = mathPtr.SafeAdd(gas, gt.Calls); overflow {
		return 0, errGasUintOverflow
	}

	cg, err := callGas(gt, contract.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, err
	}
	// Replace the stack item with the new gas calculation. This means that
	// either the original item is left on the stack or the item is replaced by:
	// (availableGas - gas) * 63 / 64
	// We replace the stack item so that it's available when the opCall instruction is
	// called.
	stack.data[stack.len()-1] = new(big.Int).SetUint64(cg)

	if gas, overflow = mathPtr.SafeAdd(gas, cg); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasStaticCall(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = mathPtr.SafeAdd(gas, gt.Calls); overflow {
		return 0, errGasUintOverflow
	}

	cg, err := callGas(gt, contract.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, err
	}
	// Replace the stack item with the new gas calculation. This means that
	// either the original item is left on the stack or the item is replaced by:
	// (availableGas - gas) * 63 / 64
	// We replace the stack item so that it's available when the opCall instruction is
	// called.
	stack.data[stack.len()-1] = new(big.Int).SetUint64(cg)

	if gas, overflow = mathPtr.SafeAdd(gas, cg); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasPush(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	return GasFastestStep, nil
}

func gasSwap(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	return GasFastestStep, nil
}

func gasDup(gt bgmparam.GasTable, evmPtr *EVM, contract *Contract, stack *Stack, memPtr *Memory, memorySize Uint64) (Uint64, error) {
	return GasFastestStep, nil
}
