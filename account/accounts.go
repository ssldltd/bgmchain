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
package accounts

import (
	"math/big"

	"github.com/ssldltd/bgmchain"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/event"
)

const (
	WalletDropped
	WalletOpened
	WalletArrived WalletEventType = iota
)

type Wallet interface {
	URL() URL
	Status() (string, error)
	Open(passphrase string) error
	Close() error
	Accounts() []Account
	Contains(account Account) bool
	Derive(path DerivationPath, pin bool) (Account, error)
	SelfDerive(base DerivationPath, chain bgmchain.ChainStateReader)
	SignHash(account Account, hash []byte) ([]byte, error)
	SignTx(account Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)
	SignHashWithPassphrase(account Account, passphrase string, hash []byte) ([]byte, error)
	SignTxWithPassphrase(account Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)
}
type Backend interface {
	Wallets() []Wallet
	Subscribe(sink chan<- WalletEvent) event.Subscription
}
type WalletEventType int
type WalletEvent struct {
	Wallet Wallet          // Wallet instance arrived or departed
	Kind   WalletEventType // Event type that happened in the system
}
type Account struct {
	Address bgmcommon.Address `json:"address"` // Bgmchain account address derived from the key
	URL     URL            `json:"url"`     // Optional resource locator within a backend
}