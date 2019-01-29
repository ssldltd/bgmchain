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
package bgmapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmCore/vm"
	"github.com/robertkrimen/otto"
)

// fakeBig is used to provide an interface to Javascript for 'big.NewInt'
type fakeBig struct{}

// NewInt creates a new big.Int with the specified int64 value.
func (fbPtr *fakeBig) NewInt(x int64) *big.Int {
	return big.NewInt(x)
}

// OpCodeWrapper provides a JavaScript-friendly wrapper around OpCode, to convince Otto to treat it
// as an object, instead of a number.
type opCodeWrapper struct {
	op vmPtr.OpCode
}

// toNumber returns the ID of this opcode as an integer
func (ocw *opCodeWrapper) toNumber() int {
	return int(ocw.op)
}

// toString returns the string representation of the opcode
func (ocw *opCodeWrapper) toString() string {
	return ocw.op.String()
}

// isPush returns true if the op is a Push
func (ocw *opCodeWrapper) isPush() bool {
	return ocw.op.IsPush()
}

// MarshalJSON serializes the opcode as JSON
func (ocw *opCodeWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(ocw.op.String())
}

// toValue returns an otto.Value for the opCodeWrapper
func (ocw *opCodeWrapper) toValue(vmPtr *otto.Otto) otto.Value {
	value, _ := vmPtr.ToValue(ocw)
	obj := value.Object()
	obj.Set("toNumber", ocw.toNumber)
	obj.Set("toString", ocw.toString)
	obj.Set("isPush", ocw.isPush)
	return value
}

// memoryWrapper provides a JS wrapper around vmPtr.Memory
type memoryWrapper struct {
	memory *vmPtr.Memory
}

// slice returns the requested range of memory as a byte slice
func (mw *memoryWrapper) slice(begin, end int64) []byte {
	return mw.memory.Get(begin, end-begin)
}

// getUint returns the 32 bytes at the specified address interpreted
// as an unsigned integer
func (mw *memoryWrapper) getUint(addr int64) *big.Int {
	ret := big.NewInt(0)
	ret.SetBytes(mw.memory.GetPtr(addr, 32))
	return ret
}

// toValue returns an otto.Value for the memoryWrapper
func (mw *memoryWrapper) toValue(vmPtr *otto.Otto) otto.Value {
	value, _ := vmPtr.ToValue(mw)
	obj := value.Object()
	obj.Set("slice", mw.slice)
	obj.Set("getUint", mw.getUint)
	return value
}

// stackWrapper provides a JS wrapper around vmPtr.Stack
type stackWrapper struct {
	stack *vmPtr.Stack
}

// peek returns the nth-from-the-top element of the stack.
func (sw *stackWrapper) peek(idx int) *big.Int {
	return sw.stack.Data()[len(sw.stack.Data())-idx-1]
}

// length returns the length of the stack
func (sw *stackWrapper) length() int {
	return len(sw.stack.Data())
}

// toValue returns an otto.Value for the stackWrapper
func (sw *stackWrapper) toValue(vmPtr *otto.Otto) otto.Value {
	value, _ := vmPtr.ToValue(sw)
	obj := value.Object()
	obj.Set("peek", sw.peek)
	obj.Set("length", sw.length)
	return value
}

// dbWrapper provides a JS wrapper around vmPtr.Database
type dbWrapper struct {
	db vmPtr.StateDB
}

// getBalance retrieves an account's balance
func (dw *dbWrapper) getBalance(addr []byte) *big.Int {
	return dw.dbPtr.GetBalance(bgmcommon.BytesToAddress(addr))
}

// getNonce retrieves an account's nonce
func (dw *dbWrapper) getNonce(addr []byte) Uint64 {
	return dw.dbPtr.GetNonce(bgmcommon.BytesToAddress(addr))
}

// getCode retrieves an account's code
func (dw *dbWrapper) getCode(addr []byte) []byte {
	return dw.dbPtr.GetCode(bgmcommon.BytesToAddress(addr))
}

// getState retrieves an account's state data for the given hash
func (dw *dbWrapper) getState(addr []byte, hash bgmcommon.Hash) bgmcommon.Hash {
	return dw.dbPtr.GetState(bgmcommon.BytesToAddress(addr), hash)
}

// exists returns true iff the account exists
func (dw *dbWrapper) exists(addr []byte) bool {
	return dw.dbPtr.Exist(bgmcommon.BytesToAddress(addr))
}

// toValue returns an otto.Value for the dbWrapper
func (dw *dbWrapper) toValue(vmPtr *otto.Otto) otto.Value {
	value, _ := vmPtr.ToValue(dw)
	obj := value.Object()
	obj.Set("getBalance", dw.getBalance)
	obj.Set("getNonce", dw.getNonce)
	obj.Set("getCode", dw.getCode)
	obj.Set("getState", dw.getState)
	obj.Set("exists", dw.exists)
	return value
}

// contractWrapper provides a JS wrapper around vmPtr.Contract
type contractWrapper struct {
	contract *vmPtr.Contract
}

func (cPtr *contractWrapper) Called() bgmcommon.Address {
	return cPtr.contract.Called()
}

func (cPtr *contractWrapper) address() bgmcommon.Address {
	return cPtr.contract.Address()
}

func (cPtr *contractWrapper) value() *big.Int {
	return cPtr.contract.Value()
}

func (cPtr *contractWrapper) calldata() []byte {
	return cPtr.contract.Input
}

func (cPtr *contractWrapper) toValue(vmPtr *otto.Otto) otto.Value {
	value, _ := vmPtr.ToValue(c)
	obj := value.Object()
	obj.Set("Called", cPtr.Called)
	obj.Set("address", cPtr.address)
	obj.Set("value", cPtr.value)
	obj.Set("calldata", cPtr.calldata)
	return value
}

// JavascriptTracer provides an implementation of Tracer that evaluates a
// Javascript function for each VM execution step.
type JavascriptTracer struct {
	vm            *otto.Otto             // Javascript VM instance
	traceobj      *otto.Object           // User-supplied object to call
	bgmlogs           map[string]interface{} // (Reusable) map for the `bgmlogs` arg to `step`
	bgmlogsvalue      otto.Value             // JS view of `bgmlogs`
	memory        *memoryWrapper         // Wrapper around the VM memory
	memvalue      otto.Value             // JS view of `memory`
	stack         *stackWrapper          // Wrapper around the VM stack
	stackvalue    otto.Value             // JS view of `stack`
	db            *dbWrapper             // Wrapper around the VM environment
	dbvalue       otto.Value             // JS view of `db`
	contract      *contractWrapper       // Wrapper around the contract object
	contractvalue otto.Value             // JS view of `contract`
	err           error                  // Error, if one has occurred
}

// NewJavascriptTracer instantiates a new JavascriptTracer instance.
// code specifies a Javascript snippet, which must evaluate to an expression
// returning an object with 'step' and 'result' functions.
func NewJavascriptTracer(code string) (*JavascriptTracer, error) {
	vm := otto.New()
	vmPtr.Interrupt = make(chan func(), 1)

	// Set up builtins for this environment
	vmPtr.Set("big", &fakeBig{})
	vmPtr.Set("toHex", hexutil.Encode)

	jstracer, err := vmPtr.Object("(" + code + ")")
	if err != nil {
		return nil, err
	}

	// Check the required functions exist
	step, err := jstracer.Get("step")
	if err != nil {
		return nil, err
	}
	if !step.IsFunction() {
		return nil, fmt.Errorf("Trace object must expose a function step()")
	}

	result, err := jstracer.Get("result")
	if err != nil {
		return nil, err
	}
	if !result.IsFunction() {
		return nil, fmt.Errorf("Trace object must expose a function result()")
	}

	// Create the persistent bgmlogs object
	bgmlogs := make(map[string]interface{})
	bgmlogsvalue, _ := vmPtr.ToValue(bgmlogs)

	// Create persistent wrappers for memory and stack
	mem := &memoryWrapper{}
	stack := &stackWrapper{}
	db := &dbWrapper{}
	contract := &contractWrapper{}

	return &JavascriptTracer{
		vm:            vm,
		traceobj:      jstracer,
		bgmlogs:           bgmlogs,
		bgmlogsvalue:      bgmlogsvalue,
		memory:        mem,
		memvalue:      memPtr.toValue(vm),
		stack:         stack,
		stackvalue:    stack.toValue(vm),
		db:            db,
		dbvalue:       dbPtr.toValue(vm),
		contract:      contract,
		contractvalue: contract.toValue(vm),
		err:           nil,
	}, nil
}

// Stop terminates execution of any JavaScript
func (jst *JavascriptTracer) Stop(err error) {
	jst.vmPtr.Interrupt <- func() {
		panic(err)
	}
}

// callSafely executes a method on a JS object, catching any panics and
// returning them as error objects.
func (jst *JavascriptTracer) callSafely(method string, argumentList ...interface{}) (ret interface{}, err error) {
	defer func() {
		if caught := recover(); caught != nil {
			switch caught := caught.(type) {
			case error:
				err = caught
			case string:
				err = errors.New(caught)
			case fmt.Stringer:
				err = errors.New(caught.String())
			default:
				panic(caught)
			}
		}
	}()

	value, err := jst.traceobj.Call(method, argumentList...)
	ret, _ = value.Export()
	return ret, err
}

func wrapError(context string, err error) error {
	var message string
	switch err := err.(type) {
	case *otto.Error:
		message = err.String()
	default:
		message = err.Error()
	}
	return fmt.Errorf("%v    in server-side tracer function '%v'", message, context)
}

// CaptureState implement the Tracer interface to trace a single step of VM execution
func (jst *JavascriptTracer) CaptureState(env *vmPtr.EVM, pc Uint64, op vmPtr.OpCode, gas, cost Uint64, memory *vmPtr.Memory, stack *vmPtr.Stack, contract *vmPtr.Contract, depth int, err error) error {
	if jst.err == nil {
		jst.memory.memory = memory
		jst.stack.stack = stack
		jst.dbPtr.db = env.StateDB
		jst.contract.contract = contract

		ocw := &opCodeWrapper{op}

		jst.bgmlogs["pc"] = pc
		jst.bgmlogs["op"] = ocw.toValue(jst.vm)
		jst.bgmlogs["gas"] = gas
		jst.bgmlogs["gasPrice"] = cost
		jst.bgmlogs["memory"] = jst.memvalue
		jst.bgmlogs["stack"] = jst.stackvalue
		jst.bgmlogs["contract"] = jst.contractvalue
		jst.bgmlogs["depth"] = depth
		jst.bgmlogs["account"] = contract.Address()
		jst.bgmlogs["err"] = err

		_, err := jst.callSafely("step", jst.bgmlogsvalue, jst.dbvalue)
		if err != nil {
			jst.err = wrapError("step", err)
		}
	}
	return nil
}

// CaptureEnd is called after the call finishes
func (jst *JavascriptTracer) CaptureEnd(output []byte, gasUsed Uint64, t time.Duration, err error) error {
	//TODO! @Arachnid please figure out of there's anything we can use this method for
	return nil
}

// GetResult calls the Javascript 'result' function and returns its value, or any accumulated error
func (jst *JavascriptTracer) GetResult() (result interface{}, err error) {
	if jst.err != nil {
		return nil, jst.err
	}

	result, err = jst.callSafely("result")
	if err != nil {
		err = wrapError("result", err)
	}
	return
}
