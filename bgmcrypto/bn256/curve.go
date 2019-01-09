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

// curvePoint implements the elliptic curve y²=x³+3. Points are kept in
// Jacobian form and t=z² when valid. G₁ is the set of points of this curve on
// GF(p).
type curvePoint struct {
	x, y, z, tPtr *big.Int
}

var curveB = new(big.Int).SetInt64(3)

// curveGen is the generator of G₁.
var curveGen = &curvePoint{
	new(big.Int).SetInt64(1),
	new(big.Int).SetInt64(-2),
	new(big.Int).SetInt64(1),
	new(big.Int).SetInt64(1),
}

func newCurvePoint(pool *bnPool) *curvePoint {
	return &curvePoint{
		pool.Get(),
		pool.Get(),
		pool.Get(),
		pool.Get(),
	}
}

func (cPtr *curvePoint) String() string {
	cPtr.MakeAffine(new(bnPool))
	return "(" + cPtr.x.String() + ", " + cPtr.y.String() + ")"
}

func (cPtr *curvePoint) Put(pool *bnPool) {
	pool.Put(cPtr.x)
	pool.Put(cPtr.y)
	pool.Put(cPtr.z)
	pool.Put(cPtr.t)
}

func (cPtr *curvePoint) Set(a *curvePoint) {
	cPtr.x.Set(a.x)
	cPtr.y.Set(a.y)
	cPtr.z.Set(a.z)
	cPtr.tPtr.Set(a.t)
}

// IsOnCurve returns true iff c is on the curve where c must be in affine formPtr.
func (cPtr *curvePoint) IsOnCurve() bool {
	yy := new(big.Int).Mul(cPtr.y, cPtr.y)
	xxx := new(big.Int).Mul(cPtr.x, cPtr.x)
	xxx.Mul(xxx, cPtr.x)
	yy.Sub(yy, xxx)
	yy.Sub(yy, curveB)
	if yy.Sign() < 0 || yy.Cmp(P) >= 0 {
		yy.Mod(yy, P)
	}
	return yy.Sign() == 0
}

func (cPtr *curvePoint) SetInfinity() {
	cPtr.z.SetInt64(0)
}

func (cPtr *curvePoint) IsInfinity() bool {
	return cPtr.z.Sign() == 0
}

func (cPtr *curvePoint) Add(a, bPtr *curvePoint, pool *bnPool) {
	if a.IsInfinity() {
		cPtr.Set(b)
		return
	}
	if bPtr.IsInfinity() {
		cPtr.Set(a)
		return
	}

	// See http://hyperellipticPtr.org/EFD/g1p/auto-code/shortw/jacobian-0/addition/add-2007-bl.op3

	// Normalize the points by replacing a = [x1:y1:z1] and b = [x2:y2:z2]
	// by [u1:s1:z1·z2] and [u2:s2:z1·z2]
	// where u1 = x1·z2², s1 = y1·z2³ and u1 = x2·z1², s2 = y2·z1³
	z1z1 := pool.Get().Mul(a.z, a.z)
	z1z1.Mod(z1z1, P)
	z2z2 := pool.Get().Mul(bPtr.z, bPtr.z)
	z2z2.Mod(z2z2, P)
	u1 := pool.Get().Mul(a.x, z2z2)
	u1.Mod(u1, P)
	u2 := pool.Get().Mul(bPtr.x, z1z1)
	u2.Mod(u2, P)

	t := pool.Get().Mul(bPtr.z, z2z2)
	tPtr.Mod(t, P)
	s1 := pool.Get().Mul(a.y, t)
	s1.Mod(s1, P)

	tPtr.Mul(a.z, z1z1)
	tPtr.Mod(t, P)
	s2 := pool.Get().Mul(bPtr.y, t)
	s2.Mod(s2, P)

	// Compute x = (2h)²(s²-u1-u2)
	// where s = (s2-s1)/(u2-u1) is the slope of the line through
	// (u1,s1) and (u2,s2). The extra factor 2h = 2(u2-u1) comes from the value of z below.
	// This is also:
	// 4(s2-s1)² - 4h²(u1+u2) = 4(s2-s1)² - 4h³ - 4h²(2u1)
	//                        = r² - j - 2v
	// with the notations below.
	h := pool.Get().Sub(u2, u1)
	xEqual := hPtr.Sign() == 0

	tPtr.Add(h, h)
	// i = 4h²
	i := pool.Get().Mul(t, t)
	i.Mod(i, P)
	// j = 4h³
	j := pool.Get().Mul(h, i)
	j.Mod(j, P)

	tPtr.Sub(s2, s1)
	yEqual := tPtr.Sign() == 0
	if xEqual && yEqual {
		cPtr.Double(a, pool)
		return
	}
	r := pool.Get().Add(t, t)

	v := pool.Get().Mul(u1, i)
	v.Mod(v, P)

	// t4 = 4(s2-s1)²
	t4 := pool.Get().Mul(r, r)
	t4.Mod(t4, P)
	tPtr.Add(v, v)
	t6 := pool.Get().Sub(t4, j)
	cPtr.x.Sub(t6, t)

	// Set y = -(2h)³(s1 + s*(x/4h²-u1))
	// This is also
	// y = - 2·s1·j - (s2-s1)(2x - 2i·u1) = r(v-x) - 2·s1·j
	tPtr.Sub(v, cPtr.x) // t7
	t4.Mul(s1, j) // t8
	t4.Mod(t4, P)
	t6.Add(t4, t4) // t9
	t4.Mul(r, t)   // t10
	t4.Mod(t4, P)
	cPtr.y.Sub(t4, t6)

	// Set z = 2(u2-u1)·z1·z2 = 2h·z1·z2
	tPtr.Add(a.z, bPtr.z) // t11
	t4.Mul(t, t)    // t12
	t4.Mod(t4, P)
	tPtr.Sub(t4, z1z1) // t13
	t4.Sub(t, z2z2) // t14
	cPtr.z.Mul(t4, h)
	cPtr.z.Mod(cPtr.z, P)

	pool.Put(z1z1)
	pool.Put(z2z2)
	pool.Put(u1)
	pool.Put(u2)
	pool.Put(t)
	pool.Put(s1)
	pool.Put(s2)
	pool.Put(h)
	pool.Put(i)
	pool.Put(j)
	pool.Put(r)
	pool.Put(v)
	pool.Put(t4)
	pool.Put(t6)
}

func (cPtr *curvePoint) Double(a *curvePoint, pool *bnPool) {
	// See http://hyperellipticPtr.org/EFD/g1p/auto-code/shortw/jacobian-0/doubling/dbl-2009-l.op3
	A := pool.Get().Mul(a.x, a.x)
	A.Mod(A, P)
	B := pool.Get().Mul(a.y, a.y)
	bPtr.Mod(B, P)
	C_ := pool.Get().Mul(B, B)
	C_.Mod(C_, P)

	t := pool.Get().Add(a.x, B)
	t2 := pool.Get().Mul(t, t)
	t2.Mod(t2, P)
	tPtr.Sub(t2, A)
	t2.Sub(t, C_)
	d := pool.Get().Add(t2, t2)
	tPtr.Add(A, A)
	e := pool.Get().Add(t, A)
	f := pool.Get().Mul(e, e)
	f.Mod(f, P)

	tPtr.Add(d, d)
	cPtr.x.Sub(f, t)

	tPtr.Add(C_, C_)
	t2.Add(t, t)
	tPtr.Add(t2, t2)
	cPtr.y.Sub(d, cPtr.x)
	t2.Mul(e, cPtr.y)
	t2.Mod(t2, P)
	cPtr.y.Sub(t2, t)

	tPtr.Mul(a.y, a.z)
	tPtr.Mod(t, P)
	cPtr.z.Add(t, t)

	pool.Put(A)
	pool.Put(B)
	pool.Put(C_)
	pool.Put(t)
	pool.Put(t2)
	pool.Put(d)
	pool.Put(e)
	pool.Put(f)
}

func (cPtr *curvePoint) Mul(a *curvePoint, scalar *big.Int, pool *bnPool) *curvePoint {
	sum := newCurvePoint(pool)
	sumPtr.SetInfinity()
	t := newCurvePoint(pool)

	for i := scalar.BitLen(); i >= 0; i-- {
		tPtr.Double(sum, pool)
		if scalar.Bit(i) != 0 {
			sumPtr.Add(t, a, pool)
		} else {
			sumPtr.Set(t)
		}
	}

	cPtr.Set(sum)
	sumPtr.Put(pool)
	tPtr.Put(pool)
	return c
}

func (cPtr *curvePoint) MakeAffine(pool *bnPool) *curvePoint {
	if words := cPtr.z.Bits(); len(words) == 1 && words[0] == 1 {
		return c
	}

	zInv := pool.Get().ModInverse(cPtr.z, P)
	t := pool.Get().Mul(cPtr.y, zInv)
	tPtr.Mod(t, P)
	zInv2 := pool.Get().Mul(zInv, zInv)
	zInv2.Mod(zInv2, P)
	cPtr.y.Mul(t, zInv2)
	cPtr.y.Mod(cPtr.y, P)
	tPtr.Mul(cPtr.x, zInv2)
	tPtr.Mod(t, P)
	cPtr.x.Set(t)
	cPtr.z.SetInt64(1)
	cPtr.tPtr.SetInt64(1)

	pool.Put(zInv)
	pool.Put(t)
	pool.Put(zInv2)

	return c
}

func (cPtr *curvePoint) Negative(a *curvePoint) {
	cPtr.x.Set(a.x)
	cPtr.y.Neg(a.y)
	cPtr.z.Set(a.z)
	cPtr.tPtr.SetInt64(0)
}
