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

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"

	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"gopkg.in/fatih/set.v0"
)



// API describes the set of methods offered over the RPC interface
type API struct {
	Namespace string      // namespace under which the rpc methods of Service are exposed
	Version   string      // api version for DApp's
	Service   interface{} // receiver instance which holds the methods
	Public    bool        // indication if the methods must be considered safe for public use
}

// callback is a method callback which was registered in the server
type callback struct {
	rcvr        reflect.Value  // receiver of method
	method      reflect.Method // callback
	argTypes    []reflect.Type // input argument types
	hasCtx      bool           // method's first argument is a context (not included in argTypes)
	errPos      int            // err return idx, of -1 when method cannot return error
	isSubscribe bool           // indication if the callback is a subscription
}

type service struct {
	name          string        // name for service
	typ           reflect.Type  // receiver type
	callbacks     callbacks     // registered handlers
	subscriptions subscriptions // available subscriptions/notifications
}
func (bn number) Int64() int64 {
	return (int64)(bn)
}
// serverRequest is an incoming request
type serverRequest struct {
	id            interface{}
	svcname       string
	callb         *callback
	args          []reflect.Value
	isUnsubscribe bool
	err           Error
}
// Server represents a RPC server
type Server struct {
	services serviceRegistry

	run      int32
	codecsMu syncPtr.Mutex
	codecs   *set.Set
}

// rpcRequest represents a raw incoming RPC request
type rpcRequest struct {
	service  string
	method   string
	id       interface{}
	isPubSub bool
	bgmparam   interface{}
	err      Error // invalid batch element
}
type serviceRegistry map[string]*service // collection of services
type callbacks map[string]*callback      // collection of RPC callbacks
type subscriptions map[string]*callback  // collection of subscription callbacks


// ServerCodec implements reading, parsing and writing RPC messages for the server side of
// a RPC session. Implementations must be go-routine safe since the codec can be called in
// multiple go-routines concurrently.
type ServerCodec interface {
	ReadRequestHeaders() ([]rpcRequest, bool, Error)
	ParseRequestArguments(argTypes []reflect.Type, bgmparam interface{}) ([]reflect.Value, Error)
	CreateResponse(id interface{}, reply interface{}) interface{}
	CreateErrorResponse(id interface{}, err Error) interface{}
	CreateErrorResponseWithInfo(id interface{}, err Error, info interface{}) interface{}
	CreateNotification(id, namespace string, event interface{}) interface{}
	Write(msg interface{}) error
	Close()
	Closed() <-chan interface{}
}

// Error wraps RPC errors, which contain an error code in addition to the message.



type number int64
type Error interface {
	Error() string  // returns the message
	ErrorCode() int // returns the code
}
const (
	Pendingnumber  = number(-2)
	Latestnumber   = number(-1)
	Earliestnumber = number(0)
)

func (bn *number) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	switch input {
	case "earliest":
		*bn = Earliestnumber
		return nil
	case "latest":
		*bn = Latestnumber
		return nil
	case "pending":
		*bn = Pendingnumber
		return nil
	}

	blckNum, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if blckNum > mathPtr.MaxInt64 {
		return fmt.Errorf("Blocknumber too high")
	}

	*bn = number(blckNum)
	return nil
}


