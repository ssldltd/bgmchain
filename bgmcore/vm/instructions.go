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
	"errors"
	"fmt"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmparam"
)

var (
	bigZero                  = new(big.Int)
	errWriteProtection       = errors.New("evm: write protection")
	errReturnDataOutOfBounds = errors.New("evm: return data out of bounds")
	errExecutionReverted     = errors.New("evm: execution reverted")
	errMaxCodeSizeExceeded   = errors.New("evm: max code size exceeded")
)

func opAdd(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(mathPtr.U256(x.Add(x, y)))

	evmPtr.interpreter.intPool.put(y)

	return nil, nil
}

func opSub(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(mathPtr.U256(x.Sub(x, y)))

	evmPtr.interpreter.intPool.put(y)

	return nil, nil
}

func opMul(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(mathPtr.U256(x.Mul(x, y)))

	evmPtr.interpreter.intPool.put(y)

	return nil, nil
}

func opDiv(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if y.Sign() != 0 {
		stack.push(mathPtr.U256(x.Div(x, y)))
	} else {
		stack.push(new(big.Int))
	}

	evmPtr.interpreter.intPool.put(y)

	return nil, nil
}

func opSdiv(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := mathPtr.S256(stack.pop()), mathPtr.S256(stack.pop())
	if y.Sign() == 0 {
		stack.push(new(big.Int))
		return nil, nil
	} else {
		n := new(big.Int)
		if evmPtr.interpreter.intPool.get().Mul(x, y).Sign() < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		res := x.Div(x.Abs(x), y.Abs(y))
		res.Mul(res, n)

		stack.push(mathPtr.U256(res))
	}
	evmPtr.interpreter.intPool.put(y)
	return nil, nil
}

func opMod(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if y.Sign() == 0 {
		stack.push(new(big.Int))
	} else {
		stack.push(mathPtr.U256(x.Mod(x, y)))
	}
	evmPtr.interpreter.intPool.put(y)
	return nil, nil
}

func opSmod(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := mathPtr.S256(stack.pop()), mathPtr.S256(stack.pop())

	if y.Sign() == 0 {
		stack.push(new(big.Int))
	} else {
		n := new(big.Int)
		if x.Sign() < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		res := x.Mod(x.Abs(x), y.Abs(y))
		res.Mul(res, n)

		stack.push(mathPtr.U256(res))
	}
	evmPtr.interpreter.intPool.put(y)
	return nil, nil
}

func opExp(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	base, exponent := stack.pop(), stack.pop()
	stack.push(mathPtr.Exp(base, exponent))

	evmPtr.interpreter.intPool.put(base, exponent)

	return nil, nil
}

func opSignExtend(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	back := stack.pop()
	if back.Cmp(big.NewInt(31)) < 0 {
		bit := uint(back.Uint64()*8 + 7)
		num := stack.pop()
		mask := back.Lsh(bgmcommon.Big1, bit)
		mask.Sub(mask, bgmcommon.Big1)
		if numPtr.Bit(int(bit)) > 0 {
			numPtr.Or(num, mask.Not(mask))
		} else {
			numPtr.And(num, mask)
		}

		stack.push(mathPtr.U256(num))
	}

	evmPtr.interpreter.intPool.put(back)
	return nil, nil
}

func opNot(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x := stack.pop()
	stack.push(mathPtr.U256(x.Not(x)))
	return nil, nil
}

func opLt(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) < 0 {
		stack.push(evmPtr.interpreter.intPool.get().SetUint64(1))
	} else {
		stack.push(new(big.Int))
	}

	evmPtr.interpreter.intPool.put(x, y)
	return nil, nil
}

func opGt(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) > 0 {
		stack.push(evmPtr.interpreter.intPool.get().SetUint64(1))
	} else {
		stack.push(new(big.Int))
	}

	evmPtr.interpreter.intPool.put(x, y)
	return nil, nil
}

func opSlt(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := mathPtr.S256(stack.pop()), mathPtr.S256(stack.pop())
	if x.Cmp(mathPtr.S256(y)) < 0 {
		stack.push(evmPtr.interpreter.intPool.get().SetUint64(1))
	} else {
		stack.push(new(big.Int))
	}

	evmPtr.interpreter.intPool.put(x, y)
	return nil, nil
}

func opSgt(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := mathPtr.S256(stack.pop()), mathPtr.S256(stack.pop())
	if x.Cmp(y) > 0 {
		stack.push(evmPtr.interpreter.intPool.get().SetUint64(1))
	} else {
		stack.push(new(big.Int))
	}

	evmPtr.interpreter.intPool.put(x, y)
	return nil, nil
}

func opEq(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) == 0 {
		stack.push(evmPtr.interpreter.intPool.get().SetUint64(1))
	} else {
		stack.push(new(big.Int))
	}

	evmPtr.interpreter.intPool.put(x, y)
	return nil, nil
}

func opIszero(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x := stack.pop()
	if x.Sign() > 0 {
		stack.push(new(big.Int))
	} else {
		stack.push(evmPtr.interpreter.intPool.get().SetUint64(1))
	}

	evmPtr.interpreter.intPool.put(x)
	return nil, nil
}

func opAnd(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.And(x, y))

	evmPtr.interpreter.intPool.put(y)
	return nil, nil
}

func opOr(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.Or(x, y))

	evmPtr.interpreter.intPool.put(y)
	return nil, nil
}

func opXor(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.Xor(x, y))

	evmPtr.interpreter.intPool.put(y)
	return nil, nil
}

func opByte(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	th, val := stack.pop(), stack.peek()
	if thPtr.Cmp(bgmcommon.Big32) < 0 {
		b := mathPtr.Byte(val, 32, int(thPtr.Int64()))
		val.SetUint64(Uint64(b))
	} else {
		val.SetUint64(0)
	}
	evmPtr.interpreter.intPool.put(th)
	return nil, nil
}

func opAddmod(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Cmp(bigZero) > 0 {
		add := x.Add(x, y)
		add.Mod(add, z)
		stack.push(mathPtr.U256(add))
	} else {
		stack.push(new(big.Int))
	}

	evmPtr.interpreter.intPool.put(y, z)
	return nil, nil
}

func opMulmod(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Cmp(bigZero) > 0 {
		mul := x.Mul(x, y)
		mul.Mod(mul, z)
		stack.push(mathPtr.U256(mul))
	} else {
		stack.push(new(big.Int))
	}

	evmPtr.interpreter.intPool.put(y, z)
	return nil, nil
}

func opSha3(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	data := memory.Get(offset.Int64(), size.Int64())
	hash := bgmcrypto.Keccak256(data)

	if evmPtr.vmConfig.EnablePreimageRecording {
		evmPtr.StateDbPtr.AddPreimage(bgmcommon.BytesToHash(hash), data)
	}

	stack.push(new(big.Int).SetBytes(hash))

	evmPtr.interpreter.intPool.put(offset, size)
	return nil, nil
}

func opAddress(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(contract.Address().Big())
	return nil, nil
}

func opBalance(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	addr := bgmcommon.BigToAddress(stack.pop())
	balance := evmPtr.StateDbPtr.GetBalance(addr)

	stack.push(new(big.Int).Set(balance))
	return nil, nil
}

func opOrigin(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(evmPtr.Origin.Big())
	return nil, nil
}

func opCalled(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(contract.Called().Big())
	return nil, nil
}

func opCallValue(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(evmPtr.interpreter.intPool.get().Set(contract.value))
	return nil, nil
}

func opCallDataLoad(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(new(big.Int).SetBytes(getDataBig(contract.Input, stack.pop(), big32)))
	return nil, nil
}

func opCallDataSize(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(evmPtr.interpreter.intPool.get().SetInt64(int64(len(contract.Input))))
	return nil, nil
}

func opCallDataCopy(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		memOffset  = stack.pop()
		dataOffset = stack.pop()
		length     = stack.pop()
	)
	memory.Set(memOffset.Uint64(), lengthPtr.Uint64(), getDataBig(contract.Input, dataOffset, length))

	evmPtr.interpreter.intPool.put(memOffset, dataOffset, length)
	return nil, nil
}

func opReturnDataSize(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(evmPtr.interpreter.intPool.get().SetUint64(Uint64(len(evmPtr.interpreter.returnData))))
	return nil, nil
}

func opReturnDataCopy(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		memOffset  = stack.pop()
		dataOffset = stack.pop()
		length     = stack.pop()
	)
	defer evmPtr.interpreter.intPool.put(memOffset, dataOffset, length)

	end := new(big.Int).Add(dataOffset, length)
	if end.BitLen() > 64 || Uint64(len(evmPtr.interpreter.returnData)) < end.Uint64() {
		return nil, errReturnDataOutOfBounds
	}
	memory.Set(memOffset.Uint64(), lengthPtr.Uint64(), evmPtr.interpreter.returnData[dataOffset.Uint64():end.Uint64()])

	return nil, nil
}

func opExtCodeSize(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	a := stack.pop()

	addr := bgmcommon.BigToAddress(a)
	a.SetInt64(int64(evmPtr.StateDbPtr.GetCodeSize(addr)))
	stack.push(a)

	return nil, nil
}

func opCodeSize(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	l := evmPtr.interpreter.intPool.get().SetInt64(int64(len(contract.Code)))
	stack.push(l)
	return nil, nil
}

func opCodeCopy(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		memOffset  = stack.pop()
		codeOffset = stack.pop()
		length     = stack.pop()
	)
	codeCopy := getDataBig(contract.Code, codeOffset, length)
	memory.Set(memOffset.Uint64(), lengthPtr.Uint64(), codeCopy)

	evmPtr.interpreter.intPool.put(memOffset, codeOffset, length)
	return nil, nil
}

func opExtCodeCopy(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		addr       = bgmcommon.BigToAddress(stack.pop())
		memOffset  = stack.pop()
		codeOffset = stack.pop()
		length     = stack.pop()
	)
	codeCopy := getDataBig(evmPtr.StateDbPtr.GetCode(addr), codeOffset, length)
	memory.Set(memOffset.Uint64(), lengthPtr.Uint64(), codeCopy)

	evmPtr.interpreter.intPool.put(memOffset, codeOffset, length)
	return nil, nil
}

func opGasprice(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(evmPtr.interpreter.intPool.get().Set(evmPtr.GasPrice))
	return nil, nil
}

func opBlockhash(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	num := stack.pop()

	n := evmPtr.interpreter.intPool.get().Sub(evmPtr.number, bgmcommon.Big257)
	if numPtr.Cmp(n) > 0 && numPtr.Cmp(evmPtr.number) < 0 {
		stack.push(evmPtr.GetHash(numPtr.Uint64()).Big())
	} else {
		stack.push(new(big.Int))
	}

	evmPtr.interpreter.intPool.put(num, n)
	return nil, nil
}

func opCoinbase(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(evmPtr.Coinbase.Big())
	return nil, nil
}

func optimestamp(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(mathPtr.U256(new(big.Int).Set(evmPtr.time)))
	return nil, nil
}

func opNumber(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(mathPtr.U256(new(big.Int).Set(evmPtr.number)))
	return nil, nil
}

func opDifficulty(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(mathPtr.U256(new(big.Int).Set(evmPtr.Difficulty)))
	return nil, nil
}

func opGasLimit(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(mathPtr.U256(new(big.Int).Set(evmPtr.GasLimit)))
	return nil, nil
}

func opPop(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	evmPtr.interpreter.intPool.put(stack.pop())
	return nil, nil
}

func opMload(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset := stack.pop()
	val := new(big.Int).SetBytes(memory.Get(offset.Int64(), 32))
	stack.push(val)

	evmPtr.interpreter.intPool.put(offset)
	return nil, nil
}

func opMstore(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// pop value of the stack
	mStart, val := stack.pop(), stack.pop()
	memory.Set(mStart.Uint64(), 32, mathPtr.PaddedBigBytes(val, 32))

	evmPtr.interpreter.intPool.put(mStart, val)
	return nil, nil
}

func opMstore8(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	off, val := stack.pop().Int64(), stack.pop().Int64()
	memory.store[off] = byte(val & 0xff)

	return nil, nil
}

func opSload(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	loc := bgmcommon.BigToHash(stack.pop())
	val := evmPtr.StateDbPtr.GetState(contract.Address(), loc).Big()
	stack.push(val)
	return nil, nil
}

func opSstore(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	loc := bgmcommon.BigToHash(stack.pop())
	val := stack.pop()
	evmPtr.StateDbPtr.SetState(contract.Address(), loc, bgmcommon.BigToHash(val))

	evmPtr.interpreter.intPool.put(val)
	return nil, nil
}

func opJump(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	pos := stack.pop()
	if !contract.jumpdests.has(contract.CodeHash, contract.Code, pos) {
		nop := contract.GetOp(pos.Uint64())
		return nil, fmt.Errorf("invalid jump destination (%v) %v", nop, pos)
	}
	*pc = pos.Uint64()

	evmPtr.interpreter.intPool.put(pos)
	return nil, nil
}

func opJumpi(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	pos, cond := stack.pop(), stack.pop()
	if cond.Sign() != 0 {
		if !contract.jumpdests.has(contract.CodeHash, contract.Code, pos) {
			nop := contract.GetOp(pos.Uint64())
			return nil, fmt.Errorf("invalid jump destination (%v) %v", nop, pos)
		}
		*pc = pos.Uint64()
	} else {
		*pc++
	}

	evmPtr.interpreter.intPool.put(pos, cond)
	return nil, nil
}

func opJumpdest(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	return nil, nil
}

func opPc(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(evmPtr.interpreter.intPool.get().SetUint64(*pc))
	return nil, nil
}

func opMsize(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(evmPtr.interpreter.intPool.get().SetInt64(int64(memory.Len())))
	return nil, nil
}

func opGas(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(evmPtr.interpreter.intPool.get().SetUint64(contract.Gas))
	return nil, nil
}

func opCreate(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		value        = stack.pop()
		offset, size = stack.pop(), stack.pop()
		input        = memory.Get(offset.Int64(), size.Int64())
		gas          = contract.Gas
	)
	if evmPtr.ChainConfig().IsEIP150(evmPtr.number) {
		gas -= gas / 64
	}

	contract.UseGas(gas)
	res, addr, returnGas, suberr := evmPtr.Create(contract, input, gas, value)
	// Push item on the stack based on the returned error. If the ruleset is
	// homestead we must check for CodeStoreOutOfGasError (homestead only
	// rule) and treat as an error, if the ruleset is frontier we must
	// ignore this error and pretend the operation was successful.
	if evmPtr.ChainConfig().IsHomestead(evmPtr.number) && suberr == ErrCodeStoreOutOfGas {
		stack.push(new(big.Int))
	} else if suberr != nil && suberr != ErrCodeStoreOutOfGas {
		stack.push(new(big.Int))
	} else {
		stack.push(addr.Big())
	}
	contract.Gas += returnGas
	evmPtr.interpreter.intPool.put(value, offset, size)

	if suberr == errExecutionReverted {
		return res, nil
	}
	return nil, nil
}

func opCall(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	gas := stack.pop().Uint64()
	// pop gas and value of the stack.
	addr, value := stack.pop(), stack.pop()
	value = mathPtr.U256(value)
	// pop input size and offset
	inOffset, inSize := stack.pop(), stack.pop()
	// pop return size and offset
	retOffset, retSize := stack.pop(), stack.pop()

	address := bgmcommon.BigToAddress(addr)

	// Get the arguments from the memory
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if value.Sign() != 0 {
		gas += bgmparam.CallStipend
	}
	ret, returnGas, err := evmPtr.Call(contract, address, args, gas, value)
	if err != nil {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
	if err == nil || err == errExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	evmPtr.interpreter.intPool.put(addr, value, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

func opCallCode(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	gas := stack.pop().Uint64()
	// pop gas and value of the stack.
	addr, value := stack.pop(), stack.pop()
	value = mathPtr.U256(value)
	// pop input size and offset
	inOffset, inSize := stack.pop(), stack.pop()
	// pop return size and offset
	retOffset, retSize := stack.pop(), stack.pop()

	address := bgmcommon.BigToAddress(addr)

	// Get the arguments from the memory
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if value.Sign() != 0 {
		gas += bgmparam.CallStipend
	}

	ret, returnGas, err := evmPtr.CallCode(contract, address, args, gas, value)
	if err != nil {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
	if err == nil || err == errExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	evmPtr.interpreter.intPool.put(addr, value, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

func opDelegateCall(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	gas, to, inOffset, inSize, outOffset, outSize := stack.pop().Uint64(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()

	toAddr := bgmcommon.BigToAddress(to)
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	ret, returnGas, err := evmPtr.DelegateCall(contract, toAddr, args, gas)
	if err != nil {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
	if err == nil || err == errExecutionReverted {
		memory.Set(outOffset.Uint64(), outSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	evmPtr.interpreter.intPool.put(to, inOffset, inSize, outOffset, outSize)
	return ret, nil
}

func opStaticCall(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// pop gas
	gas := stack.pop().Uint64()
	// pop address
	addr := stack.pop()
	// pop input size and offset
	inOffset, inSize := stack.pop(), stack.pop()
	// pop return size and offset
	retOffset, retSize := stack.pop(), stack.pop()

	address := bgmcommon.BigToAddress(addr)

	// Get the arguments from the memory
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	ret, returnGas, err := evmPtr.StaticCall(contract, address, args, gas)
	if err != nil {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
	if err == nil || err == errExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	evmPtr.interpreter.intPool.put(addr, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

func opReturn(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	evmPtr.interpreter.intPool.put(offset, size)
	return ret, nil
}

func opRevert(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	evmPtr.interpreter.intPool.put(offset, size)
	return ret, nil
}

func opStop(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	return nil, nil
}

func opSuicide(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	balance := evmPtr.StateDbPtr.GetBalance(contract.Address())
	evmPtr.StateDbPtr.AddBalance(bgmcommon.BigToAddress(stack.pop()), balance)

	evmPtr.StateDbPtr.Suicide(contract.Address())
	return nil, nil
}

// following functions are used by the instruction jump  table

// make bgmlogs instruction function
func makebgmlogs(size int) executionFunc {
	return func(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		topics := make([]bgmcommon.Hash, size)
		mStart, mSize := stack.pop(), stack.pop()
		for i := 0; i < size; i++ {
			topics[i] = bgmcommon.BigToHash(stack.pop())
		}

		d := memory.Get(mStart.Int64(), mSize.Int64())
		evmPtr.StateDbPtr.Addbgmlogs(&types.bgmlogs{
			Address: contract.Address(),
			Topics:  topics,
			Data:    d,
			// This is a non-consensus field, but assigned here because
			// bgmCore/state doesn't know the current block number.
			number: evmPtr.number.Uint64(),
		})

		evmPtr.interpreter.intPool.put(mStart, mSize)
		return nil, nil
	}
}

// make push instruction function
func makePush(size Uint64, pushByteSize int) executionFunc {
	return func(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		codeLen := len(contract.Code)

		startMin := codeLen
		if int(*pc+1) < startMin {
			startMin = int(*pc + 1)
		}

		endMin := codeLen
		if startMin+pushByteSize < endMin {
			endMin = startMin + pushByteSize
		}

		integer := evmPtr.interpreter.intPool.get()
		stack.push(integer.SetBytes(bgmcommon.RightPadBytes(contract.Code[startMin:endMin], pushByteSize)))

		*pc += size
		return nil, nil
	}
}

// make push instruction function
func makeDup(size int64) executionFunc {
	return func(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		stack.dup(evmPtr.interpreter.intPool, int(size))
		return nil, nil
	}
}

// make swap instruction function
func makeSwap(size int64) executionFunc {
	// switch n + 1 otherwise n would be swapped with n
	size += 1
	return func(pcPtr *Uint64, evmPtr *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		stack.swap(int(size))
		return nil, nil
	}
}
