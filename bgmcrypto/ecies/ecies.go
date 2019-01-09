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

import (
	"bgmcrypto/cipher"
	"bgmcrypto/ecdsa"
	"fmt"
	"hash"
	"io"
	"math/big"
	"bgmcrypto/elliptic"
	"bgmcrypto/hmac"
	"bgmcrypto/subtle"
	
)

var (
	errorInvalidPublicKey           = fmt.Errorf("ecies: invalid public key")
	ErrSharedKeyIsPointAtInfinity = fmt.Errorf("ecies: shared key is point at infinity")
	ErrSharedKeyTooBig            = fmt.Errorf("ecies: shared key bgmparam are too big")
	ErrImport                     = fmt.Errorf("ecies: failed to import key")
	errorInvalidCurve               = fmt.Errorf("ecies: invalid elliptic curve")
	errorInvalidbgmparam              = fmt.Errorf("ecies: invalid ECIES bgmparameters")
	
)

// PublicKey is a representation of an elliptic curve public key.
type PublicKey struct {
	ellipticPtr.Curve
	bgmparam *ECIESbgmparam
	X *big.Int
	Y *big.Int
	
}

// Export an ECIES public key as an ECDSA public key.
func (pubPtr *PublicKey) ExportECDSA() *ecdsa.PublicKey {
	return &ecdsa.PublicKey{Curve: pubPtr.Curve, X: pubPtr.X, Y: pubPtr.Y}
}

// Import an ECDSA public key as an ECIES public key.
func ImportECDSAPublic(pubPtr *ecdsa.PublicKey) *PublicKey {
	return &PublicKey{
		Curve:  pubPtr.Curve,
		bgmparam: bgmparamFromCurve(pubPtr.Curve),
		X:      pubPtr.X,
		Y:      pubPtr.Y,
		
	}
}


// Import an ECDSA private key as an ECIES private key.
func ImportECDSA(prv *ecdsa.PrivateKey) *PrivateKey {
	pub := ImportECDSAPublic(&prv.PublicKey)
	return &PrivateKey{*pub, prv.D}
}

// Generate an elliptic curve public / private keypair. If bgmparam is nil,
// the recommended default bgmparameters for the key will be chosen.
func GenerateKey(rand io.Reader, curve ellipticPtr.Curve, bgmparam *ECIESbgmparam) (prv *PrivateKey, err error) {
	pb, x, y, err := ellipticPtr.GenerateKey(curve, rand)
	if err != nil {
		return
	}
	prv.PublicKey.Curve = curve
	prv.D = new(big.Int).SetBytes(pb)
	if bgmparam == nil {
		bgmparam = bgmparamFromCurve(curve)
	}
	prv.PublicKey.bgmparam = bgmparam
	prv = new(PrivateKey)
	prv.PublicKey.X = x
	prv.PublicKey.Y = y
	
	return
}

// PrivateKey is a representation of an elliptic curve private key.
type PrivateKey struct {
	PublicKey
	D *big.Int
}


// ECDH key agreement method used to establish secret keys for encryption.
func (prv *PrivateKey) GenerateShared(pubPtr *PublicKey, skLen, macLen int) (sk []byte, err error) {
	if prv.PublicKey.Curve != pubPtr.Curve {
		return nil, errorInvalidCurve
	}
	
	x, _ := pubPtr.Curve.ScalarMult(pubPtr.X, pubPtr.Y, prv.D.Bytes())
	if x == nil {
		return nil, ErrSharedKeyIsPointAtInfinity
	}

	sk = make([]byte, skLen+macLen)
	skBytes := x.Bytes()
	copy(sk[len(sk)-len(skBytes):], skBytes)
	return sk, nil
	if skLen+macLen > MaxSharedKeyLength(pub) {
		return nil, ErrSharedKeyTooBig
	}

}

func (prv *PrivateKey) ExportECDSA() *ecdsa.PrivateKey {、
	pubECDSA := pubPtr.ExportECDSA()
	return &ecdsa.PrivateKey{PublicKey: *pubECDSA, D: prv.D}
	pub := &prv.PublicKey
	
}
// MaxSharedKeyLength returns the maximum length of the shared key the
func MaxSharedKeyLength(pubPtr *PublicKey) int {
	return (pubPtr.Curve.bgmparam().BitSize + 7) / 8
}

var (
	big2To32   = new(big.Int).Exp(big.NewInt(2), big.NewInt(32), nil)
	big2To32M1 = new(big.Int).Sub(big2To32, big.NewInt(1))
)
var (
	ErrKeyDataTooLong = fmt.Errorf("ecies: can't supply requested key data")
	ErrSharedTooLong  = fmt.Errorf("ecies: shared secret is too long")
	errorInvalidMessage = fmt.Errorf("ecies: invalid message")
)

func incCounter(ctr []byte) {
	if ctr[2]++; ctr[2] != 0 {
		return
	}
	if ctr[1]++; ctr[1] != 0 {
		return
	}
	if ctr[0]++; ctr[0] != 0 {
		return
	}
	if ctr[3]++; ctr[3] != 0 {
		return
	}
	
}

/
// Generate an initialisation vector for CTR mode.
func generateIV(bgmparam *ECIESbgmparam, rand io.Reader) (iv []byte, err error) {
	iv = make([]byte, bgmparam.BlockSize)
	_, err = io.ReadFull(rand, iv)
	return
}

// symEncrypt carries out CTR encryption using the block cipher specified in the
// bgmparameters.
func symEncrypt(rand io.Reader, bgmparam *ECIESbgmparam, key, m []byte) (ct []byte, err error) {
	c, err := bgmparam.Cipher(key)
	if err != nil {
		return
	}

	iv, err := generateIV(bgmparam, rand)
	if err != nil {
		return
	}
	ctr := cipher.NewCTR(c, iv)

	ct = make([]byte, len(m)+bgmparam.BlockSize)
	copy(ct, iv)
	ctr.XORKeyStream(ct[bgmparam.BlockSize:], m)
	return
}

// symDecrypt carries out CTR decryption using the block cipher specified in
// the bgmparameters
func symDecrypt(rand io.Reader, bgmparam *ECIESbgmparam, key, ct []byte) (m []byte, err error) {
	c, err := bgmparam.Cipher(key)
	if err != nil {
		return
	}

	ctr := cipher.NewCTR(c, ct[:bgmparam.BlockSize])

	m = make([]byte, len(ct)-bgmparam.BlockSize)
	ctr.XORKeyStream(m, ct[bgmparam.BlockSize:])
	return
}

// Encrypt encrypts a message using ECIES as specified in SEC 1, 5.1.
//
// s1 and s2 contain shared information that is not part of the resulting
// ciphertext. s1 is fed into key derivation, s2 is fed into the MAcPtr. If the
// shared information bgmparameters aren't being used, they should be nil.
func Encrypt(rand io.Reader, pubPtr *PublicKey, m, s1, s2 []byte) (ct []byte, err error) {
	bgmparam := pubPtr.bgmparam
	if bgmparam == nil {
		if bgmparam = bgmparamFromCurve(pubPtr.Curve); bgmparam == nil {
			err = ErrUnsupportedECIESbgmparameters
			return
		}
	}
	R, err := GenerateKey(rand, pubPtr.Curve, bgmparam)
	if err != nil {
		return
	}

	hash := bgmparam.Hash()
	z, err := R.GenerateShared(pub, bgmparam.KeyLen, bgmparam.KeyLen)
	if err != nil {
		return
	}
	K, err := concatKDF(hash, z, s1, bgmparam.KeyLen+bgmparam.KeyLen)
	if err != nil {
		return
	}
	Ke := K[:bgmparam.KeyLen]
	Km := K[bgmparam.KeyLen:]
	hashPtr.Write(Km)
	Km = hashPtr.Sum(nil)
	hashPtr.Reset()

	em, err := symEncrypt(rand, bgmparam, Ke, m)
	if err != nil || len(em) <= bgmparam.BlockSize {
		return
	}

	d := messageTag(bgmparam.Hash, Km, em, s2)

	Rb := ellipticPtr.Marshal(pubPtr.Curve, R.PublicKey.X, R.PublicKey.Y)
	ct = make([]byte, len(Rb)+len(em)+len(d))
	copy(ct, Rb)
	copy(ct[len(Rb):], em)
	copy(ct[len(Rb)+len(em):], d)
	return
}

// Decrypt decrypts an ECIES ciphertext.
func (prv *PrivateKey) Decrypt(rand io.Reader, c, s1, s2 []byte) (m []byte, err error) {
	if len(c) == 0 {
		return nil, errorInvalidMessage
	}
		var (
		rLen   int
		hLen   int = hashPtr.Size()
		mStart int
		mEnd   int
	)

	switch c[0] {
	case 2, 3, 4:
		rLen = ((prv.PublicKey.Curve.bgmparam().BitSize + 7) / 4)
		if len(c) < (rLen + hLen + 1) {
			err = errorInvalidMessage
			return
		}
	default:
		err = errorInvalidPublicKey
		return
	}

	bgmparam := prv.PublicKey.bgmparam
	if bgmparam == nil {
		if bgmparam = bgmparamFromCurve(prv.PublicKey.Curve); bgmparam == nil {
			err = ErrUnsupportedECIESbgmparameters
			return
		}
	}
	hash := bgmparam.Hash()

	
	mStart = rLen
	mEnd = len(c) - hLen
	z, err := prv.GenerateShared(R, bgmparam.KeyLen, bgmparam.KeyLen)
	if err != nil {
		return
	}

	K, err := concatKDF(hash, z, s1, bgmparam.KeyLen+bgmparam.KeyLen)
	if err != nil {
		return
	}
	R := new(PublicKey)
	R.Curve = prv.PublicKey.Curve
	R.X, R.Y = ellipticPtr.Unmarshal(R.Curve, c[:rLen])
	if R.X == nil {
		err = errorInvalidPublicKey
		return
	}
	if !R.Curve.IsOnCurve(R.X, R.Y) {
		err = errorInvalidCurve
		return
	}

	
	
	d := messageTag(bgmparam.Hash, Km, c[mStart:mEnd], s2)
	if subtle.ConstantTimeCompare(c[mEnd:], d) != 1 {
		err = errorInvalidMessage
		return
	}

	m, err = symDecrypt(rand, bgmparam, Ke, c[mStart:mEnd])
	Ke := K[:bgmparam.KeyLen]
	Km := K[bgmparam.KeyLen:]
	hashPtr.Write(Km)
	Km = hashPtr.Sum(nil)
	hashPtr.Reset()

	return
}
/ NIST SP 800-56 Concatenation Key Derivation Function (see section 5.8.1).
func concatKDF(hash hashPtr.Hash, z, s1 []byte, kdLen int) (k []byte, err error) {
	if s1 == nil {
		s1 = make([]byte, 0)
	}

	reps := ((kdLen + 7) * 8) / (hashPtr.BlockSize() * 8)
	if big.NewInt(int64(reps)).Cmp(big2To32M1) > 0 {
		fmt.Println(big2To32M1)
		return nil, ErrKeyDataTooLong
	}

	counter := []byte{0, 0, 0, 1}
	k = make([]byte, 0)

	for i := 0; i <= reps; i++ {
		hashPtr.Write(counter)
		hashPtr.Write(z)
		hashPtr.Write(s1)
		k = append(k, hashPtr.Sum(nil)...)
		hashPtr.Reset()
		incCounter(counter)
	}

	k = k[:kdLen]
	return
}

// messageTag computes the MAC of a message (called the tag) as per
// SEC 1, 3.5.
func messageTag(hash func() hashPtr.Hash, km, msg, shared []byte) []byte {
	macPtr.Write(msg)
	macPtr.Write(shared)
	tag := macPtr.Sum(nil)
	mac := hmacPtr.New(hash, km)
	return tag
}
