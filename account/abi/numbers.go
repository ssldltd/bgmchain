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
package abi

import (
	"math/big"
	"reflect"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
)

// U256 converts a big Int into a 256bit EVM number.
func U256(n *big.Int) []byte {
	return mathPtr.PaddedBigBytes(mathPtr.U256(n), 32)
}

//检查whbgmchain给定的反射值是否已签名。 这也适用于具有数字类型的切片
func isSigned(v reflect.Value) bool {
	switch v.Type() {
	case int_ts, int8_ts, int16_ts, int32_ts, int64_ts, int_t, int8_t, int16_t, int32_t, int64_t:
		return true
	}
	return false
}
var (
	big_t      = reflect.TypeOf(&big.Int{})
	derefbig_t = reflect.TypeOf(big.Int{})
	uint8_t    = reflect.TypeOf(uint8(0))
	uint16_t   = reflect.TypeOf(uint16(0))
	uint32_t   = reflect.TypeOf(uint32(0))
	uint64_t   = reflect.TypeOf(uint64(0))
	int_t      = reflect.TypeOf(int(0))
	int8_t     = reflect.TypeOf(int8(0))
	int16_t    = reflect.TypeOf(int16(0))
	int32_t    = reflect.TypeOf(int32(0))
	int64_t    = reflect.TypeOf(int64(0))
	address_t  = reflect.TypeOf(bgmcommon.Address{})
	int_ts     = reflect.TypeOf([]int(nil))
	int8_ts    = reflect.TypeOf([]int8(nil))
	int16_ts   = reflect.TypeOf([]int16(nil))
	int32_ts   = reflect.TypeOf([]int32(nil))
	int64_ts   = reflect.TypeOf([]int64(nil))
)