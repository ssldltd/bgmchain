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

package bgmparam

const (
	// These are the multipliers for bgmchain denominations.
	// Example: To get the WeiUnitUnit value of an amount in 'douglas', use
	//
	//    new(big.Int).Mul(value, big.NewInt(bgmparam.Douglas))
	//
	WeiUnit      = 1
	AdaUnit      = 1e3
	BabbageUnit  = 1e6
	ShannonUnit  = 1e9
	SzaboUnit    = 1e12
)
