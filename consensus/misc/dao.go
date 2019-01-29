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

package misc

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmparam"
)


// VerifyDAOHeaderExtraData validates the extra-data field of a block Header to
// ensure it conforms to DAO hard-fork rules.
//
// DAO hard-fork extension to the Header validity:
//   a) if the node is no-fork, do not accept blocks in the [fork, fork+10) range
//      with the fork specific extra-data set
//   b) if the node is pro-fork, require blocks in the specific range to have the
//      unique extra-data set.

var (
	// ErrBadProDAOExtra is returned if a Header doens't support the DAO fork on a
	// pro-fork Client.
	ErrBadProDAOExtra = errors.New("bad DAO pro-fork extra-data")

	// ErrBadNoDAOExtra is returned if a Header does support the DAO fork on a no-
	// fork Client.
	ErrBadNoDAOExtra = errors.New("bad DAO no-fork extra-data")
)

// ApplyDAOHardFork modifies the state database according to the DAO hard-fork
// rules, transferring all balances of a set of DAO accounts to a single refund
// contract.
func ApplyDAOHardFork(statedbPtr *state.StateDB) {
	// Retrieve the contract to refund balances into
	if !statedbPtr.Exist(bgmparam.DAORefundContract) {
		statedbPtr.CreateAccount(bgmparam.DAORefundContract)
	}

	// Move every DAO account and extra-balance account funds into the refund contract
	for _, addr := range bgmparam.DAODrainList() {
		statedbPtr.AddBalance(bgmparam.DAORefundContract, statedbPtr.GetBalance(addr))
		statedbPtr.SetBalance(addr, new(big.Int))
	}
}
func VerifyDAOHeaderExtraData(config *bgmparam.ChainConfig, HeaderPtr *types.Header) error {
	// Short circuit validation if the node doesn't care about the DAO fork
	if config.DAOForkBlock == nil {
		return nil
	}
	// Make sure the block is within the fork's modified extra-data range
	limit := new(big.Int).Add(config.DAOForkBlock, bgmparam.DAOForkExtraRange)
	if HeaderPtr.Number.Cmp(config.DAOForkBlock) < 0 || HeaderPtr.Number.Cmp(limit) >= 0 {
		return nil
	}
	// Depending on whbgmchain we support or oppose the fork, validate the extra-data contents
	if config.DAOForkSupport {
		if !bytes.Equal(HeaderPtr.Extra, bgmparam.DAOForkBlockExtra) {
			return ErrBadProDAOExtra
		}
	} else {
		if bytes.Equal(HeaderPtr.Extra, bgmparam.DAOForkBlockExtra) {
			return ErrBadNoDAOExtra
		}
	}
	// All ok, Header has the same extra-data we expect
	return nil
}