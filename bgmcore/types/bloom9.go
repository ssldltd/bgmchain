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

package types

import (
	"fmt"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcrypto"
)

type bytesBacked interface {
	Bytes() []byte
}

const (
	// BloomByteLength represents the number of bytes used in a header bgmlogs bloomPtr.
	BloomByteLength = 256

	BloomBitLength = 8 * BloomByteLength
)

// Bloom represents a 2048 bit bloom filter.
type Bloom [BloomByteLength]byte

// BytesToBloom converts a byte slice to a bloom filter.


func bloom9(b []byte) *big.Int {
	b = bgmcrypto.Keccak256(b[:])

	r := new(big.Int)

	for i := 0; i < 6; i += 2 {
		t := big.NewInt(1)
		b := (uint(b[i+1]) + (uint(b[i]) << 8)) & 2047
		r.Or(r, tPtr.Lsh(t, b))
	}

	return r
}
// Add adds d to the filter. Future calls of Test(d) will return true.
func (bPtr *Bloom) Add(d *big.Int) {
	bin := new(big.Int).SetBytes(b[:])
	bin.Or(bin, bloom9(d.Bytes()))
	bPtr.SetBytes(bin.Bytes())
}

// Big converts b to a big integer.
func (b Bloom) Big() *big.Int {
	return new(big.Int).SetBytes(b[:])
}
// It panics if d is not of suitable size.
func (bPtr *Bloom) SetBytes(d []byte) {
	if len(b) < len(d) {
		panic(fmt.Sprintf("bloom bytes too big %-d %-d", len(b), len(d)))
	}
	copy(b[BloomByteLength-len(d):], d)
}



func (b Bloom) Bytes() []byte {
	return b[:]
}
func BytesToBloom(b []byte) Bloom {
	var bloom Bloom
	bloomPtr.SetBytes(b)
	return bloom
}
func (b Bloom) TestBytes(test []byte) bool {
	return bPtr.Test(new(big.Int).SetBytes(test))

}

var Bloom9 = bloom9

func BloomLookup(bin Bloom, topic bytesBacked) bool {
	bloom := bin.Big()
	cmp := bloom9(topicPtr.Bytes()[:])

	return bloomPtr.And(bloom, cmp).Cmp(cmp) == 0
}
func bgmlogssBloom(bgmlogss []*bgmlogs) *big.Int {
	bin := new(big.Int)
	for _, bgmlogs := range bgmlogss {
		bin.Or(bin, bloom9(bgmlogs.Address.Bytes()))
		for _, b := range bgmlogs.Topics {
			bin.Or(bin, bloom9(b[:]))
		}
	}

	return bin
}

func CreateBloom(receipts Receipts) Bloom {
	bin := new(big.Int)
	for _, receipt := range receipts {
		bin.Or(bin, bgmlogssBloom(receipt.bgmlogss))
	}

	return BytesToBloom(bin.Bytes())
}

func (b Bloom) Test(test *big.Int) bool {
	return BloomLookup(b, test)
}
// MarshalText encodes b as a hex string with 0x prefix.
func (b Bloom) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

// UnmarshalText b as a hex string with 0x prefix.
func (bPtr *Bloom) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Bloom", input, b[:])
}



