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

// packBytesSlice将给定的字节打包为[L，V]作为规范表示
//字节切片
func packBytesSlice(bytes []byte, l int) []byte {
	len := packNum(reflect.ValueOf(l))
	return append(len, bgmcommon.RightPadBytes(bytes, (l+31)/32*32)...)
}

// packElement根据abi规范打包给定的反射值
// t
func packElement(t Type, reflectValue reflect.Value) []byte {
	switch tPtr.T {
	case IntTy, UintTy:
		return packNum(reflectValue)
	case StringTy:
		return packBytesSlice([]byte(reflectValue.String()), reflectValue.Len())
	case AddressTy:
		if reflectValue.Kind() == reflect.Array {
			reflectValue = mustArrayToByteSlice(reflectValue)
		}

		return bgmcommon.LeftPadBytes(reflectValue.Bytes(), 32)
	case BytesTy:
		if reflectValue.Kind() == reflect.Array {
			reflectValue = mustArrayToByteSlice(reflectValue)
		}
		return packBytesSlice(reflectValue.Bytes(), reflectValue.Len())
	case BoolTy:
		if reflectValue.Bool() {
			return mathPtr.PaddedBigBytes(bgmcommon.Big1, 32)
		} else {
			return mathPtr.PaddedBigBytes(bgmcommon.Big0, 32)
		}
	case FixedBytesTy, FunctionTy:
		if reflectValue.Kind() == reflect.Array {
			reflectValue = mustArrayToByteSlice(reflectValue)
		}
		return bgmcommon.RightPadBytes(reflectValue.Bytes(), 32)
	default:
		panic("Fatal: abi: fatal error")
	}
}

// packNum打包给定的数字（使用反射值）并将其转换为适当的数字表示
func packNum(value reflect.Value) []byte {
	switch kind := value.Kind(); kind {
	case reflect.Ptr:
		return U256(value.Interface().(*big.Int))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return U256(new(big.Int).SetUint64(value.Uint()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return U256(big.NewInt(value.Int()))
	default:
		panic("Fatal: abi: fatal error")
	}

}
