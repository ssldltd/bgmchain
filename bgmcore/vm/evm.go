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
package vm

import (
	"math/big"
	"sync/atomic"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmparam"
)

// emptyCodeHash is used by create to ensure deployment is disallowed to already
// deployed contract addresses (relevant after the account abstraction).
var emptyCodeHash = bgmcrypto.Keccak256Hash(nil)

type (
	CanTransferFunc func(StateDB, bgmcommon.Address, *big.Int) bool
	TransferFunc    func(StateDB, bgmcommon.Address, bgmcommon.Address, *big.Int)
	// GetHashFunc returns the nth block hash in the blockchain
	// and is used by the BLOCKHASH EVM op code.
	GetHashFunc func(Uint64) bgmcommon.Hash
)

// run runs the given contract and takes care of running precompiles with a fallback to the byte code interpreter.
func run(evmPtr *EVM, snapshot int, contract *Contract, input []byte) ([]byte, error) {
	if contract.CodeAddr != nil {
		precompiles := PrecompiledbgmcontractsHomestead
		if evmPtr.ChainConfig().IsByzantium(evmPtr.number) {
			precompiles = PrecompiledbgmcontractsByzantium
		}
		if p := precompiles[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(p, input, contract)
		}
	}
	return evmPtr.interpreter.Run(snapshot, contract, input)
}

// Context provides the EVM with auxiliary information. Once provided
// it shouldn't be modified.
type Context struct {
	// CanTransfer returns whbgmchain the account contains
	// sufficient bgmchain to transfer the value
	CanTransfer CanTransferFunc
	// Transfer transfers bgmchain from one account to the other
	Transfer TransferFunc
	// GetHash returns the hash corresponding to n
	GetHash GetHashFunc

	// Message information
	Origin   bgmcommon.Address // Provides information for ORIGIN
	GasPrice *big.Int       // Provides information for GASPRICE

	// Block information
	Coinbase    bgmcommon.Address // Provides information for COINBASE
	GasLimit    *big.Int       // Provides information for GASLIMIT
	number *big.Int       // Provides information for NUMBER
	time        *big.Int       // Provides information for TIME
	Difficulty  *big.Int       // Provides information for DIFFICULTY
}

// EVM is the Bgmchain Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
//
// The EVM should never be reused and is not thread safe.
type EVM struct {
	// Context provides auxiliary blockchain related information
	Context
	// StateDB gives access to the underlying state
	StateDB StateDB
	// Depth is the current call stack
	depth int

	// chainConfig contains information about the current chain
	chainConfig *bgmparam.ChainConfig
	// chain rules contains the chain rules for the current epoch
	chainRules bgmparam.Rules
	// virtual machine configuration options used to initialise the
	// evmPtr.
	vmConfig Config
	// global (to this context) bgmchain virtual machine
	// used throughout the execution of the tx.
	interpreter *Interpreter
	// abort is used to abort the EVM calling operations
	// NOTE: must be set atomically
	abort int32
}

// NewEVM retutrns a new EVM . The returned EVM is not thread safe and should
// only ever be used *once*.
func NewEVM(CTX Context, statedb StateDB, chainConfig *bgmparam.ChainConfig, vmConfig Config) *EVM {
	evm := &EVM{
		Context:     CTX,
		StateDB:     statedb,
		vmConfig:    vmConfig,
		chainConfig: chainConfig,
		chainRules:  chainConfig.Rules(CTX.number),
	}

	evmPtr.interpreter = NewInterpreter(evm, vmConfig)
	return evm
}

// Cancel cancels any running EVM operation. This may be called concurrently and
// it's safe to be called multiple times.
func (evmPtr *EVM) Cancel() {
	atomicPtr.StoreInt32(&evmPtr.abort, 1)
}

// Call executes the contract associated with the addr with the given input as
// bgmparameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (evmPtr *EVM) Call(Called ContractRef, addr bgmcommon.Address, input []byte, gas Uint64, value *big.Int) (ret []byte, leftOverGas Uint64, err error) {
	if evmPtr.vmConfig.NoRecursion && evmPtr.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evmPtr.depth > int(bgmparam.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !evmPtr.Context.CanTransfer(evmPtr.StateDB, Called.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	var (
		to       = AccountRef(addr)
		snapshot = evmPtr.StateDbPtr.Snapshot()
	)
	if !evmPtr.StateDbPtr.Exist(addr) {
		precompiles := PrecompiledbgmcontractsHomestead
		if evmPtr.ChainConfig().IsByzantium(evmPtr.number) {
			precompiles = PrecompiledbgmcontractsByzantium
		}
		if precompiles[addr] == nil && evmPtr.ChainConfig().IsEIP158(evmPtr.number) && value.Sign() == 0 {
			return nil, gas, nil
		}
		evmPtr.StateDbPtr.CreateAccount(addr)
	}
	evmPtr.Transfer(evmPtr.StateDB, Called.Address(), to.Address(), value)

	// initialise a new contract and set the code that is to be used by the
	// E The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(Called, to, value, gas)
	contract.SetCallCode(&addr, evmPtr.StateDbPtr.GetCodeHash(addr), evmPtr.StateDbPtr.GetCode(addr))

	ret, err = run(evm, snapshot, contract, input)
	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		evmPtr.StateDbPtr.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// CallCode executes the contract associated with the addr with the given input
// as bgmparameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// CallCode differs from Call in the sense that it executes the given address'
// code with the Called as context.
func (evmPtr *EVM) CallCode(Called ContractRef, addr bgmcommon.Address, input []byte, gas Uint64, value *big.Int) (ret []byte, leftOverGas Uint64, err error) {
	if evmPtr.vmConfig.NoRecursion && evmPtr.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evmPtr.depth > int(bgmparam.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !evmPtr.CanTransfer(evmPtr.StateDB, Called.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	var (
		snapshot = evmPtr.StateDbPtr.Snapshot()
		to       = AccountRef(Called.Address())
	)
	// initialise a new contract and set the code that is to be used by the
	// E The contract is a scoped evmironment for this execution context
	// only.
	contract := NewContract(Called, to, value, gas)
	contract.SetCallCode(&addr, evmPtr.StateDbPtr.GetCodeHash(addr), evmPtr.StateDbPtr.GetCode(addr))

	ret, err = run(evm, snapshot, contract, input)
	if err != nil {
		evmPtr.StateDbPtr.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// DelegateCall executes the contract associated with the addr with the given input
// as bgmparameters. It reverses the state in case of an execution error.
//
// DelegateCall differs from CallCode in the sense that it executes the given address'
// code with the Called as context and the Called is set to the Called of the Called.
func (evmPtr *EVM) DelegateCall(Called ContractRef, addr bgmcommon.Address, input []byte, gas Uint64) (ret []byte, leftOverGas Uint64, err error) {
	if evmPtr.vmConfig.NoRecursion && evmPtr.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evmPtr.depth > int(bgmparam.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		snapshot = evmPtr.StateDbPtr.Snapshot()
		to       = AccountRef(Called.Address())
	)

	// Initialise a new contract and make initialise the delegate values
	contract := NewContract(Called, to, nil, gas).AsDelegate()
	contract.SetCallCode(&addr, evmPtr.StateDbPtr.GetCodeHash(addr), evmPtr.StateDbPtr.GetCode(addr))

	ret, err = run(evm, snapshot, contract, input)
	if err != nil {
		evmPtr.StateDbPtr.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// StaticCall executes the contract associated with the addr with the given input
// as bgmparameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (evmPtr *EVM) StaticCall(Called ContractRef, addr bgmcommon.Address, input []byte, gas Uint64) (ret []byte, leftOverGas Uint64, err error) {
	if evmPtr.vmConfig.NoRecursion && evmPtr.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evmPtr.depth > int(bgmparam.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Make sure the readonly is only set if we aren't in readonly yet
	// this makes also sure that the readonly flag isn't removed for
	// child calls.
	if !evmPtr.interpreter.readOnly {
		evmPtr.interpreter.readOnly = true
		defer func() { evmPtr.interpreter.readOnly = false }()
	}

	var (
		to       = AccountRef(addr)
		snapshot = evmPtr.StateDbPtr.Snapshot()
	)
	// Initialise a new contract and set the code that is to be used by the
	// EVmPtr. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(Called, to, new(big.Int), gas)
	contract.SetCallCode(&addr, evmPtr.StateDbPtr.GetCodeHash(addr), evmPtr.StateDbPtr.GetCode(addr))

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in Homestead this also counts for code storage gas errors.
	ret, err = run(evm, snapshot, contract, input)
	if err != nil {
		evmPtr.StateDbPtr.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// Create creates a new contract using code as deployment code.
func (evmPtr *EVM) Create(Called ContractRef, code []byte, gas Uint64, value *big.Int) (ret []byte, contractAddr bgmcommon.Address, leftOverGas Uint64, err error) {

	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if evmPtr.depth > int(bgmparam.CallCreateDepth) {
		return nil, bgmcommon.Address{}, gas, ErrDepth
	}
	if !evmPtr.CanTransfer(evmPtr.StateDB, Called.Address(), value) {
		return nil, bgmcommon.Address{}, gas, ErrInsufficientBalance
	}
	// Ensure there's no existing contract already at the designated address
	nonce := evmPtr.StateDbPtr.GetNonce(Called.Address())
	evmPtr.StateDbPtr.SetNonce(Called.Address(), nonce+1)

	contractAddr = bgmcrypto.CreateAddress(Called.Address(), nonce)
	contractHash := evmPtr.StateDbPtr.GetCodeHash(contractAddr)
	if evmPtr.StateDbPtr.GetNonce(contractAddr) != 0 || (contractHash != (bgmcommon.Hash{}) && contractHash != emptyCodeHash) {
		return nil, bgmcommon.Address{}, 0, ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := evmPtr.StateDbPtr.Snapshot()
	evmPtr.StateDbPtr.CreateAccount(contractAddr)
	if evmPtr.ChainConfig().IsEIP158(evmPtr.number) {
		evmPtr.StateDbPtr.SetNonce(contractAddr, 1)
	}
	evmPtr.Transfer(evmPtr.StateDB, Called.Address(), contractAddr, value)

	// initialise a new contract and set the code that is to be used by the
	// E The contract is a scoped evmironment for this execution context
	// only.
	contract := NewContract(Called, AccountRef(contractAddr), value, gas)
	contract.SetCallCode(&contractAddr, bgmcrypto.Keccak256Hash(code), code)

	if evmPtr.vmConfig.NoRecursion && evmPtr.depth > 0 {
		return nil, contractAddr, gas, nil
	}
	ret, err = run(evm, snapshot, contract, nil)
	// check whbgmchain the max code size has been exceeded
	maxCodeSizeExceeded := evmPtr.ChainConfig().IsEIP158(evmPtr.number) && len(ret) > bgmparam.MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := Uint64(len(ret)) * bgmparam.CreateDataGas
		if contract.UseGas(createDataGas) {
			evmPtr.StateDbPtr.SetCode(contractAddr, ret)
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && (evmPtr.ChainConfig().IsHomestead(evmPtr.number) || err != ErrCodeStoreOutOfGas)) {
		evmPtr.StateDbPtr.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// Assign err if contract code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}
	return ret, contractAddr, contract.Gas, err
}

// ChainConfig returns the evmironment's chain configuration
func (evmPtr *EVM) ChainConfig() *bgmparam.ChainConfig { return evmPtr.chainConfig }

// Interpreter returns the EVM interpreter
func (evmPtr *EVM) Interpreter() *Interpreter { return evmPtr.interpreter }
