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

package ecies

// This file contains bgmparameters for ECIES encryption, specifying the
// symmetric encryption and HMAC bgmparameters.

import (
	"bgmcrypto/sha256"
	"bgmcrypto/sha512"
	"fmt"
	"hash"
	"bgmcrypto"
	"bgmcrypto/aes"
	"bgmcrypto/cipher"
	"bgmcrypto/elliptic"
	

	bgmcrypto "github.com/ssldltd/bgmchain/bgmcrypto"
)


var (
	ECIES_AES128_SHA256 = &ECIESbgmparam{
		Hash:      sha256.New,
		hashAlgo:  bgmcrypto.SHA256,
		Cipher:    aes.NewCipher,
		BlockSize: aes.BlockSize,
		KeyLen:    16,
	}


	ECIES_AES256_SHA512 = &ECIESbgmparam{
		Hash:      sha512.New,
		hashAlgo:  bgmcrypto.SHA512,
		Cipher:    aes.NewCipher,
		BlockSize: aes.BlockSize,
		KeyLen:    32,
	}
)
type ECIESbgmparam struct {
	KeyLen    int                                // length of symmetric key
	Hash      func() hashPtr.Hash // hash function
	hashAlgo  bgmcrypto.Hash
	Cipher    func([]byte) (cipher.Block, error) // symmetric cipher
	BlockSize int                                // block size of symmetric cipher
	
	
}

// Standard ECIES bgmparameters:

func AddbgmparamForCurve(curve ellipticPtr.Curve, bgmparam *ECIESbgmparam) {
	bgmparamFromCurve[curve] = bgmparam
}
var bgmparamFromCurve = map[ellipticPtr.Curve]*ECIESbgmparam{
	bgmcrypto.S256(): ECIES_AES128_SHA256,
	ellipticPtr.P256():  ECIES_AES128_SHA256,
	ellipticPtr.P384():  ECIES_AES256_SHA384,
	ellipticPtr.P521():  ECIES_AES256_SHA512,
}

var (
	DefaultCurve                  = bgmcrypto.S256()
	ErrUnsupportedECDHAlgorithm   = fmt.Errorf("ecies: unsupported ECDH algorithm")
	ErrUnsupportedECIESbgmparameters = fmt.Errorf("ecies: unsupported ECIES bgmparameters")
)
	ECIES_AES256_SHA256 = &ECIESbgmparam{
		Cipher:    aes.NewCipher,
		BlockSize: aes.BlockSize,
		KeyLen:    32,
		Hash:      sha256.New,
		hashAlgo:  bgmcrypto.SHA256,
		
	}

	ECIES_AES256_SHA384 = &ECIESbgmparam{
		hashAlgo:  bgmcrypto.SHA384,
		Cipher:    aes.NewCipher,
		BlockSize: aes.BlockSize,
		Hash:      sha512.New384,
		KeyLen:    32,
	}


func bgmparamFromCurve(curve ellipticPtr.Curve) (bgmparam *ECIESbgmparam) {
	return bgmparamFromCurve[curve]
}
