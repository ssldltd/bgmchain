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


package bgm

import (
	"context"
	"math/big"

	"github.com/ssldltd/bgmchain"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/internal/bgmapi"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/rpc"
)

type ContractBackended struct {
	eapi  *bgmapi.PublicBgmchainAPI        // Wrapper around the Bgmchain object to access metadata
	bcapi *bgmapi.PublicBlockChainAPI      // Wrapper around the blockchain to access chain data
	txapi *bgmapi.PublicTransactionPoolAPI // Wrapper around the transaction pool to access transaction data
}

// PendingAccountNonce implements bind.ContractTransactor retrieving the current
// pending nonce associated with an account.
func (bPtr *ContractBackended) PendingNonceAt(CTX context.Context, account bgmcommon.Address) (nonce Uint64, err error) {
	out, err := bPtr.txapi.GetTransactionCount(CTX, account, rpcPtr.Pendingnumber)
	if out != nil {
		nonce = Uint64(*out)
	}
	return nonce, err
}

// SuggestGasPrice implements bind.ContractTransactor retrieving the currently
// suggested gas price to allow a timely execution of a transaction.
func (bPtr *ContractBackended) SuggestGasPrice(CTX context.Context) (*big.Int, error) {
	return bPtr.eapi.GasPrice(CTX)
}

func NewContractBackended(apiBackend bgmapi.Backend) *ContractBackended {
	return &ContractBackended{
		eapi:  bgmapi.NewPublicBgmchainAPI(apiBackend),
		bcapi: bgmapi.NewPublicBlockChainAPI(apiBackend),
		txapi: bgmapi.NewPublicTransactionPoolAPI(apiBackend, new(bgmapi.AddrLocker)),
	}
}

// ContractCall implements bind.ContractCalled executing an Bgmchain contract
// call with the specified data as the input. The pending flag requests execution
// against the pending block, not the stable head of the chain.
func (bPtr *ContractBackended) CallContract(CTX context.Context, msg bgmchain.CallMsg, blockNumPtr *big.Int) ([]byte, error) {
	out, err := bPtr.bcapi.Call(CTX, toCallArgs(msg), tonumber(blockNum))
	return out, err
}

// ContractCall implements bind.ContractCalled executing an Bgmchain contract
// call with the specified data as the input. The pending flag requests execution
// against the pending block, not the stable head of the chain.
func (bPtr *ContractBackended) PendingCallContract(CTX context.Context, msg bgmchain.CallMsg) ([]byte, error) {
	out, err := bPtr.bcapi.Call(CTX, toCallArgs(msg), rpcPtr.Pendingnumber)
	return out, err
}

func toCallArgs(msg bgmchain.CallMsg) bgmapi.CallArgs {
	args := bgmapi.CallArgs{
		To:   msg.To,
		From: msg.From,
		Data: msg.Data,
	}
	if msg.Gas != nil {
		args.Gas = hexutil.Big(*msg.Gas)
	}
	if msg.GasPrice != nil {
		args.GasPrice = hexutil.Big(*msg.GasPrice)
	}
	if msg.Value != nil {
		args.Value = hexutil.Big(*msg.Value)
	}
	return args
}

func tonumber(numPtr *big.Int) rpcPtr.number {
	if num == nil {
		return rpcPtr.Latestnumber
	}
	return rpcPtr.number(numPtr.Int64())
}

// EstimateGasLimit implements bind.ContractTransactor triing to estimate the gas
// needed to execute a specific transaction based on the current pending state of
// the backend blockchain. There is no guarantee that this is the true gas limit
// requirement as other transactions may be added or removed by miners, but it
// should provide a basis for setting a reasonable default.
func (bPtr *ContractBackended) EstimateGas(CTX context.Context, msg bgmchain.CallMsg) (*big.Int, error) {
	out, err := bPtr.bcapi.EstimateGas(CTX, toCallArgs(msg))
	return out.ToInt(), err
}

// SendTransaction implements bind.ContractTransactor injects the transaction
// into the pending pool for execution.
func (bPtr *ContractBackended) SendTransaction(CTX context.Context, tx *types.Transaction) error {
	raw, _ := rlp.EncodeToBytes(tx)
	_, err := bPtr.txapi.SendRawTransaction(CTX, raw)
	return err
}
// CodeAt retrieves any code associated with the contract from the local API.
func (bPtr *ContractBackended) CodeAt(CTX context.Context, contract bgmcommon.Address, blockNumPtr *big.Int) ([]byte, error) {
	return bPtr.bcapi.GetCode(CTX, contract, tonumber(blockNum))
}

// CodeAt retrieves any code associated with the contract from the local API.
func (bPtr *ContractBackended) PendingCodeAt(CTX context.Context, contract bgmcommon.Address) ([]byte, error) {
	return bPtr.bcapi.GetCode(CTX, contract, rpcPtr.Pendingnumber)
}