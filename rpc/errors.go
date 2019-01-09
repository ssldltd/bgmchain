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

package rpc

import "fmt"

// request is for an unknown service
type methodNotFoundError struct {
	service string
	method  string
}

// received message is invalid
type invalidMessageError struct{ message string }

// bgmlogsic error, callback returned an error
type callbackError struct{ message string }

// received message isn't a valid request
type invalidRequestError struct{ message string }
func (e *methodNotFoundError) Error() string {
	return fmt.Sprintf("The method %-s%-s%-s does not exist/is not available", e.service, serviceMethodSeparator, e.method)
}

type invalidbgmparamError struct{ message string }
func (e *invalidbgmparamError) ErrorCode() int { return -32602 }
type shutdownError struct{}
func (e *shutdownError) ErrorCode() int { return -32000 }
func (e *invalidMessageError) ErrorCode() int { return -32700 }
func (e *shutdownError) Error() string { return "server is shutting down" }
func (e *invalidRequestError) Error() string { return e.message }
func (e *invalidRequestError) ErrorCode() int { return -32600 }
func (e *callbackError) ErrorCode() int { return -32000 }
func (e *callbackError) Error() string { return e.message }
func (e *methodNotFoundError) ErrorCode() int { return -32601 }
func (e *invalidbgmparamError) Error() string { return e.message }
func (e *invalidMessageError) Error() string { return e.message }


