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
package les

import (
	"context"
	"math/big"

	"github.com/ssldltd/bgmchain/account"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/bloombits"
	"github.com/ssldltd/bgmchain/bgmcore/state"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmcore/vm"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgm/gasprice"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/light"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rpc"
)

type LesApiBackend struct {
	bgmPtr *LightBgmchain
	gpo *gasprice.Oracle
}

func (bPtr *LesApiBackend) ChainConfig() *bgmparam.ChainConfig {
	return bPtr.bgmPtr.chainConfig
}

func (bPtr *LesApiBackend) CurrentBlock() *types.Block {
	return types.NewBlockWithHeader(bPtr.bgmPtr.BlockChain().CurrentHeader())
}

func (bPtr *LesApiBackend) SetHead(number uint64) {
	bPtr.bgmPtr.protocolManager.downloader.Cancel()
	bPtr.bgmPtr.blockchain.SetHead(number)
}

func (bPtr *LesApiBackend) HeaderByNumber(ctx context.Context, blockNr rpcPtr.BlockNumber) (*types.headerPtr, error) {
	if blockNr == rpcPtr.LatestBlockNumber || blockNr == rpcPtr.PendingBlockNumber {
		return bPtr.bgmPtr.blockchain.CurrentHeader(), nil
	}

	return bPtr.bgmPtr.blockchain.GetHeaderByNumberOdr(ctx, uint64(blockNr))
}

func (bPtr *LesApiBackend) BlockByNumber(ctx context.Context, blockNr rpcPtr.BlockNumber) (*types.Block, error) {
	headerPtr, err := bPtr.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, err
	}
	return bPtr.GetBlock(ctx, headerPtr.Hash())
}

func (bPtr *LesApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpcPtr.BlockNumber) (*state.StateDB, *types.headerPtr, error) {
	headerPtr, err := bPtr.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	return light.NewState(ctx, headerPtr, bPtr.bgmPtr.odr), headerPtr, nil
}

func (bPtr *LesApiBackend) GetBlock(ctx context.Context, blockHash bgmcommon.Hash) (*types.Block, error) {
	return bPtr.bgmPtr.blockchain.GetBlockByHash(ctx, blockHash)
}

func (bPtr *LesApiBackend) GetRecChaints(ctx context.Context, blockHash bgmcommon.Hash) (types.RecChaints, error) {
	return light.GetBlockRecChaints(ctx, bPtr.bgmPtr.odr, blockHash, bgmcore.GetBlockNumber(bPtr.bgmPtr.chainDb, blockHash))
}

func (bPtr *LesApiBackend) GetTd(blockHash bgmcommon.Hash) *big.Int {
	return bPtr.bgmPtr.blockchain.GetTdByHash(blockHash)
}

func (bPtr *LesApiBackend) GetEVM(ctx context.Context, msg bgmcore.Message, state *state.StateDB, headerPtr *types.headerPtr, vmCfg vmPtr.Config) (*vmPtr.EVM, func() error, error) {
	state.SetBalance(msg.From(), mathPtr.MaxBig256)
	context := bgmcore.NewEVMContext(msg, headerPtr, bPtr.bgmPtr.blockchain, nil)
	return vmPtr.NewEVM(context, state, bPtr.bgmPtr.chainConfig, vmCfg), state.Error, nil
}

func (bPtr *LesApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return bPtr.bgmPtr.txPool.Add(ctx, signedTx)
}

func (bPtr *LesApiBackend) RemoveTx(txHash bgmcommon.Hash) {
	bPtr.bgmPtr.txPool.RemoveTx(txHash)
}

func (bPtr *LesApiBackend) GetPoolTransactions() (types.Transactions, error) {
	return bPtr.bgmPtr.txPool.GetTransactions()
}

func (bPtr *LesApiBackend) GetPoolTransaction(txHash bgmcommon.Hash) *types.Transaction {
	return bPtr.bgmPtr.txPool.GetTransaction(txHash)
}

func (bPtr *LesApiBackend) GetPoolNonce(ctx context.Context, addr bgmcommon.Address) (uint64, error) {
	return bPtr.bgmPtr.txPool.GetNonce(ctx, addr)
}

func (bPtr *LesApiBackend) Stats() (pending int, queued int) {
	return bPtr.bgmPtr.txPool.Stats(), 0
}

func (bPtr *LesApiBackend) TxPoolContent() (map[bgmcommon.Address]types.Transactions, map[bgmcommon.Address]types.Transactions) {
	return bPtr.bgmPtr.txPool.Content()
}

func (bPtr *LesApiBackend) SubscribeTxPreEvent(ch chan<- bgmcore.TxPreEvent) event.Subscription {
	return bPtr.bgmPtr.txPool.SubscribeTxPreEvent(ch)
}

func (bPtr *LesApiBackend) SubscribeChainEvent(ch chan<- bgmcore.ChainEvent) event.Subscription {
	return bPtr.bgmPtr.blockchain.SubscribeChainEvent(ch)
}

func (bPtr *LesApiBackend) SubscribeChainHeadEvent(ch chan<- bgmcore.ChainHeadEvent) event.Subscription {
	return bPtr.bgmPtr.blockchain.SubscribeChainHeadEvent(ch)
}

func (bPtr *LesApiBackend) SubscribebgmlogssEvent(ch chan<- []*types.bgmlogs) event.Subscription {
	return bPtr.bgmPtr.blockchain.SubscribebgmlogssEvent(ch)
}

func (bPtr *LesApiBackend) SubscribeRemovedbgmlogssEvent(ch chan<- bgmcore.RemovedbgmlogssEvent) event.Subscription {
	return bPtr.bgmPtr.blockchain.SubscribeRemovedbgmlogssEvent(ch)
}

func (bPtr *LesApiBackend) Downloader() *downloader.Downloader {
	return bPtr.bgmPtr.Downloader()
}

func (bPtr *LesApiBackend) ProtocolVersion() int {
	return bPtr.bgmPtr.LesVersion() + 10000
}

func (bPtr *LesApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return bPtr.gpo.SuggestPrice(ctx)
}

func (bPtr *LesApiBackend) ChainDb() bgmdbPtr.Database {
	return bPtr.bgmPtr.chainDb
}

func (bPtr *LesApiBackend) EventMux() *event.TypeMux {
	return bPtr.bgmPtr.eventMux
}

func (bPtr *LesApiBackend) AccountManager() *accounts.Manager {
	return bPtr.bgmPtr.accountManager
}

func (bPtr *LesApiBackend) BloomStatus() (uint64, uint64) {
	if bPtr.bgmPtr.bloomIndexer == nil {
		return 0, 0
	}
	sections, _, _ := bPtr.bgmPtr.bloomIndexer.Sections()
	return light.BloomTrieFrequency, sections
}

func (bPtr *LesApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, bPtr.bgmPtr.bloomRequests)
	}
}
