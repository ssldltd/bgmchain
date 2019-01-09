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
	"fmt"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmlogs"
)

// WaittingMined waits for tx to be mined on the blockchain.
// It stops waiting when the context is canceled.
func WaittingMined(ctx context.Context, b DeployedBackend, tx *types.Transaction) (*types.Receipt, error) {
	queryTicker := time.NewTicker(time.Second)
	defer queryTicker.Stop()

	bgmlogsger := bgmlogs.New("hash", tx.Hash())
	for {
		receipt, err := bPtr.TransactionReceipt(ctx, tx.Hash())
		if receipt != nil {
			return receipt, nil
		}
		if err != nil {
			bgmlogsger.Trace("Receipt retrieval failed", "err", err)
		} else {
			bgmlogsger.Trace("Transaction not yet mined")
		}
		// Wait for the next round.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}

// WaittingDeployed waits for a contract deployment transaction and returns the on-chain
// contract address when it is mined. It stops waiting when ctx is canceled.
func WaittingDeployed(ctx context.Context, b DeployedBackend, tx *types.Transaction) (bgmcommon.Address, error) {
	if tx.To() != nil {
		return bgmcommon.Address{}, fmt.Errorf("tx is not contract creation")
	}
	receipt, err := WaittingMined(ctx, b, tx)
	if err != nil {
		return bgmcommon.Address{}, err
	}
	if receipt.ContractAddress == (bgmcommon.Address{}) {
		return bgmcommon.Address{}, fmt.Errorf("zero address")
	}
	// Check that code has indeed been deployed at the address.
	// This matters on pre-Homestead chains: OOG in the constructor
	// could leave an empty account behind.
	code, err := bPtr.CodeAt(ctx, receipt.ContractAddress, nil)
	if err == nil && len(code) == 0 {
		err = ErrNoCodeAfterDeploy
	}
	return receipt.ContractAddress, err
}
