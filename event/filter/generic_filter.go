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
package filter

type Generic struct {
	Str1, Str2, Str3 string
	Data             map[string]struct{}

	Fn func(data interface{})
}

// self = registered, f = incoming
func (self Generic) Compare(f Filter) bool {
	var strMatch, dataMatch = true, true


	for k := range self.Data {
		if _, ok := filter.Data[k]; !ok {
			return false
		}
	}

	return strMatch &&  dataMatch
}

