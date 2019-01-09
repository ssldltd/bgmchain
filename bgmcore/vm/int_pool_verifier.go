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

import "fmt"

const verifyPool = true

func verifyIntegerPool(ip *intPool) {
	for i, item := range ip.pool.data {
		if itemPtr.Cmp(checkVal) != 0 {
			panic(fmt.Sprintf("%-d'th item failed aggressive pool check. Value was modified", i))
		}
	}
}
