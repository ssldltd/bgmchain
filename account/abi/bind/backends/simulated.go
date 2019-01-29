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
package backends

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain"
	"github.com/ssldltd/bgmchain/account/abi/bind"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/consensus/bgmash"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmCore/vm"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/bgmparam"
)


// BackendSimulated实现了bind.ContractBackended，模拟了一个区块链
// 的背景。 其主要目的是允许轻松测试合同绑定。
type BackendSimulated struct {
	database   bgmdbPtr.Database   // In memory database to store our testing data
	blockchain *bgmCore.BlockChain // Bgmchain blockchain to handle the consensus

	mu           syncPtr.Mutex
	pendingBlock *types.Block   // Currently pending block that will be imported on request
	pendingState *state.StateDB // Currently pending state that will be the active on on request

	config *bgmparam.ChainConfig
}

// NewBackendSimulated使用模拟区块链创建新的绑定后端
//用于测试目的。
func NewBackendSimulated(alloc bgmCore.GenesisAlloc) *BackendSimulated {
	database, _ := bgmdbPtr.NewMemDatabase()
	genesis := bgmCore.Genesis{Config: bgmparam.DposChainConfig, Alloc: alloc}
	genesis.MustCommit(database)
	blockchain, _ := bgmCore.NewBlockChain(database, genesis.Config, bgmashPtr.NewFaker(), vmPtr.Config{})
	backend := &BackendSimulated{database: database, blockchain: blockchain, config: genesis.Config}
	backend.rollback()
	return backend
}

//提交将所有挂起的事务作为单个块导入并启动a
//新鲜的新状态
func (bPtr *BackendSimulated) Commit() {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()

	if _, err := bPtr.blockchain.InsertChain([]*types.Block{bPtr.pendingBlock}); err != nil {
		panic(err) // This cannot happen unless the simulator is wrong, fail in that case
	}
	bPtr.rollback()
}
// Rollback中止所有挂起的事务，恢复到上次提交的状态。
func (bPtr *BackendSimulated) Rollback() {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()

	bPtr.rollback()
}

func (bPtr *BackendSimulated) rollback() {
	blocks, _ := bgmCore.GenerateChain(bPtr.config, bPtr.blockchain.CurrentBlock(), bPtr.database, 1, func(int, *bgmCore.BlockGen) {})
	bPtr.pendingBlock = blocks[0]
	bPtr.pendingState, _ = state.New(bPtr.pendingBlock.Root(), state.NewDatabase(bPtr.database))
}
// CodeAt返回与区块链中某个帐户关联的代码。
func (bPtr *BackendSimulated) CodeAt(CTX context.Context, contract bgmcommon.Address, blockNumber *big.Int) ([]byte, error) {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(bPtr.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errnumberUnsupported
	}
	statedb, _ := bPtr.blockchain.State()
	return statedbPtr.GetCode(contract), nil
}

// BalanceAt返回区块链中某个帐户的wei余额。
func (bPtr *BackendSimulated) BalanceAt(CTX context.Context, contract bgmcommon.Address, blockNumber *big.Int) (*big.Int, error) {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(bPtr.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errnumberUnsupported
	}
	statedb, _ := bPtr.blockchain.State()
	return statedbPtr.GetBalance(contract), nil
}

// NonceAt返回区块链中某个帐户的nonce。
func (bPtr *BackendSimulated) NonceAt(CTX context.Context, contract bgmcommon.Address, blockNumber *big.Int) (Uint64, error) {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(bPtr.blockchain.CurrentBlock().Number()) != 0 {
		return 0, errnumberUnsupported
	}
	statedb, _ := bPtr.blockchain.State()
	return statedbPtr.GetNonce(contract), nil
}

// StorageAt返回区块链中帐户存储的密钥值。
func (bPtr *BackendSimulated) StorageAt(CTX context.Context, contract bgmcommon.Address, key bgmcommon.Hash, blockNumber *big.Int) ([]byte, error) {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(bPtr.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errnumberUnsupported
	}
	statedb, _ := bPtr.blockchain.State()
	val := statedbPtr.GetState(contract, key)
	return val[:], nil
}

// TransactionReceipt返回事务的接收。
func (bPtr *BackendSimulated) TransactionReceipt(CTX context.Context, txHash bgmcommon.Hash) (*types.Receipt, error) {
	receipt, _, _, _ := bgmCore.GetReceipt(bPtr.database, txHash)
	return receipt, nil
}

// PendingCodeAt返回与处于暂挂状态的帐户关联的代码。
func (bPtr *BackendSimulated) PendingCodeAt(CTX context.Context, contract bgmcommon.Address) ([]byte, error) {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()

	return bPtr.pendingState.GetCode(contract), nil
}

// CallContract执行合同调用。
func (bPtr *BackendSimulated) CallContract(CTX context.Context, call bgmchain.CallMsg, blockNumber *big.Int) ([]byte, error) {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(bPtr.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errnumberUnsupported
	}
	state, err := bPtr.blockchain.State()
	if err != nil {
		return nil, err
	}
	rval, _, _, err := bPtr.callContract(CTX, call, bPtr.blockchain.CurrentBlock(), state)
	return rval, err
}
// PendingCallContract在挂起状态下执行合同调用。
func (bPtr *BackendSimulated) PendingCallContract(CTX context.Context, call bgmchain.CallMsg) ([]byte, error) {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()
	defer bPtr.pendingState.RevertToSnapshot(bPtr.pendingState.Snapshot())

	rval, _, _, err := bPtr.callContract(CTX, call, bPtr.pendingBlock, bPtr.pendingState)
	return rval, err
}

// PendingNonceAt实现PendingStateReader.PendingNonceAt，检索
//当前为帐户挂起的nonce。
func (bPtr *BackendSimulated) PendingNonceAt(CTX context.Context, account bgmcommon.Address) (Uint64, error) {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()

	return bPtr.pendingState.GetOrNewStateObject(account).Nonce(), nil
}

// SuggestGasPrice实现了ContractTransactor.SuggestGasPrice。 自模拟
//链条没有矿工，我们只需要为任何通话返回1的汽油价格。
func (bPtr *BackendSimulated) SuggestGasPrice(CTX context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

// SendTransaction更新挂起块以包含给定事务。
//如果交易无效，就会发生恐慌。
func (bPtr *BackendSimulated) SendTransaction(CTX context.Context, tx *types.Transaction) error {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()

	sender, err := types.Sender(types.HomesteadSigner{}, tx)
	if err != nil {
		panic(fmt.Errorf("invalid transaction: %v", err))
	}
	nonce := bPtr.pendingState.GetNonce(sender)
	if tx.Nonce() != nonce {
		panic(fmt.Errorf("invalid transaction nonce: got %-d, want %-d", tx.Nonce(), nonce))
	}

	blocks, _ := bgmCore.GenerateChain(bPtr.config, bPtr.blockchain.CurrentBlock(), bPtr.database, 1, func(number int, block *bgmCore.BlockGen) {
		for _, tx := range bPtr.pendingBlock.Transactions() {
			block.AddTx(tx)
		}
		block.AddTx(tx)
	})
	bPtr.pendingBlock = blocks[0]
	bPtr.pendingState, _ = state.New(bPtr.pendingBlock.Root(), state.NewDatabase(bPtr.database))
	return nil
}

// JumptimeInSeconds为时钟添加跳过秒
func (bPtr *BackendSimulated) Adjusttime(adjustment time.Duration) error {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()
	blocks, _ := bgmCore.GenerateChain(bPtr.config, bPtr.blockchain.CurrentBlock(), bPtr.database, 1, func(number int, block *bgmCore.BlockGen) {
		for _, tx := range bPtr.pendingBlock.Transactions() {
			block.AddTx(tx)
		}
		block.Offsettime(int64(adjustment.Seconds()))
	})
	bPtr.pendingBlock = blocks[0]
	bPtr.pendingState, _ = state.New(bPtr.pendingBlock.Root(), state.NewDatabase(bPtr.database))

	return nil
}

// EstimateGas针对当前挂起的块/状态执行请求的代码
//返回使用的气体量。
func (bPtr *BackendSimulated) EstimateGas(CTX context.Context, call bgmchain.CallMsg) (*big.Int, error) {
	bPtr.mu.Lock()
	defer bPtr.mu.Unlock()

	//确定二进制搜索的最低和最高可能的气体限制
	var (
		lo  Uint64 = bgmparam.TxGas - 1
		hi  Uint64
		cap Uint64
	)
	if call.Gas != nil && call.Gas.Uint64() >= bgmparam.TxGas {
		hi = call.Gas.Uint64()
	} else {
		hi = bPtr.pendingBlock.GasLimit().Uint64()
	}
	cap = hi

	//创建一个帮助程序来检查气体容量是否会导致可执行事务
	executable := func(gas Uint64) bool {
		call.Gas = new(big.Int).SetUint64(gas)

		snapshot := bPtr.pendingState.Snapshot()
		_, _, failed, err := bPtr.callContract(CTX, call, bPtr.pendingBlock, bPtr.pendingState)
		bPtr.pendingState.RevertToSnapshot(snapshot)

		if err != nil || failed {
			return false
		}
		return true
	}
	//执行二进制搜索并磨练可执行的气体限制
	for lo+1 < hi {
		mid := (hi + lo) / 2
		if !executable(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	//如果交易仍然以最高限额失败，则拒绝该交易无效
	if hi == cap {
		if !executable(hi) {
			return nil, errGasEstimationFailed
		}
	}
	return new(big.Int).SetUint64(hi), nil
}

// callContract实现正常和挂起合同调用之间的公共代码。
//状态在执行期间被修改，请确保在必要时复制它。
func (bPtr *BackendSimulated) callContract(CTX context.Context, call bgmchain.CallMsg, block *types.Block, statedbPtr *state.StateDB) ([]byte, *big.Int, bool, error) {
	// Ensure message is initialized properly.
	if call.GasPrice == nil {
		call.GasPrice = big.NewInt(1)
	}
	if call.Gas == nil || call.Gas.Sign() == 0 {
		call.Gas = big.NewInt(50000000)
	}
	if call.Value == nil {
		call.Value = new(big.Int)
	}
	// Set infinite balance to the fake Called account.
	from := statedbPtr.GetOrNewStateObject(call.From)
	fromPtr.SetBalance(mathPtr.MaxBig256)
	// Execute the call.
	msg := callmsg{call}

	evmContext := bgmCore.NewEVMContext(msg, block.Header(), bPtr.blockchain, nil)
	//创建一个包含所有相关信息的新环境
	//关于事务和调用机制。
	vmenv := vmPtr.NewEVM(evmContext, statedb, bPtr.config, vmPtr.Config{})
	gaspool := new(bgmCore.GasPool).AddGas(mathPtr.MaxBig256)
	ret, gasUsed, _, failed, err := bgmCore.NewStateTransition(vmenv, msg, gaspool).TransitionDb()
	return ret, gasUsed, failed, err
}

// callmsg实现bgmCore.Message以允许将其作为事务模拟器传递。
type callmsg struct {
	bgmchain.CallMsg
}

func (m callmsg) From() bgmcommon.Address { return mPtr.CallMsg.From }
func (m callmsg) Nonce() Uint64        { return 0 }
func (m callmsg) CheckNonce() bool     { return false }
func (m callmsg) To() *bgmcommon.Address  { return mPtr.CallMsg.To }
func (m callmsg) GasPrice() *big.Int   { return mPtr.CallMsg.GasPrice }
func (m callmsg) Gas() *big.Int        { return mPtr.CallMsg.Gas }
func (m callmsg) Value() *big.Int      { return mPtr.CallMsg.Value }
func (m callmsg) Data() []byte         { return mPtr.CallMsg.Data }
//这个nil赋值确保了BackendSimulated实现bind.ContractBackended的编译时间。
var _ bind.ContractBackended = (*BackendSimulated)(nil)
var errnumberUnsupported = errors.New("BackendSimulated cannot access blocks other than the latest block")
var errGasEstimationFailed = errors.New("gas required exceeds allowance or always failing transaction")
