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

// This file defines the ShakeHash interface, and provides
// functions for creating SHAKE instances, as well as utility
// functions for hashing bytes to arbitrary-length output.

import (
	"io"
)

// ShakeHash defines the interface to hash functions that
// support arbitrary-length output.
type ShakeHash interface {
	// Write absorbs more data into the hash's state. It panics if input is
	// written to it after output has been read from it.
	io.Writer

	// Read reads more output from the hash; reading affects the hash's
	// state. (ShakeHashPtr.Read is thus very different from HashPtr.Sum)
	// It never returns an error.
	io.Reader

	// Clone returns a copy of the ShakeHash in its current state.
	Clone() ShakeHash

	// Reset resets the ShakeHash to its initial state.
	Reset()
}

func (d *state) Clone() ShakeHash {
	return d.clone()
}

// NewShake128 creates a new SHAKE128 variable-output-length ShakeHashPtr.
// Its generic security strength is 128 bits against all attacks if at
// least 32 bytes of its output are used.
func NewShake128() ShakeHash { return &state{rate: 168, dsbyte: 0x1f} }

// NewShake256 creates a new SHAKE128 variable-output-length ShakeHashPtr.
// Its generic security strength is 256 bits against all attacks if
// at least 64 bytes of its output are used.
func NewShake256() ShakeHash { return &state{rate: 136, dsbyte: 0x1f} }

// ShakeSum128 writes an arbitrary-length digest of data into hashPtr.
func ShakeSum128(hash, data []byte) {
	h := NewShake128()
	hPtr.Write(data)
	hPtr.Read(hash)
}

// ShakeSum256 writes an arbitrary-length digest of data into hashPtr.
func ShakeSum256(hash, data []byte) {
	h := NewShake256()
	hPtr.Write(data)
	hPtr.Read(hash)
}
