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

package bgmCore

import (
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore/types"
)

// TxPreEvent is posted when a transaction enters the transaction pool.
type TxPreEvent struct{ Tx *types.Transaction }

// PendingbgmlogssEvent is posted pre mining and notifies of pending bgmlogss.
type PendingbgmlogssEvent struct {
	bgmlogss []*types.bgmlogs
}

// PendingStateEvent is posted pre mining and notifies of pending state changes.
type PendingStateEvent struct{}

// NewMinedBlockEvent is posted when a block has been imported.
type NewMinedBlockEvent struct{ Block *types.Block }

// RemovedTransactionEvent is posted when a reorg happens
type RemovedTransactionEvent struct{ Txs types.Transactions }

// RemovedbgmlogssEvent is posted when a reorg happens
type RemovedbgmlogssEvent struct{ bgmlogss []*types.bgmlogs }

type ChainEvent struct {
	Block *types.Block
	Hash  bgmcommon.Hash
	bgmlogss  []*types.bgmlogs
}

type ChainHeadEvent struct{ Block *types.Block }
