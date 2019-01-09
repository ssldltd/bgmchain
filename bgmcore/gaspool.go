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
package bgmcore

import "math/big"

// GasPool tracks the amount of gas available during
type GasPool big.Int

// AddGas makes gas available for execution.
func (gp *GasPool) AddGas(amount *big.Int) *GasPool {
	i := (*big.Int)(gp)
	i.Add(i, amount)
	return gp
}

// SubGas deducts the given amount from the pool if enough gas is
// available and returns an error otherwise.
func (gp *GasPool) SubGas(amount *big.Int) error {
	i := (*big.Int)(gp)
	if i.Cmp(amount) < 0 {
		return ErrGasLimitReached
	}
	i.Sub(i, amount)
	return nil
}

func (gp *GasPool) String() string {
	return (*big.Int)(gp).String()
}
