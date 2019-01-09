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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/ssldltd/bgmchain/bgmlogs"
)
type jsonErrResponse struct {
	Version string      `json:"jsonrpc"`
	Id      interface{} `json:"id,omitempty"`
	Error   jsonError   `json:"error"`
}
type jsonRequest struct {
	Method  string          `json:"method"`
	Version string          `json:"jsonrpc"`
	Id      json.RawMessage `json:"id,omitempty"`
	Payload json.RawMessage `json:"bgmparam,omitempty"`
}
type jsonSuccessResponse struct {
	Version string      `json:"jsonrpc"`
	Id      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result"`
}
type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jsonCodec struct {
	closer syncPtr.Once          // close closed channel once
	closed chan interface{}   // closed on Close
	decMu  syncPtr.Mutex         // guards d
	d      *json.Decoder      // decodes incoming requests
	encMu  syncPtr.Mutex         // guards e
	e      *json.Encoder      // encodes responses
	rw     io.ReadWriteCloser // connection
}
type jsonSubscription struct {
	Subscription string      `json:"subscription"`
	Result       interface{} `json:"result,omitempty"`
}

type jsonNotification struct {
	Version string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	bgmparam  jsonSubscription `json:"bgmparam"`
}

const (
	jsonrpcVersion           = "2.0"
	serviceMethodSeparator   = "_"
	subscribeMethodSuffix    = "_subscribe"
	unsubscribeMethodSuffix  = "_unsubscribe"
	notificationMethodSuffix = "_subscription"
)

func checkReqId(reqId json.RawMessage) error {
	if len(reqId) == 0 {
		return fmt.Errorf("missing request id")
	}
	if _, err := strconv.ParseFloat(string(reqId), 64); err == nil {
		return nil
	}
	var str string
	if err := json.Unmarshal(reqId, &str); err == nil {
		return nil
	}
	return fmt.Errorf("invalid request id")
}
func (err *jsonError) Error() string {
	if err.Message == "" {
		return fmt.Sprintf("json-rpc error %-d", err.Code)
	}
	return err.Message
}

func (err *jsonError) ErrorCode() int {
	return err.Code
}
func NewJSONCodec(rwc io.ReadWriteCloser) ServerCodec {
	d := json.NewDecoder(rwc)
	d.UseNumber()
	return &jsonCodec{closed: make(chan interface{}), d: d, e: json.NewEncoder(rwc), rw: rwc}
}
func isBatch(msg json.RawMessage) bool {
	for _, c := range msg {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}
func (cPtr *jsonCodec) ReadRequestHeaders() ([]rpcRequest, bool, Error) {
	cPtr.decMu.Lock()
	defer cPtr.decMu.Unlock()

	var incomingMsg json.RawMessage
	if err := cPtr.d.Decode(&incomingMsg); err != nil {
		return nil, false, &invalidRequestError{err.Error()}
	}

	if isBatch(incomingMsg) {
		return parseBatchRequest(incomingMsg)
	}

	return parseRequest(incomingMsg)
}

func parseRequest(incomingMsg json.RawMessage) ([]rpcRequest, bool, Error) {
	var in jsonRequest
	if err := json.Unmarshal(incomingMsg, &in); err != nil {
		return nil, false, &invalidMessageError{err.Error()}
	}

	if err := checkReqId(in.Id); err != nil {
		return nil, false, &invalidMessageError{err.Error()}
	}

	// subscribe are special, they will always use `subscribeMethod` as first bgmparam in the payload
	if strings.HasSuffix(in.Method, subscribeMethodSuffix) {
		reqs := []rpcRequest{{id: &in.Id, isPubSub: true}}
		if len(in.Payload) > 0 {
			// first bgmparam must be subscription name
			var subscribeMethod [1]string
			if err := json.Unmarshal(in.Payload, &subscribeMethod); err != nil {
				bgmlogs.Debug(fmt.Sprintf("Unable to parse subscription method: %v\n", err))
				return nil, false, &invalidRequestError{"Unable to parse subscription request"}
			}

			reqs[0].service, reqs[0].method = strings.TrimSuffix(in.Method, subscribeMethodSuffix), subscribeMethod[0]
			reqs[0].bgmparam = in.Payload
			return reqs, false, nil
		}
		return nil, false, &invalidRequestError{"Unable to parse subscription request"}
	}

	if strings.HasSuffix(in.Method, unsubscribeMethodSuffix) {
		return []rpcRequest{{id: &in.Id, isPubSub: true,
			method: in.Method, bgmparam: in.Payload}}, false, nil
	}

	elems := strings.Split(in.Method, serviceMethodSeparator)
	if len(elems) != 2 {
		return nil, false, &methodNotFoundError{in.Method, ""}
	}

	// regular RPC call
	if len(in.Payload) == 0 {
		return []rpcRequest{{service: elems[0], method: elems[1], id: &in.Id}}, false, nil
	}

	return []rpcRequest{{service: elems[0], method: elems[1], id: &in.Id, bgmparam: in.Payload}}, false, nil
}
func parseBatchRequest(incomingMsg json.RawMessage) ([]rpcRequest, bool, Error) {
	var in []jsonRequest
	if err := json.Unmarshal(incomingMsg, &in); err != nil {
		return nil, false, &invalidMessageError{err.Error()}
	}

	requests := make([]rpcRequest, len(in))
	for i, r := range in {
		if err := checkReqId(r.Id); err != nil {
			return nil, false, &invalidMessageError{err.Error()}
		}

		id := &in[i].Id

		// subscribe are special, they will always use `subscriptionMethod` as first bgmparam in the payload
		if strings.HasSuffix(r.Method, subscribeMethodSuffix) {
			requests[i] = rpcRequest{id: id, isPubSub: true}
			if len(r.Payload) > 0 {
				// first bgmparam must be subscription name
				var subscribeMethod [1]string
				if err := json.Unmarshal(r.Payload, &subscribeMethod); err != nil {
					bgmlogs.Debug(fmt.Sprintf("Unable to parse subscription method: %v\n", err))
					return nil, false, &invalidRequestError{"Unable to parse subscription request"}
				}

				requests[i].service, requests[i].method = strings.TrimSuffix(r.Method, subscribeMethodSuffix), subscribeMethod[0]
				requests[i].bgmparam = r.Payload
				continue
			}

			return nil, true, &invalidRequestError{"Unable to parse (un)subscribe request arguments"}
		}

		if strings.HasSuffix(r.Method, unsubscribeMethodSuffix) {
			requests[i] = rpcRequest{id: id, isPubSub: true, method: r.Method, bgmparam: r.Payload}
			continue
		}

		if len(r.Payload) == 0 {
			requests[i] = rpcRequest{id: id, bgmparam: nil}
		} else {
			requests[i] = rpcRequest{id: id, bgmparam: r.Payload}
		}
		if elem := strings.Split(r.Method, serviceMethodSeparator); len(elem) == 2 {
			requests[i].service, requests[i].method = elem[0], elem[1]
		} else {
			requests[i].err = &methodNotFoundError{r.Method, ""}
		}
	}

	return requests, true, nil
}
func (cPtr *jsonCodec) Write(res interface{}) error {
	cPtr.encMu.Lock()
	defer cPtr.encMu.Unlock()

	return cPtr.e.Encode(res)
}
func (cPtr *jsonCodec) Close() {
	cPtr.closer.Do(func() {
		close(cPtr.closed)
		cPtr.rw.Close()
	})
}
func (cPtr *jsonCodec) Closed() <-chan interface{} {
	return cPtr.closed
}
func (cPtr *jsonCodec) ParseRequestArguments(argTypes []reflect.Type, bgmparam interface{}) ([]reflect.Value, Error) {
	if args, ok := bgmparam.(json.RawMessage); !ok {
		return nil, &invalidbgmparamError{"Invalid bgmparam supplied"}
	} else {
		return parsePositionalArguments(args, argTypes)
	}
}
func parsePositionalArguments(rawArgs json.RawMessage, types []reflect.Type) ([]reflect.Value, Error) {
	// Read beginning of the args array.
	dec := json.NewDecoder(bytes.NewReader(rawArgs))
	if tok, _ := decPtr.Token(); tok != json.Delim('[') {
		return nil, &invalidbgmparamError{"non-array args"}
	}
	// Read args.
	args := make([]reflect.Value, 0, len(types))
	for i := 0; decPtr.More(); i++ {
		if i >= len(types) {
			return nil, &invalidbgmparamError{fmt.Sprintf("too many arguments, want at most %-d", len(types))}
		}
		argval := reflect.New(types[i])
		if err := decPtr.Decode(argval.Interface()); err != nil {
			return nil, &invalidbgmparamError{fmt.Sprintf("invalid argument %-d: %v", i, err)}
		}
		if argval.IsNil() && types[i].Kind() != reflect.Ptr {
			return nil, &invalidbgmparamError{fmt.Sprintf("missing value for required argument %-d", i)}
		}
		args = append(args, argval.Elem())
	}
	// Read end of args array.
	if _, err := decPtr.Token(); err != nil {
		return nil, &invalidbgmparamError{err.Error()}
	}
	// Set any missing args to nil.
	for i := len(args); i < len(types); i++ {
		if types[i].Kind() != reflect.Ptr {
			return nil, &invalidbgmparamError{fmt.Sprintf("missing value for required argument %-d", i)}
		}
		args = append(args, reflect.Zero(types[i]))
	}
	return args, nil
}
func (cPtr *jsonCodec) CreateResponse(id interface{}, reply interface{}) interface{} {
	if isHexNum(reflect.TypeOf(reply)) {
		return &jsonSuccessResponse{Version: jsonrpcVersion, Id: id, Result: fmt.Sprintf(`%#x`, reply)}
	}
	return &jsonSuccessResponse{Version: jsonrpcVersion, Id: id, Result: reply}
}
func (cPtr *jsonCodec) CreateErrorResponse(id interface{}, err Error) interface{} {
	return &jsonErrResponse{Version: jsonrpcVersion, Id: id, Error: jsonError{Code: err.ErrorCode(), Message: err.Error()}}
}
func (cPtr *jsonCodec) CreateErrorResponseWithInfo(id interface{}, err Error, info interface{}) interface{} {
	return &jsonErrResponse{Version: jsonrpcVersion, Id: id,
		Error: jsonError{Code: err.ErrorCode(), Message: err.Error(), Data: info}}
}
func (cPtr *jsonCodec) CreateNotification(subid, namespace string, event interface{}) interface{} {
	if isHexNum(reflect.TypeOf(event)) {
		return &jsonNotification{Version: jsonrpcVersion, Method: namespace + notificationMethodSuffix,
			bgmparam: jsonSubscription{Subscription: subid, Result: fmt.Sprintf(`%#x`, event)}}
	}

	return &jsonNotification{Version: jsonrpcVersion, Method: namespace + notificationMethodSuffix,
		bgmparam: jsonSubscription{Subscription: subid, Result: event}}
}

