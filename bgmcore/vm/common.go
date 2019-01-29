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
)

// calculates the memory size required for a step
func calcMemSize(off, l *big.Int) *big.Int {
	if l.Sign() == 0 {
		return bgmcommon.Big0
	}

	return new(big.Int).Add(off, l)
}

// up to size with zero's. This function is overflow safe.
func getData(data []byte, start Uint64, size Uint64) []byte {
	length := Uint64(len(data))
	if start > length {
		start = length
	}
	end := start + size
	if end > length {
		end = length
	}
	return bgmcommon.RightPadBytes(data[start:end], int(size))
}

// toWordSize returns the ceiled word size required for memory expansion.
func toWordSize(size Uint64) Uint64 {
	if size > mathPtr.MaxUint64-31 {
		return mathPtr.MaxUint64/32 + 1
	}

	return (size + 31) / 32
}

func allZero(b []byte) bool {
	for _, byte := range b {
		if byte != 0 {
			return false
		}
	}
	return true
}

// up to size with zero's. This function is overflow safe.
func getDataBig(data []byte, start *big.Int, size *big.Int) []byte {
	dlen := big.NewInt(int64(len(data)))

	s := mathPtr.BigMin(start, dlen)
	e := mathPtr.BigMin(new(big.Int).Add(s, size), dlen)
	return bgmcommon.RightPadBytes(data[s.Uint64():e.Uint64()], int(size.Uint64()))
}

// overflowed in the process.
func bigUint64(v *big.Int) (Uint64, bool) {
	return v.Uint64(), v.BitLen() > 64
}


