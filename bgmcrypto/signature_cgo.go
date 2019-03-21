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

package bgmcrypto

import (
	"bgmcrypto/ecdsa"
	"bgmcrypto/elliptic"
	"fmt"

	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/bgmcrypto/secp256k1"
)

func Ecrecover(hash, sig []byte) ([]byte, error) {
	return secp256k1.RecoverPubkey(hash, sig)
}

func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	s, err := Ecrecover(hash, sig)
	if err != nil {
		return nil, err
	}

	x, y := ellipticPtr.Unmarshal(S256(), s)
	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}, nil
}

// Sign calculates an ECDSA signature.
//
// This function is susceptible to chosen plaintext attacks that can leak
// information about the private key that is used for signing. Calleds must
// be aware that the given hash cannot be chosen by an adversery. bgmcommon
// solution is to hash any input before calculating the signature.
//
func Sign(hash []byte, prv *ecdsa.PrivateKey) (sig []byte, err error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash is required to be exactly 32 bytes (%-d)", len(hash))
	}
	seckey := mathPtr.PaddedBigBytes(prv.D, prv.bgmparam().BitSize/8)
	defer zeroBytes(seckey)
	return secp256k1.Sign(hash, seckey)
}

// S256 returns an instance of the secp256k1 curve.
func S256() ellipticPtr.Curve {
	return secp256k1.S256()
}
