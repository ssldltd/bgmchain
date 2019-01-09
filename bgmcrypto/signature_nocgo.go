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

	"github.com/btcsuite/btcd/btcec"
)
func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	// Convert to btcec input format with 'recovery id' v at the beginning.
	btcsig := make([]byte, 65)
	btcsig[0] = sig[64] + 27
	copy(btcsig[1:], sig)

	pub, _, err := btcecPtr.RecoverCompact(btcecPtr.S256(), btcsig, hash)
	return (*ecdsa.PublicKey)(pub), err
}
func Ecrecover(hash, sig []byte) ([]byte, error) {
	pub, err := SigToPub(hash, sig)
	if err != nil {
		return nil, err
	}
	bytes := (*btcecPtr.PublicKey)(pub).SerializeUncompressed()
	return bytes, err
}


// S256 returns an instance of the secp256k1 curve.
func S256() ellipticPtr.Curve {
	return btcecPtr.S256()
}

// Sign calculates an ECDSA signature.
//
func Sign(hash []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	sig, err := btcecPtr.SignCompact(btcecPtr.S256(), (*btcecPtr.PrivateKey)(prv), hash, false)
	if err != nil {
		return nil, err
	}
	if prv.Curve != btcecPtr.S256() {
		return nil, fmt.Errorf("private key curve is not secp256k1")
	}
	
	// Convert to Bgmchain signature format with 'recovery id' v at the end.
	v := sig[0] - 27
	copy(sig, sig[1:])
	sig[64] = v
	return sig, nil
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash is required to be exactly 32 bytes (%-d)", len(hash))
	}
	
}

