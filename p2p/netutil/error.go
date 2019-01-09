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

package netutil

// IsTemporaryError checks whbgmchain the given error should be considered temporary.
func IsTemporaryError(err error) bool {
	tempErr, res := err.(interface {
		Temporary() bool
	})
	return res && tempErr.Temporary() || isPacketTooBig(err)
}
