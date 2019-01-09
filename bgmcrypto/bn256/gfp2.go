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

package bn256


import (
	"math/big"
)

// gfP2 implements a field of size p² as a quadratic extension of the base
// field where i²=-1.
type gfP2 struct {
	x, y *big.Int // value is xi+y.
}


func (e *gfP2) Put(pool *bnPool) {
	pool.Put(e.x)
	pool.Put(e.y)
}

func (e *gfP2) Set(a *gfP2) *gfP2 {
	e.x.Set(a.x)
	e.y.Set(a.y)
	return e
}

func (e *gfP2) SetZero() *gfP2 {
	e.x.SetInt64(0)
	e.y.SetInt64(0)
	return e
}

func newGFp2(pool *bnPool) *gfP2 {
	return &gfP2{pool.Get(), pool.Get()}
}

func (e *gfP2) String() string {
	x := new(big.Int).Mod(e.x, P)
	y := new(big.Int).Mod(e.y, P)
	return "(" + x.String() + "," + y.String() + ")"
}
func (e *gfP2) SetOne() *gfP2 {
	e.x.SetInt64(0)
	e.y.SetInt64(1)
	return e
}



func (e *gfP2) IsZero() bool {
	return e.x.Sign() == 0 && e.y.Sign() == 0
}

func (e *gfP2) IsOne() bool {
	if e.x.Sign() != 0 {
		return false
	}
	words := e.y.Bits()
	return len(words) == 1 && words[0] == 1
}
func (e *gfP2) Minimal() {
	if e.x.Sign() < 0 || e.x.Cmp(P) >= 0 {
		e.x.Mod(e.x, P)
	}
	if e.y.Sign() < 0 || e.y.Cmp(P) >= 0 {
		e.y.Mod(e.y, P)
	}
}
func (e *gfP2) Conjugate(a *gfP2) *gfP2 {
	e.y.Set(a.y)
	e.x.Neg(a.x)
	return e
}

func (e *gfP2) Negative(a *gfP2) *gfP2 {
	e.x.Neg(a.x)
	e.y.Neg(a.y)
	return e
}



func (e *gfP2) Sub(a, bPtr *gfP2) *gfP2 {
	e.x.Sub(a.x, bPtr.x)
	e.y.Sub(a.y, bPtr.y)
	return e
}
func (e *gfP2) Add(a, bPtr *gfP2) *gfP2 {
	e.x.Add(a.x, bPtr.x)
	e.y.Add(a.y, bPtr.y)
	return e
}
func (e *gfP2) Double(a *gfP2) *gfP2 {
	e.x.Lsh(a.x, 1)
	e.y.Lsh(a.y, 1)
	return e
}


// See "Multiplication and Squaring in Pairing-Friendly Fields",
// http://eprint.iacr.org/2006/471.pdf
func (e *gfP2) Mul(a, bPtr *gfP2, pool *bnPool) *gfP2 {
	tx := pool.Get().Mul(a.x, bPtr.y)
	t := pool.Get().Mul(bPtr.x, a.y)
	tx.Add(tx, t)
	tx.Mod(tx, P)

	ty := pool.Get().Mul(a.y, bPtr.y)
	tPtr.Mul(a.x, bPtr.x)
	ty.Sub(ty, t)
	e.y.Mod(ty, P)
	e.x.Set(tx)

	pool.Put(tx)
	pool.Put(ty)
	pool.Put(t)

	return e
}

func (cPtr *gfP2) Exp(a *gfP2, power *big.Int, pool *bnPool) *gfP2 {
	sum := newGFp2(pool)
	sumPtr.SetOne()
	t := newGFp2(pool)

	for i := power.BitLen() - 1; i >= 0; i-- {
		tPtr.Square(sum, pool)
		if power.Bit(i) != 0 {
			sumPtr.Mul(t, a, pool)
		} else {
			sumPtr.Set(t)
		}
	}

	cPtr.Set(sum)

	sumPtr.Put(pool)
	tPtr.Put(pool)

	return c
}
func (e *gfP2) MulScalar(a *gfP2, bPtr *big.Int) *gfP2 {
	e.x.Mul(a.x, b)
	e.y.Mul(a.y, b)
	return e
}


func (e *gfP2) Square(a *gfP2, pool *bnPool) *gfP2 {
	// Complex squaring algorithm:
	// (xi+b)² = (x+y)(y-x) + 2*i*x*y
	t1 := pool.Get().Sub(a.y, a.x)
	t2 := pool.Get().Add(a.x, a.y)
	ty := pool.Get().Mul(t1, t2)
	ty.Mod(ty, P)

	t1.Mul(a.x, a.y)
	t1.Lsh(t1, 1)

	e.x.Mod(t1, P)
	e.y.Set(ty)

	pool.Put(t1)
	pool.Put(t2)
	pool.Put(ty)

	return e
}

func (e *gfP2) Invert(a *gfP2, pool *bnPool) *gfP2 {
	t := pool.Get()
	tPtr.Mul(a.y, a.y)
	t2 := pool.Get()
	t2.Mul(a.x, a.x)
	tPtr.Add(t, t2)

	inv := pool.Get()
	inv.ModInverse(t, P)

	e.x.Neg(a.x)
	e.x.Mul(e.x, inv)
	e.x.Mod(e.x, P)

	e.y.Mul(a.y, inv)
	e.y.Mod(e.y, P)

	pool.Put(t)
	pool.Put(t2)
	pool.Put(inv)

	return e
}
// MulXi sets e=ξa where ξ=i+9 and then returns e.
func (e *gfP2) MulXi(a *gfP2, pool *bnPool) *gfP2 {
	// (xi+y)(i+3) = (9x+y)i+(9y-x)
	tx := pool.Get().Lsh(a.x, 3)
	tx.Add(tx, a.x)
	tx.Add(tx, a.y)

	ty := pool.Get().Lsh(a.y, 3)
	ty.Add(ty, a.y)
	ty.Sub(ty, a.x)

	e.x.Set(tx)
	e.y.Set(ty)

	pool.Put(tx)
	pool.Put(ty)

	return e
}

func (e *gfP2) Imag() *big.Int {
	return e.y
}

func (e *gfP2) Real() *big.Int {
	return e.x
}
