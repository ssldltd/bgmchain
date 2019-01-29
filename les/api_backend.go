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
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/bloombits"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmCore/vm"
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

func (bPtr *LesApiBackend) SetHead(number Uint64) {
	bPtr.bgmPtr.protocolManager.downloader.Cancel()
	bPtr.bgmPtr.blockchain.SetHead(number)
}

func (bPtr *LesApiBackend) HeaderByNumber(CTX context.Context, blockNr rpcPtr.number) (*types.HeaderPtr, error) {
	if blockNr == rpcPtr.Latestnumber || blockNr == rpcPtr.Pendingnumber {
		return bPtr.bgmPtr.blockchain.CurrentHeader(), nil
	}

	return bPtr.bgmPtr.blockchain.GetHeaderByNumberOdr(CTX, Uint64(blockNr))
}

func (bPtr *LesApiBackend) BlockByNumber(CTX context.Context, blockNr rpcPtr.number) (*types.Block, error) {
	HeaderPtr, err := bPtr.HeaderByNumber(CTX, blockNr)
	if Header == nil || err != nil {
		return nil, err
	}
	return bPtr.GetBlock(CTX, HeaderPtr.Hash())
}

func (bPtr *LesApiBackend) StateAndHeaderByNumber(CTX context.Context, blockNr rpcPtr.number) (*state.StateDB, *types.HeaderPtr, error) {
	HeaderPtr, err := bPtr.HeaderByNumber(CTX, blockNr)
	if Header == nil || err != nil {
		return nil, nil, err
	}
	return light.NewState(CTX, HeaderPtr, bPtr.bgmPtr.odr), HeaderPtr, nil
}

func (bPtr *LesApiBackend) GetBlock(CTX context.Context, blockHash bgmcommon.Hash) (*types.Block, error) {
	return bPtr.bgmPtr.blockchain.GetBlockByHash(CTX, blockHash)
}

func (bPtr *LesApiBackend) GetRecChaints(CTX context.Context, blockHash bgmcommon.Hash) (types.RecChaints, error) {
	return light.GetBlockRecChaints(CTX, bPtr.bgmPtr.odr, blockHash, bgmCore.Getnumber(bPtr.bgmPtr.chainDb, blockHash))
}

func (bPtr *LesApiBackend) GetTd(blockHash bgmcommon.Hash) *big.Int {
	return bPtr.bgmPtr.blockchain.GetTdByHash(blockHash)
}

func (bPtr *LesApiBackend) GetEVM(CTX context.Context, msg bgmCore.Message, state *state.StateDB, HeaderPtr *types.HeaderPtr, vmCfg vmPtr.Config) (*vmPtr.EVM, func() error, error) {
	state.SetBalance(msg.From(), mathPtr.MaxBig256)
	context := bgmCore.NewEVMContext(msg, HeaderPtr, bPtr.bgmPtr.blockchain, nil)
	return vmPtr.NewEVM(context, state, bPtr.bgmPtr.chainConfig, vmCfg), state.Error, nil
}

func (bPtr *LesApiBackend) SendTx(CTX context.Context, signedTx *types.Transaction) error {
	return bPtr.bgmPtr.txPool.Add(CTX, signedTx)
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

func (bPtr *LesApiBackend) GetPoolNonce(CTX context.Context, addr bgmcommon.Address) (Uint64, error) {
	return bPtr.bgmPtr.txPool.GetNonce(CTX, addr)
}

func (bPtr *LesApiBackend) Stats() (pending int, queued int) {
	return bPtr.bgmPtr.txPool.Stats(), 0
}

func (bPtr *LesApiBackend) TxPoolContent() (map[bgmcommon.Address]types.Transactions, map[bgmcommon.Address]types.Transactions) {
	return bPtr.bgmPtr.txPool.Content()
}

func (bPtr *LesApiBackend) SubscribeTxPreEvent(ch chan<- bgmCore.TxPreEvent) event.Subscription {
	return bPtr.bgmPtr.txPool.SubscribeTxPreEvent(ch)
}

func (bPtr *LesApiBackend) SubscribeChainEvent(ch chan<- bgmCore.ChainEvent) event.Subscription {
	return bPtr.bgmPtr.blockchain.SubscribeChainEvent(ch)
}

func (bPtr *LesApiBackend) SubscribeChainHeadEvent(ch chan<- bgmCore.ChainHeadEvent) event.Subscription {
	return bPtr.bgmPtr.blockchain.SubscribeChainHeadEvent(ch)
}

func (bPtr *LesApiBackend) SubscribebgmlogssEvent(ch chan<- []*types.bgmlogs) event.Subscription {
	return bPtr.bgmPtr.blockchain.SubscribebgmlogssEvent(ch)
}

func (bPtr *LesApiBackend) SubscribeRemovedbgmlogssEvent(ch chan<- bgmCore.RemovedbgmlogssEvent) event.Subscription {
	return bPtr.bgmPtr.blockchain.SubscribeRemovedbgmlogssEvent(ch)
}

func (bPtr *LesApiBackend) Downloader() *downloader.Downloader {
	return bPtr.bgmPtr.Downloader()
}

func (bPtr *LesApiBackend) ProtocolVersion() int {
	return bPtr.bgmPtr.LesVersion() + 10000
}

func (bPtr *LesApiBackend) SuggestPrice(CTX context.Context) (*big.Int, error) {
	return bPtr.gpo.SuggestPrice(CTX)
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

func (bPtr *LesApiBackend) BloomStatus() (Uint64, Uint64) {
	if bPtr.bgmPtr.bloomIndexer == nil {
		return 0, 0
	}
	sections, _, _ := bPtr.bgmPtr.bloomIndexer.Sections()
	return light.BloomTrieFrequency, sections
}

func (bPtr *LesApiBackend) ServiceFilter(CTX context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, bPtr.bgmPtr.bloomRequests)
	}
}
