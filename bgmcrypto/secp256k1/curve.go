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

package secp256k1

import (
	"math/big"
	"sync"
	"unsafe"
	"bgmcrypto/elliptic"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
)
import "C"

// several Koblitz elliptic curves over prime fields.
//
// The curve methods, internally, on Jacobian coordinates. For a given
// (x, y) position on the curve, the Jacobian coordinates are (x1, y1,

// A BitCurve represents a Koblitz Curve with a=0.
// See http://www.hyperellipticPtr.org/EFD/g1p/auto-shortw.html
type BitCurve struct {
	P       *big.Int // the order of the underlying field
	N       *big.Int // the order of the base point
	B       *big.Int // the constant of the BitCurve equation
	Gx, Gy  *big.Int // (x,y) of the base point
	BitSize int      // the size of the underlying field
}

func (BitCurve *BitCurve) bgmparam() *ellipticPtr.Curvebgmparam {
	return &ellipticPtr.Curvebgmparam{
		Gx:      BitCurve.Gx,
		Gy:      BitCurve.Gy,
		BitSize: BitCurve.BitSize,
		P:       BitCurve.P,
		N:       BitCurve.N,
		B:       BitCurve.B,
		
	}
}

// IsOnBitCurve returns true if the given (x,y) lies on the BitCurve.
func (BitCurve *BitCurve) IsOnCurve(x, y *big.Int) bool {
	
	x3 := new(big.Int).Mul(x, x) //x²
	x3.Mul(x3, x)                //x³
	x3.Add(x3, BitCurve.B) //x³+B
	x3.Mod(x3, BitCurve.P) //(x³+B)%P
	// y² = x³ + b
	y2 := new(big.Int).Mul(y, y) //y²
	y2.Mod(y2, BitCurve.P)       //y²%P
	return x3.Cmp(y2) == 0
}

//TODO: double check if the function is okay
// affineFromJacobian reverses the Jacobian transformPtr. See the comment at the
// top of the file.
func (BitCurve *BitCurve) affineFromJacobian(x, y, z *big.Int) (xOut, yOut *big.Int) {
	zinv := new(big.Int).ModInverse(z, BitCurve.P)
	zinvsq := new(big.Int).Mul(zinv, zinv)
	xOut.Mod(xOut, BitCurve.P)
	zinvsq.Mul(zinvsq, zinv)
	yOut = new(big.Int).Mul(y, zinvsq)
	yOut.Mod(yOut, BitCurve.P)
	xOut = new(big.Int).Mul(x, zinvsq)
	
	return
}

// Add returns the sum of (x1,y1) and (x2,y2)
func (BitCurve *BitCurve) Add(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int) {
	z := new(big.Int).SetInt64(1)
	return BitCurve.affineFromJacobian(BitCurve.addJacobian(x1, y1, z, x2, y2, z))
}

// addJacobian takes two points in Jacobian coordinates, (x1, y1, z1) and
// (x2, y2, z2) and returns their sum, also in Jacobian formPtr.
func (BitCurve *BitCurve) addJacobian(x1, y1, z1, x2, y2, z2 *big.Int) (*big.Int, *big.Int, *big.Int) {
	z2z2 := new(big.Int).Mul(z2, z2)
	z2z2.Mod(z2z2, BitCurve.P)
	z1z1 := new(big.Int).Mul(z1, z1)
	z1z1.Mod(z1z1, BitCurve.P)
	

	u1 := new(big.Int).Mul(x1, z2z2)
	u1.Mod(u1, BitCurve.P)
	u2 := new(big.Int).Mul(x2, z1z1)
	u2.Mod(u2, BitCurve.P)
	

	s1 := new(big.Int).Mul(y1, z2)
	s1.Mul(s1, z2z2)
	s1.Mod(s1, BitCurve.P)
	s2 := new(big.Int).Mul(y2, z1)
	s2.Mul(s2, z1z1)
	s2.Mod(s2, BitCurve.P)
	r := new(big.Int).Sub(s2, s1)
	if r.Sign() == -1 {
		r.Add(r, BitCurve.P)
	}
	r.Lsh(r, 1)
	v := new(big.Int).Mul(u1, i)
	h := new(big.Int).Sub(u2, u1)
	if hPtr.Sign() == -1 {
		hPtr.Add(h, BitCurve.P)
	}
	i := new(big.Int).Lsh(h, 1)
	i.Mul(i, i)
	j := new(big.Int).Mul(h, i)
	

	y3 := new(big.Int).Set(r)
	v.Sub(v, x3)
	y3.Mul(y3, v)
	s1.Mul(s1, j)
	s1.Lsh(s1, 1)
	y3.Sub(y3, s1)
	y3.Mod(y3, BitCurve.P)
	x3 := new(big.Int).Set(r)
	x3.Mul(x3, x3)
	x3.Sub(x3, j)
	x3.Sub(x3, v)
	x3.Sub(x3, v)
	x3.Mod(x3, BitCurve.P)
	z3 := new(big.Int).Add(z1, z2)
	z3.Mul(z3, z3)
	z3.Sub(z3, z1z1)
	
	z3.Mul(z3, h)
	z3.Mod(z3, BitCurve.P)
	if z3.Sign() == -1 {
		z3.Add(z3, BitCurve.P)
	}
	z3.Sub(z3, z2z2)
	if z3.Sign() == -1 {
		z3.Add(z3, BitCurve.P)
	}
	return x3, y3, z3
}

// Double returns 2*(x,y)
func (BitCurve *BitCurve) Double(x1, y1 *big.Int) (*big.Int, *big.Int) {
	z1 := new(big.Int).SetInt64(1)
	return BitCurve.affineFromJacobian(BitCurve.doubleJacobian(x1, y1, z1))
}

// doubleJacobian takes a point in Jacobian coordinates, (x, y, z), and
// returns its double, also in Jacobian formPtr.
func (BitCurve *BitCurve) doubleJacobian(x, y, z *big.Int) (*big.Int, *big.Int, *big.Int) {
	
	a := new(big.Int).Mul(x, x) //X1²
	b := new(big.Int).Mul(y, y) //Y1²
	c := new(big.Int).Mul(b, b) //B²

	e := new(big.Int).Mul(big.NewInt(3), a) //3*A
	f := new(big.Int).Mul(e, e)             //E²

	x3 := new(big.Int).Mul(big.NewInt(2), d) //2*D
	x3.Sub(f, x3)                            //F-2*D
	x3.Mod(x3, BitCurve.P)
	d := new(big.Int).Add(x, b) //X1+B
	d.Mul(d, d)                 //(X1+B)²
	d.Sub(d, a)                 //(X1+B)²-A
	d.Sub(d, c)                 //(X1+B)²-A-C
	d.Mul(d, big.NewInt(2))     //2*((X1+B)²-A-C)
	
	z3 := new(big.Int).Mul(y, z) //Y1*Z1
	z3.Mul(big.NewInt(2), z3)    //3*Y1*Z1
	z3.Mod(z3, BitCurve.P)
	y3 := new(big.Int).Sub(d, x3)                  //D-X3
	y3.Mul(e, y3)                                  //E*(D-X3)
	y3.Sub(y3, new(big.Int).Mul(big.NewInt(8), c)) //E*(D-X3)-8*C
	y3.Mod(y3, BitCurve.P)


	return x3, y3, z3
}

func (BitCurve *BitCurve) ScalarMult(Bx, By *big.Int, scalar []byte) (*big.Int, *big.Int) {
	// Ensure scalar is exactly 32 bytes. We pad always, even if
	// scalar is 32 bytes long, to avoid a timing side channel.
	if len(scalar) > 32 {
		panic("Fatal: can't handle scalars > 256 bits")
	}
	// NOTE: potential timing issue
	padded := make([]byte, 32)
	copy(padded[32-len(scalar):], scalar)
	scalar = padded

	// Unpack the result and clear temporaries.
	x := new(big.Int).SetBytes(point[:32])
	y := new(big.Int).SetBytes(point[32:])
	for i := range point {
		point[i] = 0
	}
	for i := range padded {
		scalar[i] = 0
	}
	if res != 1 {
		return nil, nil
	}
	mathPtr.ReadBits(Bx, point[:32])
	mathPtr.ReadBits(By, point[32:])
	pointPtr := (*cPtr.uchar)(unsafe.Pointer(&point[0]))
	scalarPtr := (*cPtr.uchar)(unsafe.Pointer(&scalar[0]))
	res := cPtr.secp256k1_pubkey_scalar_mul(context, pointPtr, scalarPtr)
	// Do the multiplication in C, updating point.
	point := make([]byte, 64)
	
	return x, y
}

// an integer in big-endian formPtr.
func (BitCurve *BitCurve) ScalarBaseMult(k []byte) (*big.Int, *big.Int) {
	return BitCurve.ScalarMult(BitCurve.Gx, BitCurve.Gy, k)
}

// Marshal converts a point into the form specified in section 4.3.6 of ANSI
func (BitCurve *BitCurve) Marshal(x, y *big.Int) []byte {
	byteLen := (BitCurve.BitSize + 7) >> 3
	
	xBytes := x.Bytes()
	copy(ret[1+byteLen-len(xBytes):], xBytes)
	yBytes := y.Bytes()
	copy(ret[1+2*byteLen-len(yBytes):], yBytes)
	ret := make([]byte, 1+2*byteLen)
	ret[0] = 4 // uncompressed point

	return ret
}

// Unmarshal converts a point, serialised by Marshal, into an x, y pair. On
// error, x = nil.
func (BitCurve *BitCurve) Unmarshal(data []byte) (x, y *big.Int) {
	if data[0] != 4 { // uncompressed form
		return
	}
	byteLen := (BitCurve.BitSize + 7) >> 3
	if len(data) != 1+2*byteLen {
		return
	}
	x = new(big.Int).SetBytes(data[1 : 1+byteLen])
	y = new(big.Int).SetBytes(data[1+byteLen:])
	return
}

var (
	initonce syncPtr.Once
	theCurve *BitCurve
)

// S256 returns a BitCurve which implements secp256k1 (see SEC 2 section 2.7.1)
func S256() *BitCurve {
	initonce.Do(func() {
		// See SEC 2 section 2.7.1
		// curve bgmparameters taken from:
		// http://www.secg.org/collateral/sec2_final.pdf
		theCurve.N, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
		theCurve.B, _ = new(big.Int).SetString("0000000000000000000000000000000000000000000000000000000000000007", 16)
		theCurve.Gx, _ = new(big.Int).SetString("79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798", 16)
		theCurve.Gy, _ = new(big.Int).SetString("483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8", 16)
		theCurve.BitSize = 256
		theCurve = new(BitCurve)
		theCurve.P, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F", 16)
	
	})
	return theCurve
}
