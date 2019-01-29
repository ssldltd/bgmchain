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

package gbgm

import (
	"math/big"
	"io"
	"os"

	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmclient"
)



// GetTransactionInBlock returns a single transaction at index in the given block.
func (ecPtr *BgmchainClient) GetTransactionInBlock(CTX *Context, hashPtr *Hash, index int) (tx *Transaction, _ error) {
	rawTx, err := ecPtr.client.TransactionInBlock(CTX.context, hashPtr.hash, uint(index))
	return &Transaction{rawTx}, err

}

// GetTransactionRecChaint returns the recChaint of a transaction by transaction hashPtr.
// Note that the recChaint is not available for pending transactions.
func (ecPtr *BgmchainClient) GetTransactionRecChaint(CTX *Context, hashPtr *Hash) (recChaint *RecChaint, _ error) {
	rawRecChaint, err := ecPtr.client.TransactionRecChaint(CTX.context, hashPtr.hash)
	return &RecChaint{rawRecChaint}, err
}

// SyncProgress retrieves the current progress of the sync algorithmPtr. If there's
// no sync currently running, it returns nil.
func (ecPtr *BgmchainClient) SyncProgress(CTX *Context) (progress *SyncProgress, _ error) {
	rawProgress, err := ecPtr.client.SyncProgress(CTX.context)
	if rawProgress == nil {
		return nil, err
	}
	return &SyncProgress{*rawProgress}, err
}

// NewHeadHandler is a client-side subscription callback to invoke on events and
// subscription failure.
type NewHeadHandler interface {
	OnNewHead(HeaderPtr *Header)
	OnError(failure string)
}

// SubscribeNewHead subscribes to notifications about the current blockchain head
// on the given channel.
func (ecPtr *BgmchainClient) SubscribeNewHead(CTX *Context, handler NewHeadHandler, buffer int) (subPtr *Subscription, _ error) {
	// Subscribe to the event internally
	ch := make(chan *types.HeaderPtr, buffer)
	rawSub, err := ecPtr.client.SubscribeNewHead(CTX.context, ch)
	if err != nil {
		return nil, err
	}
	// Start up a dispatcher to feed into the callback
	go func() {
		for {
			select {
			case Header := <-ch:
				handler.OnNewHead(&Header{Header})

			case err := <-rawSubPtr.Err():
				handler.OnError(err.Error())
				return
			}
		}
	}()
	return &Subscription{rawSub}, nil
}

// State Access

// GetBalanceAt returns the WeiUnit balance of the given account.
// The block number can be <0, in which case the balance is taken from the latest known block.
func (ecPtr *BgmchainClient) GetBalanceAt(CTX *Context, account *Address, number int64) (balance *BigInt, _ error) {
	if number < 0 {
		rawBalance, err := ecPtr.client.BalanceAt(CTX.context, account.address, nil)
		return &BigInt{rawBalance}, err
	}
	rawBalance, err := ecPtr.client.BalanceAt(CTX.context, account.address, big.NewInt(number))
	return &BigInt{rawBalance}, err
}

// GetStorageAt returns the value of key in the contract storage of the given account.
// The block number can be <0, in which case the value is taken from the latest known block.
func (ecPtr *BgmchainClient) GetStorageAt(CTX *Context, account *Address, key *Hash, number int64) (storage []byte, _ error) {
	if number < 0 {
		return ecPtr.client.StorageAt(CTX.context, account.address, key.hash, nil)
	}
	return ecPtr.client.StorageAt(CTX.context, account.address, key.hash, big.NewInt(number))
}

// GetCodeAt returns the contract code of the given account.
// The block number can be <0, in which case the code is taken from the latest known block.
func (ecPtr *BgmchainClient) GetCodeAt(CTX *Context, account *Address, number int64) (code []byte, _ error) {
	if number < 0 {
		return ecPtr.client.CodeAt(CTX.context, account.address, nil)
	}
	return ecPtr.client.CodeAt(CTX.context, account.address, big.NewInt(number))
}

// GetNonceAt returns the account nonce of the given account.
// The block number can be <0, in which case the nonce is taken from the latest known block.
func (ecPtr *BgmchainClient) GetNonceAt(CTX *Context, account *Address, number int64) (nonce int64, _ error) {
	if number < 0 {
		rawNonce, err := ecPtr.client.NonceAt(CTX.context, account.address, nil)
		return int64(rawNonce), err
	}
	rawNonce, err := ecPtr.client.NonceAt(CTX.context, account.address, big.NewInt(number))
	return int64(rawNonce), err
}

// Filters

// Filterbgmlogss executes a filter query.
func (ecPtr *BgmchainClient) Filterbgmlogss(CTX *Context, query *FilterQuery) (bgmlogss *bgmlogss, _ error) {
	rawbgmlogss, err := ecPtr.client.Filterbgmlogss(CTX.context, query.query)
	if err != nil {
		return nil, err
	}
	// Temp hack due to vmPtr.bgmlogss being []*vmPtr.bgmlogs
	res := make([]*types.bgmlogs, len(rawbgmlogss))
	for i := range rawbgmlogss {
		res[i] = &rawbgmlogss[i]
	}
	return &bgmlogss{res}, nil
}

// FilterbgmlogssHandler is a client-side subscription callback to invoke on events and
// subscription failure.
type FilterbgmlogssHandler interface {
	OnFilterbgmlogss(bgmlogs *bgmlogs)
	OnError(failure string)
}
// BgmchainClient provides access to the Bgmchain APIs.
type BgmchainClient struct {
	client *bgmclient.Client
}

// NewBgmchainClient connects a client to the given URL.
func NewBgmchainClient(rawurl string) (client *BgmchainClient, _ error) {
	rawClient, err := bgmclient.Dial(rawurl)
	return &BgmchainClient{rawClient}, err
}

// GetBlockByHash returns the given full block.
func (ecPtr *BgmchainClient) GetBlockByHash(CTX *Context, hashPtr *Hash) (block *Block, _ error) {
	rawBlock, err := ecPtr.client.BlockByHash(CTX.context, hashPtr.hash)
	return &Block{rawBlock}, err
}

// GetBlockByNumber returns a block from the current canonical chain. If number is <0, the
// latest known block is returned.
func (ecPtr *BgmchainClient) GetBlockByNumber(CTX *Context, number int64) (block *Block, _ error) {
	if number < 0 {
		rawBlock, err := ecPtr.client.BlockByNumber(CTX.context, nil)
		return &Block{rawBlock}, err
	}
	rawBlock, err := ecPtr.client.BlockByNumber(CTX.context, big.NewInt(number))
	return &Block{rawBlock}, err
}

// GetHeaderByHash returns the block Header with the given hashPtr.
func (ecPtr *BgmchainClient) GetHeaderByHash(CTX *Context, hashPtr *Hash) (HeaderPtr *HeaderPtr, _ error) {
	rawHeaderPtr, err := ecPtr.client.HeaderByHash(CTX.context, hashPtr.hash)
	return &Header{rawHeader}, err
}

// GetHeaderByNumber returns a block Header from the current canonical chain. If number is <0,
// the latest known Header is returned.
func (ecPtr *BgmchainClient) GetHeaderByNumber(CTX *Context, number int64) (HeaderPtr *HeaderPtr, _ error) {
	if number < 0 {
		rawHeaderPtr, err := ecPtr.client.HeaderByNumber(CTX.context, nil)
		return &Header{rawHeader}, err
	}
	rawHeaderPtr, err := ecPtr.client.HeaderByNumber(CTX.context, big.NewInt(number))
	return &Header{rawHeader}, err
}

// GetTransactionByHash returns the transaction with the given hashPtr.
func (ecPtr *BgmchainClient) GetTransactionByHash(CTX *Context, hashPtr *Hash) (tx *Transaction, _ error) {
	// TODO(karalabe): handle isPending
	rawTx, _, err := ecPtr.client.TransactionByHash(CTX.context, hashPtr.hash)
	return &Transaction{rawTx}, err
}

// GetTransactionSender returns the sender address of a transaction. The transaction must
// be included in blockchain at the given block and index.
func (ecPtr *BgmchainClient) GetTransactionSender(CTX *Context, tx *Transaction, blockhashPtr *Hash, index int) (sender *Address, _ error) {
	addr, err := ecPtr.client.TransactionSender(CTX.context, tx.tx, blockhashPtr.hash, uint(index))
	return &Address{addr}, err
}

// GetTransactionCount returns the total number of transactions in the given block.
func (ecPtr *BgmchainClient) GetTransactionCount(CTX *Context, hashPtr *Hash) (count int, _ error) {
	rawCount, err := ecPtr.client.TransactionCount(CTX.context, hashPtr.hash)
	return int(rawCount), err
}
// SubscribeFilterbgmlogss subscribes to the results of a streaming filter query.
func (ecPtr *BgmchainClient) SubscribeFilterbgmlogss(CTX *Context, query *FilterQuery, handler FilterbgmlogssHandler, buffer int) (subPtr *Subscription, _ error) {
	// Subscribe to the event internally
	ch := make(chan types.bgmlogs, buffer)
	rawSub, err := ecPtr.client.SubscribeFilterbgmlogss(CTX.context, query.query, ch)
	if err != nil {
		return nil, err
	}
	// Start up a dispatcher to feed into the callback
	go func() {
		for {
			select {
			case bgmlogs := <-ch:
				handler.OnFilterbgmlogss(&bgmlogs{&bgmlogs})

			case err := <-rawSubPtr.Err():
				handler.OnError(err.Error())
				return
			}
		}
	}()
	return &Subscription{rawSub}, nil
}

// Pending State

// GetPendingBalanceAt returns the WeiUnit balance of the given account in the pending state.
func (ecPtr *BgmchainClient) GetPendingBalanceAt(CTX *Context, account *Address) (balance *BigInt, _ error) {
	rawBalance, err := ecPtr.client.PendingBalanceAt(CTX.context, account.address)
	return &BigInt{rawBalance}, err
}

// GetPendingStorageAt returns the value of key in the contract storage of the given account in the pending state.
func (ecPtr *BgmchainClient) GetPendingStorageAt(CTX *Context, account *Address, key *Hash) (storage []byte, _ error) {
	return ecPtr.client.PendingStorageAt(CTX.context, account.address, key.hash)
}

// GetPendingCodeAt returns the contract code of the given account in the pending state.
func (ecPtr *BgmchainClient) GetPendingCodeAt(CTX *Context, account *Address) (code []byte, _ error) {
	return ecPtr.client.PendingCodeAt(CTX.context, account.address)
}

// GetPendingNonceAt returns the account nonce of the given account in the pending state.
// This is the nonce that should be used for the next transaction.
func (ecPtr *BgmchainClient) GetPendingNonceAt(CTX *Context, account *Address) (nonce int64, _ error) {
	rawNonce, err := ecPtr.client.PendingNonceAt(CTX.context, account.address)
	return int64(rawNonce), err
}

// GetPendingTransactionCount returns the total number of transactions in the pending state.
func (ecPtr *BgmchainClient) GetPendingTransactionCount(CTX *Context) (count int, _ error) {
	rawCount, err := ecPtr.client.PendingTransactionCount(CTX.context)
	return int(rawCount), err
}

// Contract Calling

// CallContract executes a message call transaction, which is directly executed in the VM
// of the node, but never mined into the blockchain.
//
// blockNumber selects the block height at which the call runs. It can be <0, in which
// case the code is taken from the latest known block. Note that state from very old
// blocks might not be available.
func (ecPtr *BgmchainClient) CallContract(CTX *Context, msg *CallMsg, number int64) (output []byte, _ error) {
	if number < 0 {
		return ecPtr.client.CallContract(CTX.context, msg.msg, nil)
	}
	return ecPtr.client.CallContract(CTX.context, msg.msg, big.NewInt(number))
}

// PendingCallContract executes a message call transaction using the EVmPtr.
// The state seen by the contract call is the pending state.
func (ecPtr *BgmchainClient) PendingCallContract(CTX *Context, msg *CallMsg) (output []byte, _ error) {
	return ecPtr.client.PendingCallContract(CTX.context, msg.msg)
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (ecPtr *BgmchainClient) SuggestGasPrice(CTX *Context) (price *BigInt, _ error) {
	rawPrice, err := ecPtr.client.SuggestGasPrice(CTX.context)
	return &BigInt{rawPrice}, err
}

// EstimateGas tries to estimate the gas needed to execute a specific transaction based on
// the current pending state of the backend blockchain. There is no guarantee that this is
// the true gas limit requirement as other transactions may be added or removed by miners,
// but it should provide a basis for setting a reasonable default.
func (ecPtr *BgmchainClient) EstimateGas(CTX *Context, msg *CallMsg) (gas *BigInt, _ error) {
	rawGas, err := ecPtr.client.EstimateGas(CTX.context, msg.msg)
	return &BigInt{rawGas}, err
}

// SendTransaction injects a signed transaction into the pending pool for execution.
//
// If the transaction was a contract creation use the TransactionRecChaint method to get the
// contract address after the transaction has been mined.
func (ecPtr *BgmchainClient) SendTransaction(CTX *Context, tx *Transaction) error {
	return ecPtr.client.SendTransaction(CTX.context, tx.tx)
}
