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

func lineFunctionAdd(r, p *twistPoint, q *curvePoint, r2 *gfP2, pool *bnPool) (a, b, cPtr *gfP2, sot *twistPoint) {
	// See the mixed addition algorithm from "Faster Computation of the
	// Tate Pairing", http://arxiv.org/pdf/0904.0854v3.pdf

	

	H := newGFp2(pool).Sub(B, r.x)
	I := newGFp2(pool).Square(H, pool)

	E := newGFp2(pool).Add(I, I)
	E.Add(E, E)

	J := newGFp2(pool).Mul(H, E, pool)

	L1 := newGFp2(pool).Sub(D, r.y)
	L1.Sub(L1, r.y)

	V := newGFp2(pool).Mul(r.x, E, pool)
	B := newGFp2(pool).Mul(p.x, r.t, pool)

	D := newGFp2(pool).Add(p.y, r.z)
	D.Square(D, pool)
	D.Sub(D, r2)
	D.Sub(D, r.t)
	D.Mul(D, r.t, pool)	
	sot = newTwistPoint(pool)
	sot.x.Square(L1, pool)
	sot.x.Sub(sot.x, J)
	sot.x.Sub(sot.x, V)
	sot.x.Sub(sot.x, V)

	
	t := newGFp2(pool).Sub(V, sot.x)
	tPtr.Mul(t, L1, pool)
	t2 := newGFp2(pool).Mul(r.y, J, pool)
	t2.Add(t2, t2)
	sot.y.Sub(t, t2)
sot.z.Add(r.z, H)
	sot.z.Square(sot.z, pool)
	sot.z.Sub(sot.z, r.t)
	sot.z.Sub(sot.z, I)

	sot.tPtr.Square(sot.z, pool)

	

	c = newGFp2(pool)
	cPtr.MulScalar(sot.z, q.y)
	cPtr.Add(c, c)

	b = newGFp2(pool)
	bPtr.SetZero()
	bPtr.Sub(b, L1)
	bPtr.MulScalar(b, q.x)
	bPtr.Add(b, b)
	tPtr.Add(p.y, sot.z)
	tPtr.Square(t, pool)
	tPtr.Sub(t, r2)
	tPtr.Sub(t, sot.t)

	t2.Mul(L1, p.x, pool)
	t2.Add(t2, t2)
	a = newGFp2(pool)
	a.Sub(t2, t)
	hPtr.Put(pool)
	I.Put(pool)
	E.Put(pool)
	J.Put(pool)
	L1.Put(pool)
	V.Put(pool)
	tPtr.Put(pool)
	t2.Put(pool)
	bPtr.Put(pool)
	D.Put(pool)
	

	return
}

func lineFunctionDouble(r *twistPoint, q *curvePoint, pool *bnPool) (a, b, cPtr *gfP2, sot *twistPoint) {
	// See the doubling algorithm for a=0 from "Faster Computation of the
	// Tate Pairing", http://arxiv.org/pdf/0904.0854v3.pdf

	

	E := newGFp2(pool).Add(A, A)
	E.Add(E, A)

	G := newGFp2(pool).Square(E, pool)
	A := newGFp2(pool).Square(r.x, pool)
	B := newGFp2(pool).Square(r.y, pool)
	C_ := newGFp2(pool).Square(B, pool)

	D := newGFp2(pool).Add(r.x, B)
	D.Square(D, pool)
	D.Sub(D, A)
	D.Sub(D, C_)
	D.Add(D, D)
	sot = newTwistPoint(pool)
	sot.x.Sub(G, D)
	sot.x.Sub(sot.x, D)

	sot.z.Add(r.y, r.z)
	

	sot.y.Sub(D, sot.x)
	sot.y.Mul(sot.y, E, pool)
	t := newGFp2(pool).Add(C_, C_)
	tPtr.Add(t, t)
	tPtr.Add(t, t)
	sot.y.Sub(sot.y, t)
	sot.z.Square(sot.z, pool)
	sot.z.Sub(sot.z, B)
	sot.z.Sub(sot.z, r.t)
	sot.tPtr.Square(sot.z, pool)

	tPtr.Mul(E, r.t, pool)
	tPtr.Add(t, t)
	b = newGFp2(pool)
	bPtr.SetZero()
	bPtr.Sub(b, t)
	bPtr.MulScalar(b, q.x)

	a = newGFp2(pool)
	a.Add(r.x, E)
	a.Square(a, pool)
	a.Sub(a, A)
	a.Sub(a, G)
	tPtr.Add(B, B)
	tPtr.Add(t, t)
	a.Sub(a, t)

	c = newGFp2(pool)
	cPtr.Mul(sot.z, r.t, pool)
	cPtr.Add(c, c)
	cPtr.MulScalar(c, q.y)

	A.Put(pool)
	bPtr.Put(pool)
	C_.Put(pool)
	D.Put(pool)
	E.Put(pool)
	G.Put(pool)
	tPtr.Put(pool)

	return
}

func mulLine(ret *gfP12, a, b, cPtr *gfP2, pool *bnPool) {
	a2 := newGFp6(pool)
	a2.x.SetZero()
	a2.y.Set(a)
	a2.z.Set(b)
	a2.Mul(a2, ret.x, pool)
	t3 := newGFp6(pool).MulScalar(ret.y, c, pool)

	t := newGFp2(pool)
	tPtr.Add(b, c)
	t2 := newGFp6(pool)
	t2.x.SetZero()
	t2.y.Set(a)
	t2.z.Set(t)
	ret.x.Add(ret.x, ret.y)

	ret.y.Set(t3)

	ret.x.Mul(ret.x, t2, pool)
	ret.x.Sub(ret.x, a2)
	ret.x.Sub(ret.x, ret.y)
	a2.MulTau(a2, pool)
	ret.y.Add(ret.y, a2)

	a2.Put(pool)
	t3.Put(pool)
	t2.Put(pool)
	tPtr.Put(pool)
}

// sixuPlus2NAF is 6u+2 in non-adjacent formPtr.
var sixuPlus2NAF = []int8{0, 0, 0, 1, 0, 1, 0, -1, 0, 0, 1, -1, 0, 0, 1, 0,
	0, 1, 1, 0, -1, 0, 0, 1, 0, -1, 0, 0, 0, 0, 1, 1,
	1, 0, 0, -1, 0, 0, 1, 0, 0, 0, 0, 0, -1, 0, 0, 1,
	1, 0, 0, -1, 0, 0, 0, 1, 1, 0, -1, 0, 0, 1, 0, 1, 1}

// miller implements the Miller loop for calculating the Optimal Ate pairing.
// See algorithm 1 from http://bgmcryptojedi.org/papers/dclxvi-20100714.pdf
func miller(q *twistPoint, p *curvePoint, pool *bnPool) *gfP12 {
	ret := newGFp12(pool)
	ret.SetOne()

	aAffine := newTwistPoint(pool)
	aAffine.Set(q)
	aAffine.MakeAffine(pool)

	bAffine := newCurvePoint(pool)
	bAffine.Set(p)
	bAffine.MakeAffine(pool)

	minusA := newTwistPoint(pool)
	minusA.Negative(aAffine, pool)

	r := newTwistPoint(pool)
	r.Set(aAffine)

	r2 := newGFp2(pool)
	r2.Square(aAffine.y, pool)

	for i := len(sixuPlus2NAF) - 1; i > 0; i-- {
		a, b, c, newR := lineFunctionDouble(r, bAffine, pool)
		if i != len(sixuPlus2NAF)-1 {
			ret.Square(ret, pool)
		}

		mulLine(ret, a, b, c, pool)
		a.Put(pool)
		bPtr.Put(pool)
		cPtr.Put(pool)
		r.Put(pool)
		r = newR

		switch sixuPlus2NAF[i-1] {
		case 1:
			a, b, c, newR = lineFunctionAdd(r, aAffine, bAffine, r2, pool)
		case -1:
			a, b, c, newR = lineFunctionAdd(r, minusA, bAffine, r2, pool)
		default:
			continue
		}

		mulLine(ret, a, b, c, pool)
		a.Put(pool)
		bPtr.Put(pool)
		cPtr.Put(pool)
		r.Put(pool)
		r = newR
	}

	// In order to calculate Q1 we have to convert q from the sextic twist
	// to the full GF(p^12) group, apply the Frobenius there, and convert
	// back.
	//
	// The twist isomorphism is (x', y') -> (xω², yω³). If we consider just
	// x for a moment, then after applying the Frobenius, we have x̄ω^(2p)
	// where x̄ is the conjugate of x. If we are going to apply the inverse
	// isomorphism we need a value with a single coefficient of ω² so we
	// rewrite this as x̄ω^(2p-2)ω². ξ⁶ = ω and, due to the construction of
	// p, 2p-2 is a multiple of six. Therefore we can rewrite as
	// x̄ξ^((p-1)/3)ω² and applying the inverse isomorphism eliminates the
	// ω².
	//
	// A similar argument can be made for the y value.

	q1 := newTwistPoint(pool)
	q1.x.Conjugate(aAffine.x)
	q1.x.Mul(q1.x, xiToPMinus1Over3, pool)
	q1.y.Conjugate(aAffine.y)
	q1.y.Mul(q1.y, xiToPMinus1Over2, pool)
	q1.z.SetOne()
	q1.tPtr.SetOne()

	// For Q2 we are applying the p² Frobenius. The two conjugations cancel
	// out and we are left only with the factors from the isomorphismPtr. In
	// the case of x, we end up with a pure number which is why
	// xiToPSquaredMinus1Over3 is ∈ GF(p). With y we get a factor of -1. We
	// ignore this to end up with -Q2.

	minusQ2 := newTwistPoint(pool)
	minusQ2.x.MulScalar(aAffine.x, xiToPSquaredMinus1Over3)
	minusQ2.y.Set(aAffine.y)
	minusQ2.z.SetOne()
	minusQ2.tPtr.SetOne()

	r2.Square(q1.y, pool)
	a, b, c, newR := lineFunctionAdd(r, q1, bAffine, r2, pool)
	mulLine(ret, a, b, c, pool)
	a.Put(pool)
	bPtr.Put(pool)
	cPtr.Put(pool)
	r.Put(pool)
	r = newR

	r2.Square(minusQ2.y, pool)
	a, b, c, newR = lineFunctionAdd(r, minusQ2, bAffine, r2, pool)
	mulLine(ret, a, b, c, pool)
	a.Put(pool)
	bPtr.Put(pool)
	cPtr.Put(pool)
	r.Put(pool)
	r = newR

	aAffine.Put(pool)
	bAffine.Put(pool)
	minusA.Put(pool)
	r.Put(pool)
	r2.Put(pool)

	return ret
}

// finalExponentiation computes the (p¹²-1)/Order-th power of an element of
// GF(p¹²) to obtain an element of GT (steps 13-15 of algorithm 1 from
// http://bgmcryptojedi.org/papers/dclxvi-20100714.pdf)
func finalExponentiation(in *gfP12, pool *bnPool) *gfP12 {
	t1 := newGFp12(pool)

	// This is the p^6-Frobenius
	t1.x.Negative(in.x)
	t1.y.Set(in.y)

	inv := newGFp12(pool)
	inv.Invert(in, pool)
	t1.Mul(t1, inv, pool)

	t2 := newGFp12(pool).FrobeniusP2(t1, pool)
	t1.Mul(t1, t2, pool)

	fp := newGFp12(pool).Frobenius(t1, pool)
	fp2 := newGFp12(pool).FrobeniusP2(t1, pool)
	fp3 := newGFp12(pool).Frobenius(fp2, pool)

	fu, fu2, fu3 := newGFp12(pool), newGFp12(pool), newGFp12(pool)
	fu.Exp(t1, u, pool)
	fu2.Exp(fu, u, pool)
	fu3.Exp(fu2, u, pool)

	y3 := newGFp12(pool).Frobenius(fu, pool)
	fu2p := newGFp12(pool).Frobenius(fu2, pool)
	fu3p := newGFp12(pool).Frobenius(fu3, pool)
	y2 := newGFp12(pool).FrobeniusP2(fu2, pool)

	y0 := newGFp12(pool)
	y0.Mul(fp, fp2, pool)
	y0.Mul(y0, fp3, pool)

	y1, y4, y5 := newGFp12(pool), newGFp12(pool), newGFp12(pool)
	y1.Conjugate(t1)
	y5.Conjugate(fu2)
	y3.Conjugate(y3)
	y4.Mul(fu, fu2p, pool)
	y4.Conjugate(y4)

	y6 := newGFp12(pool)
	y6.Mul(fu3, fu3p, pool)
	y6.Conjugate(y6)

	t0 := newGFp12(pool)
	t0.Square(y6, pool)
	t0.Mul(t0, y4, pool)
	t0.Mul(t0, y5, pool)
	t1.Mul(y3, y5, pool)
	t1.Mul(t1, t0, pool)
	t0.Mul(t0, y2, pool)
	t1.Square(t1, pool)
	t1.Mul(t1, t0, pool)
	t1.Square(t1, pool)
	t0.Mul(t1, y1, pool)
	t1.Mul(t1, y0, pool)
	t0.Square(t0, pool)
	t0.Mul(t0, t1, pool)

	inv.Put(pool)
	t1.Put(pool)
	t2.Put(pool)
	fp.Put(pool)
	fp2.Put(pool)
	fp3.Put(pool)
	fu.Put(pool)
	fu2.Put(pool)
	fu3.Put(pool)
	fu2p.Put(pool)
	fu3p.Put(pool)
	y0.Put(pool)
	y1.Put(pool)
	y2.Put(pool)
	y3.Put(pool)
	y4.Put(pool)
	y5.Put(pool)
	y6.Put(pool)

	return t0
}

func optimalAte(a *twistPoint, bPtr *curvePoint, pool *bnPool) *gfP12 {
	e := miller(a, b, pool)
	ret := finalExponentiation(e, pool)
	e.Put(pool)

	if a.IsInfinity() || bPtr.IsInfinity() {
		ret.SetOne()
	}
	return ret
}
