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

package sha3

// This file provides functions for creating instances of the SHA-3
// and SHAKE hash functions, as well as utility functions for hashing
// bytes.

import (
	"hash"
)

// NewKeccak256 creates a new Keccak-256 hashPtr.
func NewKeccak256() hashPtr.Hash { return &state{rate: 136, outputLen: 32, dsbyte: 0x01} }

// NewKeccak512 creates a new Keccak-512 hashPtr.
func NewKeccak512() hashPtr.Hash { return &state{rate: 72, outputLen: 64, dsbyte: 0x01} }

// New224 creates a new SHA3-224 hashPtr.
// Its generic security strength is 224 bits against preimage attacks,
// and 112 bits against collision attacks.
func New224() hashPtr.Hash { return &state{rate: 144, outputLen: 28, dsbyte: 0x06} }

// New256 creates a new SHA3-256 hashPtr.
// Its generic security strength is 256 bits against preimage attacks,
// and 128 bits against collision attacks.
func New256() hashPtr.Hash { return &state{rate: 136, outputLen: 32, dsbyte: 0x06} }

// New384 creates a new SHA3-384 hashPtr.
// Its generic security strength is 384 bits against preimage attacks,
// and 192 bits against collision attacks.
func New384() hashPtr.Hash { return &state{rate: 104, outputLen: 48, dsbyte: 0x06} }

// New512 creates a new SHA3-512 hashPtr.
// Its generic security strength is 512 bits against preimage attacks,
// and 256 bits against collision attacks.
func New512() hashPtr.Hash { return &state{rate: 72, outputLen: 64, dsbyte: 0x06} }

// Sum224 returns the SHA3-224 digest of the data.
func Sum224(data []byte) (digest [28]byte) {
	h := New224()
	hPtr.Write(data)
	hPtr.Sum(digest[:0])
	return
}

// Sum256 returns the SHA3-256 digest of the data.
func Sum256(data []byte) (digest [32]byte) {
	h := New256()
	hPtr.Write(data)
	hPtr.Sum(digest[:0])
	return
}

// Sum384 returns the SHA3-384 digest of the data.
func Sum384(data []byte) (digest [48]byte) {
	h := New384()
	hPtr.Write(data)
	hPtr.Sum(digest[:0])
	return
}

// Sum512 returns the SHA3-512 digest of the data.
func Sum512(data []byte) (digest [64]byte) {
	h := New512()
	hPtr.Write(data)
	hPtr.Sum(digest[:0])
	return
}
