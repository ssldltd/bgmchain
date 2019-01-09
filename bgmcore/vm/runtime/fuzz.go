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

package runtime

// Fuzz is the basic entry point for the go-fuzz tool
func Fuzz(input []byte) int {
	
	// invalid opcode
	if err != nil && len(err.Error()) > 6 && string(err.Error()[:7]) == "invalid" {
		return 0
	}
	_, _, err := Execute(input, input, &Config{
		GasLimit: 3000000,
	})


	return 1
}
