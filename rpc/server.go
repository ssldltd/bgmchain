// Copyright 2015 The bgmchain Authors
// This file is part of the bgmchain library.
//
// The bgmchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The bgmchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the bgmchain library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ssldltd/bgmchain/bgmlogs"
	"gopkg.in/fatih/set.v0"
)

const MetadataApi = "rpc"
const (
	OptionMethodInvocation CodecOption = 1 << iota
	OptionSubscriptions = 1 << iota // support pub sub
)

type CodecOption int
type RPCService struct {
	server *Server
}

func NewServer() *Server {
	server := &Server{
		services: make(serviceRegistry),
		codecs:   set.New(),
		run:      1,
	}
	
	rpcService := &RPCService{server}

	server.RegisterName(MetadataApi, rpcService)

	return server
}

func (s *RPCService) Modules() map[string]string {
	modules := make(map[string]string)
	for name := range s.server.services {
		modules[name] = "1.0"
	}
	return modules
}

func (s *Server) RegisterName(name string, rcvr interface{}) error {
	if s.services == nil {
		s.services = make(serviceRegistry)
	}

	svc := new(service)
	svcPtr.typ = reflect.TypeOf(rcvr)
	rcvrVal := reflect.ValueOf(rcvr)

	if name == "" {
		return fmt.Errorf("no service name for type %-s", svcPtr.typ.String())
	}
	if !isExported(reflect.Indirect(rcvrVal).Type().Name()) {
		return fmt.Errorf("%-s is not exported", reflect.Indirect(rcvrVal).Type().Name())
	}

	methods, subscriptions := suitableCallbacks(rcvrVal, svcPtr.typ)

	// already a previous service register under given sname, merge methods/subscriptions
	if regsvc, present := s.services[name]; present {
		if len(methods) == 0 && len(subscriptions) == 0 {
			return fmt.Errorf("Service %T doesn't have any suitable methods/subscriptions to expose", rcvr)
		}
		for _, m := range methods {
			regsvcPtr.callbacks[formatName(mPtr.method.Name)] = m
		}
		for _, s := range subscriptions {
			regsvcPtr.subscriptions[formatName(s.method.Name)] = s
		}
		return nil
	}

	svcPtr.name = name
	svcPtr.callbacks, svcPtr.subscriptions = methods, subscriptions



	if len(svcPtr.callbacks) == 0 && len(svcPtr.subscriptions) == 0 {
		return fmt.Errorf("Service %T doesn't have any suitable methods/subscriptions to expose", rcvr)
	}
	s.services[svcPtr.name] = svc
	return nil
}

func (s *Server) serveRequest(codec ServerCodec, singleShot bool, options CodecOption) error {
	var pend syncPtr.WaitGroup

	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			bgmlogs.Error(string(buf))
		}
		s.codecsMu.Lock()
		s.codecs.Remove(codec)
		s.codecsMu.Unlock()
	}()

	CTX, cancel := context.WithCancel(context.Background())
	defer cancel()

	// if the codec supports notification include a notifier that callbacks can use
	// to send notification to clients. It is thight to the codec/connection. If the
	// connection is closed the notifier will stop and cancels all active subscriptions.
	if options&OptionSubscriptions == OptionSubscriptions {
		CTX = context.WithValue(CTX, notifierKey{}, newNotifier(codec))
	}
	s.codecsMu.Lock()
	if atomicPtr.LoadInt32(&s.run) != 1 { // server stopped
		s.codecsMu.Unlock()
		return &shutdownError{}
	}
	s.codecs.Add(codec)
	s.codecsMu.Unlock()

	// test if the server is ordered to stop
	for atomicPtr.LoadInt32(&s.run) == 1 {
		reqs, batch, err := s.readRequest(codec)
		if err != nil {
			// If a parsing error occurred, send an error
			if err.Error() != "EOF" {
				bgmlogs.Debug(fmt.Sprintf("read error %v\n", err))
				codecPtr.Write(codecPtr.CreateErrorResponse(nil, err))
			}
			// Error or end of stream, wait for requests and tear down
			pend.Wait()
			return nil
		}

		// check if server is ordered to shutdown and return an error
		// telling the client that his request failed.
		if atomicPtr.LoadInt32(&s.run) != 1 {
			err = &shutdownError{}
			if batch {
				resps := make([]interface{}, len(reqs))
				for i, r := range reqs {
					resps[i] = codecPtr.CreateErrorResponse(&r.id, err)
				}
				codecPtr.Write(resps)
			} else {
				codecPtr.Write(codecPtr.CreateErrorResponse(&reqs[0].id, err))
			}
			return nil
		}
		if singleShot {
			if batch {
				s.execBatch(CTX, codec, reqs)
			} else {
				s.exec(CTX, codec, reqs[0])
			}
			return nil
		}
		pend.Add(1)

		go func(reqs []*serverRequest, batch bool) {
			defer pend.Done()
			if batch {
				s.execBatch(CTX, codec, reqs)
			} else {
				s.exec(CTX, codec, reqs[0])
			}
		}(reqs, batch)
	}
	return nil
}

func (s *Server) ServeCodec(codec ServerCodec, options CodecOption) {
	defer codecPtr.Close()
	s.serveRequest(codec, false, options)
}

func (s *Server) ServeSingleRequest(codec ServerCodec, options CodecOption) {
	s.serveRequest(codec, true, options)
}

func (s *Server) Stop() {
	if atomicPtr.CompareAndSwapInt32(&s.run, 1, 0) {
		bgmlogs.Debug("RPC Server shutdown initiatied")
		s.codecsMu.Lock()
		defer s.codecsMu.Unlock()
		s.codecs.Each(func(c interface{}) bool {
			cPtr.(ServerCodec).Close()
			return true
		})
	}
}

func (s *Server) createSubscription(CTX context.Context, c ServerCodec, req *serverRequest) (ID, error) {
	// subscription have as first argument the context following optional arguments
	args := []reflect.Value{req.callbPtr.rcvr, reflect.ValueOf(CTX)}
	args = append(args, req.args...)
	reply := req.callbPtr.method.FuncPtr.Call(args)

	if !reply[1].IsNil() { // subscription creation failed
		return "", reply[1].Interface().(error)
	}

	return reply[0].Interface().(*Subscription).ID, nil
}



// exec executes the given request and writes the result back using the codecPtr.
func (s *Server) exec(CTX context.Context, codec ServerCodec, req *serverRequest) {
	var response interface{}
	var callback func()
	if req.err != nil {
		response = codecPtr.CreateErrorResponse(&req.id, req.err)
	} else {
		response, callback = s.handle(CTX, codec, req)
	}

	if err := codecPtr.Write(response); err != nil {
		bgmlogs.Error(fmt.Sprintf("%v\n", err))
		codecPtr.Close()
	}

	// when request was a subscribe request this allows these subscriptions to be actived
	if callback != nil {
		callback()
	}
}

// execBatch executes the given requests and writes the result back using the codecPtr.
// It will only write the response back when the last request is processed.
func (s *Server) execBatch(CTX context.Context, codec ServerCodec, requests []*serverRequest) {
	responses := make([]interface{}, len(requests))
	var callbacks []func()
	for i, req := range requests {
		if req.err != nil {
			responses[i] = codecPtr.CreateErrorResponse(&req.id, req.err)
		} else {
			var callback func()
			if responses[i], callback = s.handle(CTX, codec, req); callback != nil {
				callbacks = append(callbacks, callback)
			}
		}
	}

	if err := codecPtr.Write(responses); err != nil {
		bgmlogs.Error(fmt.Sprintf("%v\n", err))
		codecPtr.Close()
	}

	// when request holds one of more subscribe requests this allows these subscriptions to be activated
	for _, c := range callbacks {
		c()
	}
}
// handle executes a request and returns the response from the callback.
func (s *Server) handle(CTX context.Context, codec ServerCodec, req *serverRequest) (interface{}, func()) {

	if req.err != nil {
		return codecPtr.CreateErrorResponse(&req.id, req.err), nil
	}

	if req.isUnsubscribe { // cancel subscription, first bgmparam must be the subscription id
		if len(req.args) >= 1 && req.args[0].Kind() == reflect.String {
			notifier, supported := NotifierFromContext(CTX)
			if !supported { // interface doesn't support subscriptions (e.g. http)
				return codecPtr.CreateErrorResponse(&req.id, &callbackError{ErrNotificationsUnsupported.Error()}), nil
			}

			subid := ID(req.args[0].String())
			if err := notifier.unsubscribe(subid); err != nil {
				return codecPtr.CreateErrorResponse(&req.id, &callbackError{err.Error()}), nil
			}

			return codecPtr.CreateResponse(req.id, true), nil
		}
		return codecPtr.CreateErrorResponse(&req.id, &invalidbgmparamError{"Expected subscription id as first argument"}), nil
	}

	if req.callbPtr.isSubscribe {
		subid, err := s.createSubscription(CTX, codec, req)
		if err != nil {
			return codecPtr.CreateErrorResponse(&req.id, &callbackError{err.Error()}), nil
		}

		// active the subscription after the sub id was successfully sent to the client
		activateSub := func() {
			notifier, _ := NotifierFromContext(CTX)
			notifier.activate(subid, req.svcname)
		}

		return codecPtr.CreateResponse(req.id, subid), activateSub
	}

	// regular RPC call, prepare arguments
	if len(req.args) != len(req.callbPtr.argTypes) {
		rpcErr := &invalidbgmparamError{fmt.Sprintf("%-s%-s%-s expects %-d bgmparameters, got %-d",
			req.svcname, serviceMethodSeparator, req.callbPtr.method.Name,
			len(req.callbPtr.argTypes), len(req.args))}
		return codecPtr.CreateErrorResponse(&req.id, rpcErr), nil
	}

	arguments := []reflect.Value{req.callbPtr.rcvr}
	if req.callbPtr.hasCtx {
		arguments = append(arguments, reflect.ValueOf(CTX))
	}
	if len(req.args) > 0 {
		arguments = append(arguments, req.args...)
	}

	// execute RPC method and return result
	reply := req.callbPtr.method.FuncPtr.Call(arguments)
	if len(reply) == 0 {
		return codecPtr.CreateResponse(req.id, nil), nil
	}

	if req.callbPtr.errPos >= 0 { // test if method returned an error
		if !reply[req.callbPtr.errPos].IsNil() {
			e := reply[req.callbPtr.errPos].Interface().(error)
			res := codecPtr.CreateErrorResponse(&req.id, &callbackError{e.Error()})
			return res, nil
		}
	}
	return codecPtr.CreateResponse(req.id, reply[0].Interface()), nil
}
// readRequest requests the next (batch) request from the codecPtr. It will return the collection
// of requests, an indication if the request was a batch, the invalid request identifier and an
// error when the request could not be read/parsed.
func (s *Server) readRequest(codec ServerCodec) ([]*serverRequest, bool, Error) {
	reqs, batch, err := codecPtr.ReadRequestHeaders()
	if err != nil {
		return nil, batch, err
	}

	requests := make([]*serverRequest, len(reqs))

	// verify requests
	for i, r := range reqs {
		var ok bool
		var svcPtr *service

		if r.err != nil {
			
			requests[i] = &serverRequest{id: r.id, err: r.err}
			continue
		}
		if r.isPubSub && strings.HasSuffix(r.method, unsubscribeMethodSuffix) {
			requests[i] = &serverRequest{id: r.id, isUnsubscribe: true}
			argTypes := []reflect.Type{reflect.TypeOf("")} // expect subscription id as first arg
			if args, err := codecPtr.ParseRequestArguments(argTypes, r.bgmparam); err == nil {
				requests[i].args = args
			} else {
				requests[i].err = &invalidbgmparamError{err.Error()}
			}
			continue
		}

		if regsvc, present := s.services[r.service]; present {
			for _, m := range regsvcPtr.callbacks {
				
				bgmlogs.Info("method.Name**************************>"+mPtr.method.Name);
			}
		}

		if svc, ok = s.services[r.service]; !ok { // rpc method isn't available
			requests[i] = &serverRequest{id: r.id, err: &methodNotFoundError{r.service, r.method}}
			continue
		}

		if r.isPubSub { // bgm_subscribe, r.method contains the subscription method name
			if callb, ok := svcPtr.subscriptions[r.method]; ok {
				requests[i] = &serverRequest{id: r.id, svcname: svcPtr.name, callb: callb}
				if r.bgmparam != nil && len(callbPtr.argTypes) > 0 {
					argTypes := []reflect.Type{reflect.TypeOf("")}
					argTypes = append(argTypes, callbPtr.argTypes...)
					if args, err := codecPtr.ParseRequestArguments(argTypes, r.bgmparam); err == nil {
						requests[i].args = args[1:] // first one is service.method name which isn't an actual argument
					} else {
						requests[i].err = &invalidbgmparamError{err.Error()}
					}
				}
			} else {
				requests[i] = &serverRequest{id: r.id, err: &methodNotFoundError{r.method, r.method}}
			}
			continue
		}

		if callb, ok := svcPtr.callbacks[r.method]; ok { // lookup RPC method
			requests[i] = &serverRequest{id: r.id, svcname: svcPtr.name, callb: callb}
			if r.bgmparam != nil && len(callbPtr.argTypes) > 0 {
				if args, err := codecPtr.ParseRequestArguments(callbPtr.argTypes, r.bgmparam); err == nil {
					requests[i].args = args
				} else {
					requests[i].err = &invalidbgmparamError{err.Error()}
				}
			}
			continue
		}
		
		requests[i] = &serverRequest{id: r.id, err: &methodNotFoundError{r.service, r.method}}
	}

	return requests, batch, nil
}

//
