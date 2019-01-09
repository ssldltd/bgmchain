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
package bind

import (
	"context"
	"errors"
	"math/big"

	"github.com/ssldltd/bgmchain"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
)

var (
	ErrorNoCode = errors.New("no contract code at given address")
	ErrorNoPendingState = errors.New("backend does not support pending state")
	ErrorNoCodeAfterDeploy = errors.New("no contract code after deployment")
)

// ContractCalled定义允许在读取时使用契约操作所需的方法
//仅限基础。
type ContractCalled interface {
	// CodeAt返回给定帐户的代码。 这需要区分
	//合同内部错误和本地链之间不同步。
	CodeAt(ctx context.Context, contract bgmcommon.Address, blockNumber *big.Int) ([]byte, error)
	// ContractCall使用指定的数据执行Bgmchain合约调用
	//输入
	CallContract(ctx context.Context, call bgmchain.CallMsg, blockNumber *big.Int) ([]byte, error)
}

// DeployedBackend包装WaittingMined和WaittingDeployed所需的操作。
type DeployedBackend interface {
	TransactionReceipt(ctx context.Context, txHash bgmcommon.Hash) (*types.Receipt, error)
	CodeAt(ctx context.Context, account bgmcommon.Address, blockNumber *big.Int) ([]byte, error)
}

// PendingContractCalled定义了在挂起状态下执行合同调用的方法。
//当请求访问待处理状态时，Call将尝试发现此接口。
//如果后端不支持挂起状态，则Call返回ErrorNoPendingState。
type PendingContractCalled interface {
	// PendingCodeAt返回处于暂挂状态的给定帐户的代码。
	PendingCodeAt(ctx context.Context, contract bgmcommon.Address) ([]byte, error)
	// PendingCallContract针对挂起状态执行Bgmchain合约调用。
	PendingCallContract(ctx context.Context, call bgmchain.CallMsg) ([]byte, error)
}

// ContractTransactor定义允许合同操作所需的方法
//仅限写作。 除了交易方法，其余的都是帮手
//当用户没有提供某些需要的值时使用，而是将其保留
//到交易者来决定。
type ContractTransactor interface {
	// PendingCodeAt返回处于暂挂状态的给定帐户的代码。
	PendingCodeAt(ctx context.Context, account bgmcommon.Address) ([]byte, error)
	// PendingNonceAt检索与帐户关联的当前待处理的nonce。
	PendingNonceAt(ctx context.Context, account bgmcommon.Address) (uint64, error)
	// SuggestGasPrice检索当前建议的汽油价格以及时
	//执行交易。
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	// EstimateGas尝试估算执行特定所需的气体
	//基于后端区块链当前挂起状态的事务。
	//无法保证这是其他的真正的气体限制要求
	//矿工可以添加或删除交易，但它应该提供基础
	//用于设置合理的默认值。
	EstimateGas(ctx context.Context, call bgmchain.CallMsg) (usedGas *big.Int, err error)
	// SendTransaction将事务注入待处理池以供执行。
	SendTransaction(ctx context.Context, tx *types.Transaction) error
}
// ContractBackended定义了以读写方式处理合同所需的方法。
type ContractBackended interface {
	ContractCalled
	ContractTransactor
}