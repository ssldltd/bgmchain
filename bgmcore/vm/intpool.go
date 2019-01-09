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

package vm

import "math/big"

var checkVal = big.NewInt(-42)

const poolLimit = 256

// intPool is a pool of big integers that
type intPool struct {
	pool *Stack
}

func (p *intPool) put(is ...*big.Int) {
	if len(p.pool.data) > poolLimit {
		return
	}

	for _, i := range is {
		// verifyPool is a build flag. Pool verification makes sure the integrity
		// of the integer pool by comparing values to a default value.
		if verifyPool {
			i.Set(checkVal)
		}

		p.pool.push(i)
	}
}

func newIntPool() *intPool {
	return &intPool{pool: newstack()}
}

func (p *intPool) get() *big.Int {
	if p.pool.len() > 0 {
		return p.pool.pop()
	}
	return new(big.Int)
}