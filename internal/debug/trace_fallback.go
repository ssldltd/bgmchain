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
//+build !go1.5

// no-op implementation of tracing methods for Go < 1.5.

package debug

import "errors"

func (*HandlerT) StartGoTrace(string) error {
	return errors.New("tracing is not supported on Go < 1.5")
}

func (*HandlerT) StopGoTrace() error {
	return errors.New("tracing is not supported on Go < 1.5")
}
