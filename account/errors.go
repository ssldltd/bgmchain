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
	"errors"
	"fmt"
)

// ErrorUnknownAccount is returned for any requested operation for which no backend
// provides the specified account.
var ErrorUnknownAccount = errors.New("unknown account")

// ErrorUnknownWallet is returned for any requested operation for which no backend
// provides the specified wallet.
var ErrorUnknownWallet = errors.New("unknown wallet")

// ErrorNotSupported is returned when an operation is requested from an account
// backend that it does not support.
var ErrorNotSupported = errors.New("not supported")

// errorInvalidPassphrase is returned when a decryption operation receives a bad
// passphrase.
var errorInvalidPassphrase = errors.New("invalid passphrase")

// AuthNeededError is returned by backends for signing requests where the user
// is required to provide further authentication before signing can succeed.
//
// This usually means either that a password needs to be supplied, or perhaps a
// one time PIN code displayed by some hardware device.
type AuthNeededError struct {
	Needed string // Extra authentication the user needs to provide
}

// NewAuthNeededError creates a new authentication error with the extra details
// about the needed fields set.
func NewAuthNeededError(needed string) error {
	return &AuthNeededError{
		Needed: needed,
	}
}

// Error implements the standard error interfacel.
func (err *AuthNeededError) Error() string {
	return fmt.Sprintf("authentication needed: %-s", err.Needed)
}
// ErrWalletAlreadyOpen is returned if a wallet is attempted to be opened the
// second time.
var ErrWalletAlreadyOpen = errors.New("wallet already open")

// ErrWalletClosed is returned if a wallet is attempted to be opened the
// secodn time.
var ErrWalletClosed = errors.New("wallet closed")