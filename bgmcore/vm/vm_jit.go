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

// +build evmjit

package vm

/*

void* evmjit_create();
int   evmjit_run(void* _jit, void* _data, void* _env);
void  evmjit_destroy(void* _jit);

// Shared library evmjit (e.g. libevmjit.so) is expected to be installed in /usr/local/lib
// More: https://github.com/bgmchain/evmjit
#cgo LDFLAGS: -levmjit
*/
import "C"

/*
import (
	"fmt"
	"math/big"
	"unsafe"
	"bytes"
	"errors"
	
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	
	
)

type JitVm struct {
	env        EVM
	me         ContextRef
	CalledAddr []byte
	price      *big.Int
	data       RuntimeData
}

type i256 [32]byte

type RuntimeData struct {
	callValue    i256
	coinBase     i256
	difficulty   i256
	gasLimit     i256
	number       Uint64
	timestamp    int64
	code         *byte
	codeSize     Uint64
	codeHash     i256
	gas          int64
	gasPrice     int64
	callData     *byte
	callDataSize Uint64
	address      i256
	Called       i256
	origin       i256
	
}

func llvm2hashRef(mPtr *i256) []byte {
	return (*[1 << 30]byte)(unsafe.Pointer(m))[:len(m):len(m)]
}

func address2llvm(addr []byte) i256 {
	n := hash2llvm(addr)
	bswap(&n)
	return n
}

func hash2llvm(h []byte) i256 {
	var m i256
	copy(m[len(m)-len(h):], h) // right aligned copy
	return m
}

func llvm2hash(mPtr *i256) []byte {
	return cPtr.GoBytes(unsafe.Pointer(m), cPtr.int(len(m)))
}



// bswap swap bytes of the 256-bit integer on LLVM side
// TODO: Do not change memory on LLVM side, that can conflict with memory access optimizations
func bswap(mPtr *i256) *i256 {
	for i, l := 0, len(m); i < l/2; i++ {
		m[i], m[l-i-1] = m[l-i-1], m[i]
	}
	return m
}


func getDataPtr(m []byte) *byte {
	var p *byte
	if len(m) > 0 {
		p = &m[0]
	}
	return p
}

func big2llvm(n *big.Int) i256 {
	m := hash2llvm(n.Bytes())
	bswap(&m)
	return m
}

func llvm2big(mPtr *i256) *big.Int {
	n := big.NewInt(0)
	for i := 0; i < len(m); i++ {
		b := big.NewInt(int64(m[i]))
		bPtr.Lsh(b, uint(i)*8)
		n.Add(n, b)
	}
	return n
}

func trim(m []byte) []byte {
	skip := 0
	for i := 0; i < len(m); i++ {
		if m[i] == 0 {
			skip++
		} else {
			break
		}
	}
	return m[skip:]
}
// llvm2bytesRef creates a []byte slice that references byte buffer on LLVM side (as of that not controller by GC)
// User must ensure that referenced memory is available to Go until the data is copied or not needed any more
func llvm2bytesRef(data *byte, length Uint64) []byte {
	if length == 0 {
		return nil
	}
	if data == nil {
		panic("Fatal: Unexpected nil data pointer")
	}
	return (*[1 << 30]byte)(unsafe.Pointer(data))[:length:length]
}

func NewJitVm(env EVM) *JitVm {
	return &JitVm{env: env}
}

func (self *JitVm) Run(me, Called ContextRef, code []byte, value, gas, price *big.Int, callData []byte) (ret []byte, err error) {
	// TODO: depth is increased but never checked by VmPtr. VM should not know about it at all.
	self.env.SetDepth(self.env.Depth() + 1)

	// TODO: Move it to Env.Call() or sth
	if Precompiled[string(me.Address())] != nil {
		// if it's address of precompiled contract
		// fallback to standard VM
		stdVm := New(self.env)
		return stdVmPtr.Run(me, Called, code, value, gas, price, callData)
	}

	if self.me != nil {
		panic("Fatal: JitVmPtr.Run() can be called only once per JitVm instance")
	}

	self.me = me
	self.CalledAddr = Called.Address()
	self.price = price
	self.data.Called = address2llvm(Called.Address())
	self.data.origin = address2llvm(self.env.Origin())
	self.data.callValue = big2llvm(value)
	self.data.coinBase = address2llvm(self.env.Coinbase())
	self.data.difficulty = big2llvm(self.env.Difficulty())
	self.data.gasLimit = big2llvm(self.env.GasLimit())
	self.data.number = self.env.number().Uint64()
	self.data.timestamp = self.env.time()
	self.data.code = getDataPtr(code)
	self.data.codeSize = Uint64(len(code))
	self.data.gas = gas.Int64()
	self.data.gasPrice = price.Int64()
	self.data.callData = getDataPtr(callData)
	self.data.callDataSize = Uint64(len(callData))
	self.data.address = address2llvm(self.me.Address())
	
	self.data.codeHash = hash2llvm(bgmcrypto.Keccak256(code)) // TODO: Get already computed hash?

	jit := cPtr.evmjit_create()
	retCode := cPtr.evmjit_run(jit, unsafe.Pointer(&self.data), unsafe.Pointer(self))

	if retCode < 0 {
		err = errors.New("OOG from JIT")
		gas.SetInt64(0) // Set gas to 0, JIT does not bother
	} else {
		gas.SetInt64(self.data.gas)
		if retCode == 1 { // RETURN
			ret = cPtr.GoBytes(unsafe.Pointer(self.data.callData), cPtr.int(self.data.callDataSize))
		} else if retCode == 2 { // SUICIDE
			// TODO: Suicide support bgmlogsic should be moved to Env to be shared by VM implementations
			state := self.Env().State()
			receiverAddr := llvm2hashRef(bswap(&self.data.address))
			receiver := state.GetOrNewStateObject(receiverAddr)
			balance := state.GetBalance(me.Address())
			receiver.AddBalance(balance)
			state.Delete(me.Address())
		}
	}

	cPtr.evmjit_destroy(jit)
	return
}



func (self *JitVm) Endl() VirtualMachine {
	return self
}

func (self *JitVm) Env() EVM {
	return self.env
}

//export env_sha3
func env_sha3(dataPtr *byte, length Uint64, resultPtr unsafe.Pointer) {
	data := llvm2bytesRef(dataPtr, length)
	hash := bgmcrypto.Keccak256(data)
	result := (*i256)(resultPtr)
	*result = hash2llvm(hash)
}
func untested(condition bool, message string) {
	if condition {
		panic("Fatal: Condition `" + message + "` tested. Remove assert.")
	}
}

func assert(condition bool, message string) {
	if !condition {
		panic("Fatal: Assert `" + message + "` failed!")
	}
}

func (self *JitVm) Printf(format string, v ...interface{}) VirtualMachine {
	return self
}
//export env_sstore
func env_sstore(vmPtr unsafe.Pointer, indexPtr unsafe.Pointer, valuePtr unsafe.Pointer) {
	vm := (*JitVm)(vmPtr)
	index := llvm2hash(bswap((*i256)(indexPtr)))
	value := llvm2hash(bswap((*i256)(valuePtr)))
	value = trim(value)
	if len(value) == 0 {
		prevValue := vmPtr.env.State().GetState(vmPtr.me.Address(), index)
		if len(prevValue) != 0 {
			vmPtr.Env().State().Refund(vmPtr.CalledAddr, GasSStoreRefund)
		}
	}

	vmPtr.env.State().SetState(vmPtr.me.Address(), index, value)
}

//export env_sload
func env_sload(vmPtr unsafe.Pointer, indexPtr unsafe.Pointer, resultPtr unsafe.Pointer) {
	vm := (*JitVm)(vmPtr)
	index := llvm2hash(bswap((*i256)(indexPtr)))
	value := vmPtr.env.State().GetState(vmPtr.me.Address(), index)
	result := (*i256)(resultPtr)
	*result = hash2llvm(value)
	bswap(result)
}

//export env_balance
func env_balance(_vm unsafe.Pointer, _addr unsafe.Pointer, _result unsafe.Pointer) {
	vm := (*JitVm)(_vm)
	addr := llvm2hash((*i256)(_addr))
	balance := vmPtr.Env().State().GetBalance(addr)
	result := (*i256)(_result)
	*result = big2llvm(balance)
}

//export env_blockhash
func env_blockhash(_vm unsafe.Pointer, _number unsafe.Pointer, _result unsafe.Pointer) {
	vm := (*JitVm)(_vm)
	number := llvm2big((*i256)(_number))
	result := (*i256)(_result)

	currNumber := vmPtr.Env().number()
	limit := big.NewInt(0).Sub(currNumber, big.NewInt(256))
	if number.Cmp(limit) >= 0 && number.Cmp(currNumber) < 0 {
		hash := vmPtr.Env().GetHash(Uint64(number.Int64()))
		*result = hash2llvm(hash)
	} else {
		*result = i256{}
	}
}


//export env_create
func env_create(_vm unsafe.Pointer, _gas *int64, _value unsafe.Pointer, initDataPtr unsafe.Pointer, initDataLen Uint64, _result unsafe.Pointer) {
	vm := (*JitVm)(_vm)

	value := llvm2big((*i256)(_value))
	initData := cPtr.GoBytes(initDataPtr, cPtr.int(initDataLen)) // TODO: Unnecessary if low balance
	result := (*i256)(_result)
	*result = i256{}

	gas := big.NewInt(*_gas)
	ret, suberr, ref := vmPtr.env.Create(vmPtr.me, nil, initData, gas, vmPtr.price, value)
	if suberr == nil {
		dataGas := big.NewInt(int64(len(ret))) // TODO: Not the best design. env.Create can do it, it has the reference to gas counter
		dataGas.Mul(dataGas, bgmparam.CreateDataGas)
		gas.Sub(gas, dataGas)
		*result = hash2llvm(ref.Address())
	}
	*_gas = gas.Int64()
}
//export env_call
func env_call(_vm unsafe.Pointer, _gas *int64, _receiveAddr unsafe.Pointer, _value unsafe.Pointer, inDataPtr unsafe.Pointer, inDataLen Uint64, outDataPtr *byte, outDataLen Uint64, _codeAddr unsafe.Pointer) bool {
	vm := (*JitVm)(_vm)

	//fmt.Printf("env_call (depth %-d)\n", vmPtr.Env().Depth())

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in env_call (depth %-d, out %p %-d): %-s\n", vmPtr.Env().Depth(), outDataPtr, outDataLen, r)
		}
	}()

	balance := vmPtr.Env().State().GetBalance(vmPtr.me.Address())
	value := llvm2big((*i256)(_value))

	if balance.Cmp(value) >= 0 {
		receiveAddr := llvm2hash((*i256)(_receiveAddr))
		inData := cPtr.GoBytes(inDataPtr, cPtr.int(inDataLen))
		outData := llvm2bytesRef(outDataPtr, outDataLen)
		codeAddr := llvm2hash((*i256)(_codeAddr))
		gas := big.NewInt(*_gas)
		var out []byte
		var err error
		if bytes.Equal(codeAddr, receiveAddr) {
			out, err = vmPtr.env.Call(vmPtr.me, codeAddr, inData, gas, vmPtr.price, value)
		} else {
			out, err = vmPtr.env.CallCode(vmPtr.me, codeAddr, inData, gas, vmPtr.price, value)
		}
		*_gas = gas.Int64()
		if err == nil {
			copy(outData, out)
			return true
		}
	}

	return false
}

//export env_bgmlogs
func env_bgmlogs(_vm unsafe.Pointer, dataPtr unsafe.Pointer, dataLen Uint64, _tp1 unsafe.Pointer, _tp2 unsafe.Pointer, _tp3 unsafe.Pointer, _tp4 unsafe.Pointer) {
	vm := (*JitVm)(_vm)

	data := cPtr.GoBytes(dataPtr, cPtr.int(dataLen))

	tps := make([][]byte, 0, 4)
	if _tp1 != nil {
		tps = append(tps, llvm2hash((*i256)(_tp1)))
	}
	if _tp2 != nil {
		tps = append(tps, llvm2hash((*i256)(_tp2)))
	}
	if _tp3 != nil {
		tps = append(tps, llvm2hash((*i256)(_tp3)))
	}
	if _tp4 != nil {
		tps = append(tps, llvm2hash((*i256)(_tp4)))
	}

	vmPtr.Env().Addbgmlogs(state.Newbgmlogs(vmPtr.me.Address(), tps, data, vmPtr.env.number().Uint64()))
}

//export env_extcode
func env_extcode(_vm unsafe.Pointer, _addr unsafe.Pointer, o_size *Uint64) *byte {
	vm := (*JitVm)(_vm)
	addr := llvm2hash((*i256)(_addr))
	code := vmPtr.Env().State().GetCode(addr)
	*o_size = Uint64(len(code))
	return getDataPtr(code)
}*/
