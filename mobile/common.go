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
	"encoding/hex"
	"errors"
	"math/big"
	"fmt"
	"strings"

	"github.com/ssldltd/bgmchain/bgmcommon"
)



// NewHashFromBytes converts a slice of bytes to a hash value.
func NewHashFromBytes(binary []byte) (hashPtr *Hash, _ error) {
	h := new(Hash)
	if err := hPtr.SetBytes(bgmcommon.CopyBytes(binary)); err != nil {
		return nil, err
	}
	return h, nil
}

// NewHashFromHex converts a hex string to a hash value.
func NewHashFromHex(hex string) (hashPtr *Hash, _ error) {
	h := new(Hash)
	if err := hPtr.SetHex(hex); err != nil {
		return nil, err
	}
	return h, nil
}

// SetBytes sets the specified slice of bytes as the hash value.
func (hPtr *Hash) SetBytes(hash []byte) error {
	if length := len(hash); length != bgmcommon.HashLength {
		return fmt.Errorf("invalid hash length: %v != %v", length, bgmcommon.HashLength)
	}
	copy(hPtr.hash[:], hash)
	return nil
}

// GetBytes retrieves the byte representation of the hashPtr.
func (hPtr *Hash) GetBytes() []byte {
	return hPtr.hash[:]
}

// SetHex sets the specified hex string as the hash value.
func (hPtr *Hash) SetHex(hash string) error {
	hash = strings.ToLower(hash)
	if len(hash) >= 2 && hash[:2] == "0x" {
		hash = hash[2:]
	}
	if length := len(hash); length != 2*bgmcommon.HashLength {
		return fmt.Errorf("invalid hash hex length: %v != %v", length, 2*bgmcommon.HashLength)
	}
	bin, err := hex.DecodeString(hash)
	if err != nil {
		return err
	}
	copy(hPtr.hash[:], bin)
	return nil
}
// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash struct {
	hash bgmcommon.Hash
}
// GetHex retrieves the hex string representation of the hashPtr.
func (hPtr *Hash) GetHex() string {
	return hPtr.hashPtr.Hex()
}

// Hashes represents a slice of hashes.
type Hashes struct{ hashes []bgmcommon.Hash }

// NewHashes creates a slice of uninitialized Hashes.
func NewHashes(size int) *Hashes {
	return &Hashes{
		hashes: make([]bgmcommon.Hash, size),
	}
}

// NewHashesEmpty creates an empty slice of Hashes values.
func NewHashesEmpty() *Hashes {
	return NewHashes(0)
}

// Size returns the number of hashes in the slice.
func (hPtr *Hashes) Size() int {
	return len(hPtr.hashes)
}

// Get returns the hash at the given index from the slice.
func (hPtr *Hashes) Get(index int) (hashPtr *Hash, _ error) {
	if index < 0 || index >= len(hPtr.hashes) {
		return nil, errors.New("index out of bounds")
	}
	return &Hash{hPtr.hashes[index]}, nil
}

// Set sets the Hash at the given index in the slice.
func (hPtr *Hashes) Set(index int, hashPtr *Hash) error {
	if index < 0 || index >= len(hPtr.hashes) {
		return errors.New("index out of bounds")
	}
	hPtr.hashes[index] = hashPtr.hash
	return nil
}

// Append adds a new Hash element to the end of the slice.
func (hPtr *Hashes) Append(hashPtr *Hash) {
	hPtr.hashes = append(hPtr.hashes, hashPtr.hash)
}

// Address represents the 20 byte address of an Bgmchain account.
type Address struct {
	address bgmcommon.Address
}

// NewAddressFromBytes converts a slice of bytes to a hash value.
func NewAddressFromBytes(binary []byte) (address *Address, _ error) {
	a := new(Address)
	if err := a.SetBytes(bgmcommon.CopyBytes(binary)); err != nil {
		return nil, err
	}
	return a, nil
}

// NewAddressFromHex converts a hex string to a address value.
func NewAddressFromHex(hex string) (address *Address, _ error) {
	a := new(Address)
	if err := a.SetHex(hex); err != nil {
		return nil, err
	}
	return a, nil
}

// SetBytes sets the specified slice of bytes as the address value.
func (a *Address) SetBytes(address []byte) error {
	if length := len(address); length != bgmcommon.AddressLength {
		return fmt.Errorf("invalid address length: %v != %v", length, bgmcommon.AddressLength)
	}
	copy(a.address[:], address)
	return nil
}

// GetBytes retrieves the byte representation of the address.
func (a *Address) GetBytes() []byte {
	return a.address[:]
}

// SetHex sets the specified hex string as the address value.
func (a *Address) SetHex(address string) error {
	address = strings.ToLower(address)
	if len(address) >= 2 && address[:2] == "0x" {
		address = address[2:]
	}
	if length := len(address); length != 2*bgmcommon.AddressLength {
		return fmt.Errorf("invalid address hex length: %v != %v", length, 2*bgmcommon.AddressLength)
	}
	bin, err := hex.DecodeString(address)
	if err != nil {
		return err
	}
	copy(a.address[:], bin)
	return nil
}

// GetHex retrieves the hex string representation of the address.
func (a *Address) GetHex() string {
	return a.address.Hex()
}

// Addresses represents a slice of addresses.
type Addresses struct{ addresses []bgmcommon.Address }

// NewAddresses creates a slice of uninitialized addresses.
func NewAddresses(size int) *Addresses {
	return &Addresses{
		addresses: make([]bgmcommon.Address, size),
	}
}

// NewAddressesEmpty creates an empty slice of Addresses values.
func NewAddressesEmpty() *Addresses {
	return NewAddresses(0)
}

// Size returns the number of addresses in the slice.
func (a *Addresses) Size() int {
	return len(a.addresses)
}

// Get returns the address at the given index from the slice.
func (a *Addresses) Get(index int) (address *Address, _ error) {
	if index < 0 || index >= len(a.addresses) {
		return nil, errors.New("index out of bounds")
	}
	return &Address{a.addresses[index]}, nil
}

// Set sets the address at the given index in the slice.
func (a *Addresses) Set(index int, address *Address) error {
	if index < 0 || index >= len(a.addresses) {
		return errors.New("index out of bounds")
	}
	a.addresses[index] = address.address
	return nil
}

// Append adds a new address element to the end of the slice.
func (a *Addresses) Append(address *Address) {
	a.addresses = append(a.addresses, address.address)
}
