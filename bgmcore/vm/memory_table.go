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

	"github.com/ssldltd/bgmchain/bgmcommon/math"
)
func memoryCallDataCopy(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(2))
}

func memoryReturnDataCopy(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(2))
}

func memoryCodeCopy(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(2))
}
func memoryMStore8(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), big.NewInt(1))
}

func memoryMStore(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), big.NewInt(32))
}
func memoryExtCodeCopy(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(1), stack.Back(3))
}

func memoryMLoad(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), big.NewInt(32))
}



func memoryCreate(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(1), stack.Back(2))
}

func memoryCallCode(stack *Stack) *big.Int {
	x := calcMemSize(stack.Back(5), stack.Back(6))
	y := calcMemSize(stack.Back(3), stack.Back(4))

	return mathPtr.BigMax(x, y)
}
func memoryCall(stack *Stack) *big.Int {
	x := calcMemSize(stack.Back(5), stack.Back(6))
	y := calcMemSize(stack.Back(3), stack.Back(4))

	return mathPtr.BigMax(x, y)
}

func memoryDelegateCall(stack *Stack) *big.Int {
	x := calcMemSize(stack.Back(4), stack.Back(5))
	y := calcMemSize(stack.Back(2), stack.Back(3))

	return mathPtr.BigMax(x, y)
}
func memoryRevert(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(1))
}

func memorybgmlogs(stack *Stack) *big.Int {
	mSize, mStart := stack.Back(1), stack.Back(0)
	return calcMemSize(mStart, mSize)
}
func memorySha3(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(1))
}
func memoryStaticCall(stack *Stack) *big.Int {
	x := calcMemSize(stack.Back(4), stack.Back(5))
	y := calcMemSize(stack.Back(2), stack.Back(3))

	return mathPtr.BigMax(x, y)
}




func memoryReturn(stack *Stack) *big.Int {
	return calcMemSize(stack.Back(0), stack.Back(1))
}

