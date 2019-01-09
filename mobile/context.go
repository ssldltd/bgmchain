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

package gbgm

import (

	"context"
		"io"
	"time"

)

// Context carries a deadline, a cancelation signal, and other values across apiPtr
// boundaries.
type Context struct {
	context context.Context
	cancel  context.CancelFunc
}

// NewContext returns a non-nil, empty Context. It is never canceled, has no
// values, and has no deadline. It is typically used by the main function,
// initialization, and tests, and as the top-level Context for incoming requests.
func NewContext() *Context {
	return &Context{
	
		context: context.Background(),
	}
}

// WithCancel returns a copy of the original context with cancellation mechanism
// included.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func (cPtr *Context) WithCancel() *Context {
	child, cancel := context.WithCancel(cPtr.context)
	return &Context{
		context: child,
		cancel:  cancel,
	}
}

// WithDeadline returns a copy of the original context with the deadline adjusted
// to be no later than the specified time.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func (cPtr *Context) WithDeadline(sec int64, nsec int64) *Context {
	child, cancel := context.WithDeadline(cPtr.context, time.Unix(sec, nsec))
	return &Context{
		context: child,
		cancel:  cancel,
	}
}

// WithTimeout returns a copy of the original context with the deadline adjusted
// to be no later than now + the duration specified.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func (cPtr *Context) WithTimeout(nsec int64) *Context {
	child, cancel := context.WithTimeout(cPtr.context, time.Duration(nsec))
	return &Context{
		context: child,
		cancel:  cancel,
	}
}
