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
	"bgmcrypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmparam"
)

var (
	errorInvalidChainId = errors.New("invalid chain id for signer")
)

// sigCache is used to cache the derived sender and contains
type sigCache struct {
	signer Signer
	from   bgmcommon.Address
}


// Sender may cache the address, allowing it to be used regardless of
// signing method. The cache is invalidated if the cached signer does
// not match the signer used in the current call.
func Sender(signer Signer, tx *Transaction) (bgmcommon.Address, error) {
	if sc := tx.fromPtr.Load(); sc != nil {
		sigCache := scPtr.(sigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}

	addr, err := signer.Sender(tx)
	if err != nil {
		return bgmcommon.Address{}, err
	}
	tx.fromPtr.Store(sigCache{signer: signer, from: addr})
	return addr, nil
}

// Signer encapsulates transaction signature handling. Note that this interface is not a
type Signer interface {
	// given signature.
	SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error)
	Hash(tx *Transaction) bgmcommon.Hash
	// Equal returns true if the given signer is the same as the receiver.
	Equal(Signer) bool
	Sender(tx *Transaction) (bgmcommon.Address, error)
	// SignatureValues returns the raw R, S, V values corresponding to the
	
}

// EIP155Transaction implements Signer using the EIP155 rules.
type EIP155Signer struct {
	chainId, chainIdMul *big.Int
}

func NewEIP155Signer(chainId *big.Int) EIP155Signer {
	if chainId == nil {
		chainId = new(big.Int)
	}
	return EIP155Signer{
		chainId:    chainId,
		chainIdMul: new(big.Int).Mul(chainId, big.NewInt(2)),
	}
}


// SignTx signs the transaction using the given signer and private key
func SignTx(tx *Transaction, s Signer, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := bgmcrypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}

func (s EIP155Signer) Equal(s2 Signer) bool {
	eip155, ok := s2.(EIP155Signer)
	return ok && eip155.chainId.Cmp(s.chainId) == 0
}

var big8 = big.NewInt(8)


// MakeSigner returns a Signer based on the given chain config and block number.
func MakeSigner(config *bgmparam.ChainConfig, blockNumber *big.Int) Signer {
	var signer Signer
	switch {
	case config.IsEIP155(blockNumber):
		signer = NewEIP155Signer(config.ChainId)
	case config.IsHomestead(blockNumber):
		signer = HomesteadSigner{}
	default:
		signer = FrontierSigner{}
	}
	return signer
}
// WithSignature returns a new transaction with the given signature. This signature

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s EIP155Signer) Hash(tx *Transaction) bgmcommon.Hash {
	return rlpHash([]interface{}{
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Payload,
		s.chainId, uint(0), uint(0),
	})
}
func (s EIP155Signer) Sender(tx *Transaction) (bgmcommon.Address, error) {
	if !tx.Protected() {
		return HomesteadSigner{}.Sender(tx)
	}
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return bgmcommon.Address{}, errorInvalidChainId
	}
	V := new(big.Int).Sub(tx.data.V, s.chainIdMul)
	V.Sub(V, big8)
	return recoverPlain(s.Hash(tx), tx.data.R, tx.data.S, V, true)
}
// HomesteadTransaction implements TransactionInterface using the
// homestead rules.
type HomesteadSigner struct{ FrontierSigner }

func (s HomesteadSigner) Equal(s2 Signer) bool {
	_, ok := s2.(HomesteadSigner)
	return ok
}


// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (hs HomesteadSigner) SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error) {
	return hs.FrontierSigner.SignatureValues(tx, sig)
}
// deriveChainId derives the chain id from the given v bgmparameter
func deriveChainId(v *big.Int) *big.Int {
	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}
func (hs HomesteadSigner) Sender(tx *Transaction) (bgmcommon.Address, error) {
	return recoverPlain(hs.Hash(tx), tx.data.R, tx.data.S, tx.data.V, true)
}

type FrontierSigner struct{}

func (s FrontierSigner) Equal(s2 Signer) bool {
	_, ok := s2.(FrontierSigner)
	return ok
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (fs FrontierSigner) SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %-d, want 65", len(sig)))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}
func recoverPlain(sighash bgmcommon.Hash, R, S, VbPtr *big.Int, homestead bool) (bgmcommon.Address, error) {
	if VbPtr.BitLen() > 8 {
		return bgmcommon.Address{}, errorInvalidSig
	}
	V := byte(VbPtr.Uint64() - 27)
	if !bgmcrypto.ValidateSignatureValues(V, R, S, homestead) {
		return bgmcommon.Address{}, errorInvalidSig
	}
	// encode the snature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the snature
	pub, err := bgmcrypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return bgmcommon.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return bgmcommon.Address{}, errors.New("invalid public key")
	}
	var addr bgmcommon.Address
	copy(addr[:], bgmcrypto.Keccak256(pub[1:])[12:])
	return addr, nil
}
// needs to be in the [R || S || V] format where V is 0 or 1.
func (s EIP155Signer) SignatureValues(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	R, S, V, err = HomesteadSigner{}.SignatureValues(tx, sig)
	if err != nil {
		return nil, nil, nil, err
	}
	if s.chainId.Sign() != 0 {
		V = big.NewInt(int64(sig[64] + 35))
		V.Add(V, s.chainIdMul)
	}
	return R, S, V, nil
}



func (fs FrontierSigner) Sender(tx *Transaction) (bgmcommon.Address, error) {
	return recoverPlain(fs.Hash(tx), tx.data.R, tx.data.S, tx.data.V, false)
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (fs FrontierSigner) Hash(tx *Transaction) bgmcommon.Hash {
	return rlpHash([]interface{}{
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Payload,
	})
}


