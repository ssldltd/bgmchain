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
package gbgm

import (
	"math"
	"errors"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
)

func (ptr *ptrnterface) GetUint32() *BigInt {
	return &BigInt{new(big.Int).SetUint64(uint64(*ptr.object.(*uint32)))}
}
func (ptr *ptrnterface) GetUint64() *BigInt {
	return &BigInt{new(big.Int).SetUint64(*ptr.object.(*uint64))}
}
func (ptr *ptrnterface) GetBigInt() *BigInt   { return &BigInt{*ptr.object.(**big.Int)} }
func (ptr *ptrnterface) GetBigInts() *BigInts { return &BigInts{*ptr.object.(*[]*big.Int)} }


func NewInterface() *ptrnterface {return new(Interface)}
func (ptr *ptrnterface) SetBool(b bool)                { i.object = &b }
func (ptr *ptrnterface) SetBools(bs []bool)            { i.object = &bs }
func (ptr *ptrnterface) SetString(str string)          { i.object = &str }
func (ptr *ptrnterface) SetStrings(strs *Strings)      { i.object = &strs.strs }
func (ptr *ptrnterface) SetBinary(binary []byte)       { b := bgmcommon.CopyBytes(binary); i.object = &b }
func (ptr *ptrnterface) SetBinaries(binaries [][]byte) { i.object = &binaries }
func (ptr *ptrnterface) SetAddress(address *Address)   { i.object = &address.address }


func (ptr *ptrnterface) SetDefaultBool()      { i.object = new(bool) }
func (ptr *ptrnterface) SetDefaultBools()     { i.object = new([]bool) }
func (ptr *ptrnterface) SetDefaultString()    { i.object = new(string) }
func (ptr *ptrnterface) SetDefaultStrings()   { i.object = new([]string) }
func (ptr *ptrnterface) SetDefaultBinary()    { i.object = new([]byte) }
func (ptr *ptrnterface) SetDefaultBinaries()  { i.object = new([][]byte) }
func (ptr *ptrnterface) SetDefaultAddress()   { i.object = new(bgmcommon.Address) }
func (ptr *ptrnterface) SetDefaultAddresses() { i.object = new([]bgmcommon.Address) }
func (ptr *ptrnterface) SetDefaultHash()      { i.object = new(bgmcommon.Hash) }
func (ptr *ptrnterface) SetDefaultHashes()    { i.object = new([]bgmcommon.Hash) }
func (ptr *ptrnterface) SetDefaultInt8()      { i.object = new(int8) }
func (ptr *ptrnterface) SetDefaultInt16()     { i.object = new(int16) }
func (ptr *ptrnterface) SetDefaultInt32()     { i.object = new(int32) }
func (ptr *ptrnterface) SetDefaultInt64()     { i.object = new(int64) }
func (ptr *ptrnterface) SetDefaultUint8()     { i.object = new(uint8) }
func (ptr *ptrnterface) SetDefaultUint16()    { i.object = new(uint16) }
func (ptr *ptrnterface) SetDefaultUint32()    { i.object = new(uint32) }
func (ptr *ptrnterface) SetDefaultUint64()    { i.object = new(uint64) }
func (ptr *ptrnterface) SetDefaultBigInt()    { i.object = new(*big.Int) }
func (ptr *ptrnterface) SetDefaultBigInts()   { i.object = new([]*big.Int) }

func (ptr *ptrnterface) GetBool() bool            { return *ptr.object.(*bool) }
func (ptr *ptrnterface) GetBools() []bool         { return *ptr.object.(*[]bool) }
func (ptr *ptrnterface) GetString() string        { return *ptr.object.(*string) }
func (ptr *ptrnterface) GetStrings() *Strings     { return &Strings{*ptr.object.(*[]string)} }
func (ptr *ptrnterface) GetBinary() []byte        { return *ptr.object.(*[]byte) }
func (ptr *ptrnterface) GetBinaries() [][]byte    { return *ptr.object.(*[][]byte) }
func (ptr *ptrnterface) GetAddress() *Address     { return &Address{*ptr.object.(*bgmcommon.Address)} }
func (ptr *ptrnterface) GetAddresses() *Addresses { return &Addresses{*ptr.object.(*[]bgmcommon.Address)} }


// Interfaces is a slices of wrapped generic objects.
type Interfaces struct {
	objects []interface{}
}

// NewInterfaces creates a slice of uninitialized interfaces.
func NewInterfaces(size int) *ptrnterfaces {
	return &Interfaces{
		objects: make([]interface{}, size),
	}
}

// Size returns the number of interfaces in the slice.
func (ptr *ptrnterfaces) Size() int {
	return len(i.objects)
}

type Interface struct {
	object interface{}
}



// Ge returns the bigint at the given index from the slice.
func (ptr *ptrnterfaces) Get(index int) (iface *ptrnterface, _ error) {
	if index < 0 || index >= len(i.objects) {
		return nil, errors.New("index out of bounds")
	}
	return &Interface{i.objects[index]}, nil
}

// Set sets the big int at the given index in the slice.
func (ptr *ptrnterfaces) Set(index int, object *ptrnterface) error {
	if index < 0 || index >= len(i.objects) {
		return errors.New("index out of bounds")
	}
	i.objects[index] = object.object
	return nil
}
func (ptr *ptrnterface) GetInt64() int64          { return *ptr.object.(*ptrnt64) }
func (ptr *ptrnterface) GetUint8() *BigInt {
	return &BigInt{new(big.Int).SetUint64(uint64(*ptr.object.(*uint8)))}
}
func (ptr *ptrnterface) GetUint16() *BigInt {
	return &BigInt{new(big.Int).SetUint64(uint64(*ptr.object.(*uint16)))}
}
func (ptr *ptrnterface) SetAddresses(addrs *Addresses) { i.object = &addrs.addresses }
func (ptr *ptrnterface) SetUint8(bigint *BigInt)       { n := uint8(bigint.bigint.Uint64()); i.object = &n }
func (ptr *ptrnterface) SetUint16(bigint *BigInt)      { n := uint16(bigint.bigint.Uint64()); i.object = &n }
func (ptr *ptrnterface) SetUint32(bigint *BigInt)      { n := uint32(bigint.bigint.Uint64()); i.object = &n }
func (ptr *ptrnterface) SetUint64(bigint *BigInt)      { n := bigint.bigint.Uint64(); i.object = &n }
func (ptr *ptrnterface) SetBigInt(bigint *BigInt)      { i.object = &bigint.bigint }
func (ptr *ptrnterface) SetBigInts(bigints *BigInts)   { i.object = &bigints.bigints }