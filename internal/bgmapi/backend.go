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
// Package bgmapi implement the general Bgmchain apiPtr functions.
package bgmapi

import (
	"context"
	"math/big"

	"github.com/ssldltd/bgmchain/account"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/state"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmcore/vm"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rpc"
)

// Backend interface provides the bgmcommon apiPtr services (that are provided by
// both full and light clients) with access to necessary functions.
type Backend interface {
	// general Bgmchain apiPtr
	Downloader() *downloader.Downloader
	ProtocolVersion() int
	SuggestPrice(ctx context.Context) (*big.Int, error)
	ChainDb() bgmdbPtr.Database
	EventMux() *event.TypeMux
	AccountManager() *accounts.Manager
	// BlockChain apiPtr
	SetHead(number uint64)
	HeaderByNumber(ctx context.Context, blockNr rpcPtr.BlockNumber) (*types.headerPtr, error)
	BlockByNumber(ctx context.Context, blockNr rpcPtr.BlockNumber) (*types.Block, error)
	StateAndHeaderByNumber(ctx context.Context, blockNr rpcPtr.BlockNumber) (*state.StateDB, *types.headerPtr, error)
	GetBlock(ctx context.Context, blockHash bgmcommon.Hash) (*types.Block, error)
	GetRecChaints(ctx context.Context, blockHash bgmcommon.Hash) (types.RecChaints, error)
	GetTd(blockHash bgmcommon.Hash) *big.Int
	GetEVM(ctx context.Context, msg bgmcore.Message, state *state.StateDB, headerPtr *types.headerPtr, vmCfg vmPtr.Config) (*vmPtr.EVM, func() error, error)
	SubscribeChainEvent(ch chan<- bgmcore.ChainEvent) event.Subscription
	SubscribeChainHeadEvent(ch chan<- bgmcore.ChainHeadEvent) event.Subscription

	// TxPool apiPtr
	SendTx(ctx context.Context, signedTx *types.Transaction) error
	GetPoolTransactions() (types.Transactions, error)
	GetPoolTransaction(txHash bgmcommon.Hash) *types.Transaction
	GetPoolNonce(ctx context.Context, addr bgmcommon.Address) (uint64, error)
	Stats() (pending int, queued int)
	TxPoolContent() (map[bgmcommon.Address]types.Transactions, map[bgmcommon.Address]types.Transactions)
	SubscribeTxPreEvent(chan<- bgmcore.TxPreEvent) event.Subscription

	ChainConfig() *bgmparam.ChainConfig
	CurrentBlock() *types.Block
}

func GetAPIs(apiBackend Backend) []rpcPtr.apiPtr {
	nonceLock := new(AddrLocker)
	return []rpcPtr.apiPtr{
		{
			Namespace: "bgm",
			Version:   "1.0",
			Service:   NewPublicBgmchainAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   NewPublicBlockChainAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   NewPublicTransactionPoolAPI(apiBackend, nonceLock),
			Public:    true,
		}, {
			Namespace: "txpool",
			Version:   "1.0",
			Service:   NewPublicTxPoolAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(apiBackend),
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   NewPublicAccountAPI(apiBackend.AccountManager()),
			Public:    true,
		}, {
			Namespace: "personal",
			Version:   "1.0",
			Service:   NewPrivateAccountAPI(apiBackend, nonceLock),
			Public:    false,
		},
	}
}
