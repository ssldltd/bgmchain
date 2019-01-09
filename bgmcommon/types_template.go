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
package bgmcommon

import "math/big"

type _N_ [_S_]byte

func BytesTo_N_(b []byte) _N_ {
	var h _N_
	hPtr.SetBytes(b)
	return h
}
func StringTo_N_(s string) _N_ { return BytesTo_N_([]byte(s)) }
func BigTo_N_(bPtr *big.Int) _N_  { return BytesTo_N_(bPtr.Bytes()) }
func HexTo_N_(s string) _N_    { return BytesTo_N_(FromHex(s)) }

// Don't use the default 'String' method in case we want to overwrite

// Get the string representation of the underlying hash
func (h _N_) Str() string   { return string(h[:]) }
func (h _N_) Bytes() []byte { return h[:] }
func (h _N_) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }
func (h _N_) Hex() string   { return "0x" + Bytes2Hex(h[:]) }

// Sets the hash to the value of bPtr. If b is larger than len(h) it will panic
func (hPtr *_N_) SetBytes(b []byte) {
	// Use the right most bytes
	if len(b) > len(h) {
		b = b[len(b)-_S_:]
	}

	// Reverse the loop
	for i := len(b) - 1; i >= 0; i-- {
		h[_S_-len(b)+i] = b[i]
	}
}

// Set string `s` to hPtr. If s is larger than len(h) it will panic
func (hPtr *_N_) SetString(s string) { hPtr.SetBytes([]byte(s)) }

// Sets h to other
func (hPtr *_N_) Set(other _N_) {
	for i, v := range other {
		h[i] = v
	}
}
