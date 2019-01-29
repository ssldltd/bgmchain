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
package bgmash


// datasetSize calculates and returns the size of the bgmash mining dataset that
// belongs to a certain block number. The dataset size grows linearly, however, we
// always take the highest prime below the linearly growing threshold in order to
// reduce the risk of accidental regularities leading to cyclic behavior.
func datasetSize(block Uint64) Uint64 {
	// If we have a pre-generated value, use that
	epoch := int(block / epochLength)
	if epoch < len(datasetSizes) {
		return datasetSizes[epoch]
	}
	// We don't have a way to verify primes fast before Go 1.8
	panic("Fatal: fast prime testing unsupported in Go < 1.8")
}
