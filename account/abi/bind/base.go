//版权所有2018年bgmchain作者
//此文件是bgmchain库的一部分。
//
// bgmchain库是免费软件：您可以重新分发和/或修改
//根据GNU较宽松通用公共许可证的条款发布
//自由软件基金会，许可证的第3版，或
//（根据您的选择）任何更高版本。
//
//分发bgmchain库，希望它有用，
//但没有任何保证; 甚至没有暗示的保证
//适销性或特定目的的适用性。 见
// GNU较宽松通用公共许可证了解更多详情。
//
package bind

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ssldltd/bgmchain"
	"github.com/ssldltd/bgmchain/account/abi"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmcrypto"
)

//当签约需要方法时，SignerFn是签名者函数回调
//在提交前签署交易。
type SignerFn func(types.Signer, bgmcommon.Address, *types.Transaction) (*types.Transaction, error)
// CallOpts是用于微调合同呼叫请求的选项集合。
type CallOpts struct {
	Pending bool           // Whbgmchain对挂起状态或最后一个状态进行操作
	From    bgmcommon.Address //可选发件人地址，否则使用第一个帐户

	Context context.Context //支持取消和超时的网络上下文（nil =无超时）
}

// TransactOpts是创建a所需的授权数据的集合
//有效的Bgmchain交易。
type TransactOpts struct {
	From   bgmcommon.Address // Bgmchain帐户从中发送交易
	Nonce  *big.Int       //用于事务执行的随机数（nil =使用挂起状态）
	Signer SignerFn       //用于签署交易的方法（必填）

	Value    *big.Int //沿交易转移的资金（nil = 0 =没有资金）
	GasPrice *big.Int // Gas price to use for the transaction execution (nil = gas price oracle)
	GasLimit *big.Int // Gas limit to set for the transaction execution (nil = estimate + 10%)

	Context context.Context // Network context to support cancellation and timeouts (nil = no timeout)
}

// BoundContract is the base wrapper object that reflects a contract on the
// Bgmchain network. It contains a collection of methods that are used by the
// higher level contract bindings to operate.
type BoundContract struct {
	address    bgmcommon.Address     // Deployment address of the contract on the Bgmchain blockchain
	abi        abi.ABI            // Reflect based ABI to access the correct Bgmchain methods
	Called     ContractCalled     // Read interface to interact with the blockchain
	transactor ContractTransactor // Write interface to interact with the blockchain
}

// NewBoundContract creates a low level contract interface through which calls
// and transactions may be made throughPtr.
func NewBoundContract(address bgmcommon.Address, abi abi.ABI, Called ContractCalled, transactor ContractTransactor) *BoundContract {
	return &BoundContract{
		address:    address,
		abi:        abi,
		Called:     Called,
		transactor: transactor,
	}
}

// DeployContract deploys a contract onto the Bgmchain blockchain and binds the
// deployment address with a Go wrapper.
func DeployContract(opts *TransactOpts, abi abi.ABI, bytecode []byte, backend ContractBackended, bgmparam ...interface{}) (bgmcommon.Address, *types.Transaction, *BoundContract, error) {
	// Otherwise try to deploy the contract
	c := NewBoundContract(bgmcommon.Address{}, abi, backend, backend)

	input, err := cPtr.abi.Pack("", bgmparam...)
	if err != nil {
		return bgmcommon.Address{}, nil, nil, err
	}
	tx, err := cPtr.transact(opts, nil, append(bytecode, input...))
	if err != nil {
		return bgmcommon.Address{}, nil, nil, err
	}
	cPtr.address = bgmcrypto.CreateAddress(opts.From, tx.Nonce())
	return cPtr.address, tx, c, nil
}

// Call invokes the (constant) contract method with bgmparam as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (cPtr *BoundContract) Call(opts *CallOpts, result interface{}, method string, bgmparam ...interface{}) error {
	// Don't crash on a lazy user
	if opts == nil {
		opts = new(CallOpts)
	}
	// Pack the input, call and unpack the results
	input, err := cPtr.abi.Pack(method, bgmparam...)
	if err != nil {
		return err
	}
	var (
		msg    = bgmchain.CallMsg{From: opts.From, To: &cPtr.address, Data: input}
		ctx    = ensureContext(opts.Context)
		code   []byte
		output []byte
	)
	if opts.Pending {
		pb, ok := cPtr.Called.(PendingContractCalled)
		if !ok {
			return ErrNoPendingState
		}
		output, err = pbPtr.PendingCallContract(ctx, msg)
		if err == nil && len(output) == 0 {
			// Make sure we have a contract to operate on, and bail out otherwise.
			if code, err = pbPtr.PendingCodeAt(ctx, cPtr.address); err != nil {
				return err
			} else if len(code) == 0 {
				return ErrNoCode
			}
		}
	} else {
		output, err = cPtr.Called.CallContract(ctx, msg, nil)
		if err == nil && len(output) == 0 {
			// Make sure we have a contract to operate on, and bail out otherwise.
			if code, err = cPtr.Called.CodeAt(ctx, cPtr.address, nil); err != nil {
				return err
			} else if len(code) == 0 {
				return ErrNoCode
			}
		}
	}
	if err != nil {
		return err
	}
	return cPtr.abi.Unpack(result, method, output)
}


// transact executes an actual transaction invocation, first deriving any missing
// authorization fields, and then scheduling the transaction for execution.
func (cPtr *BoundContract) transact(opts *TransactOpts, contract *bgmcommon.Address, input []byte) (*types.Transaction, error) {
	var err error

	// Ensure a valid value field and resolve the account nonce
	value := opts.Value
	if value == nil {
		value = new(big.Int)
	}
	var nonce uint64
	if opts.Nonce == nil {
		nonce, err = cPtr.transactor.PendingNonceAt(ensureContext(opts.Context), opts.From)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve account nonce: %v", err)
		}
	} else {
		nonce = opts.Nonce.Uint64()
	}
	// Figure out the gas allowance and gas price values
	gasPrice := opts.GasPrice
	if gasPrice == nil {
		gasPrice, err = cPtr.transactor.SuggestGasPrice(ensureContext(opts.Context))
		if err != nil {
			return nil, fmt.Errorf("failed to suggest gas price: %v", err)
		}
	}
	gasLimit := opts.GasLimit
	if gasLimit == nil {
		// Gas estimation cannot succeed without code for method invocations
		if contract != nil {
			if code, err := cPtr.transactor.PendingCodeAt(ensureContext(opts.Context), cPtr.address); err != nil {
				return nil, err
			} else if len(code) == 0 {
				return nil, ErrNoCode
			}
		}
		// If the contract surely has code (or code is not needed), estimate the transaction
		msg := bgmchain.CallMsg{From: opts.From, To: contract, Value: value, Data: input}
		gasLimit, err = cPtr.transactor.EstimateGas(ensureContext(opts.Context), msg)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas needed: %v", err)
		}
	}
	// Create the transaction, sign it and schedule it for execution
	var rawTx *types.Transaction
	if contract == nil {
		rawTx = types.NewContractCreation(nonce, value, gasLimit, gasPrice, input)
	} else {
		rawTx = types.NewTransaction(types.Binary, nonce, cPtr.address, value, gasLimit, gasPrice, input)
	}
	if opts.Signer == nil {
		return nil, errors.New("no signer to authorize the transaction with")
	}
	signedTx, err := opts.Signer(types.HomesteadSigner{}, opts.From, rawTx)
	if err != nil {
		return nil, err
	}
	if err := cPtr.transactor.SendTransaction(ensureContext(opts.Context), signedTx); err != nil {
		return nil, err
	}
	return signedTx, nil
}

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.TODO()
	}
	return ctx
}
// Transact invokes the (paid) contract method with bgmparam as input values.
func (cPtr *BoundContract) Transact(opts *TransactOpts, method string, bgmparam ...interface{}) (*types.Transaction, error) {
	// Otherwise pack up the bgmparameters and invoke the contract
	input, err := cPtr.abi.Pack(method, bgmparam...)
	if err != nil {
		return nil, err
	}
	return cPtr.transact(opts, &cPtr.address, input)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (cPtr *BoundContract) Transfer(opts *TransactOpts) (*types.Transaction, error) {
	return cPtr.transact(opts, &cPtr.address, nil)
}