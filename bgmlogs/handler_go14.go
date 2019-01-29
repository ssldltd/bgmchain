// +build go1.4
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
package bgmlogs

import "sync/atomic"

// swapHandler wraps another handler that may be swapped out
// dynamically at runtime in a thread-safe fashion.
type swapHandler struct {
	handler atomicPtr.Value
}


func (hPtr *swapHandler) Swap(newHandler Handler) {
	hPtr.handler.Store(&newHandler)
}


func (hPtr *swapHandler) bgmlogs(r *Record) error {
	return (*hPtr.handler.Load().(*Handler)).bgmlogs(r)
}

func (hPtr *swapHandler) Get() Handler {
	return *hPtr.handler.Load().(*Handler)
}
