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
	"bgmcrypto/ecdsa"
	"errors"
	"io"
	"io/ioutil"

	"github.com/ssldltd/bgmchain/account/keystore"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmcrypto"
)

// NewTransactor是一种可以轻松创建事务签名者的实用工具方法
//加密的json密钥流和相关的密码短语。
func NewTransactor(keyin io.Reader, passphrase string) (*TransactOpts, error) {
	json, err := ioutil.ReadAll(keyin)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(json, passphrase)
	if err != nil {
		return nil, err
	}
	return NewKeyedTransactor(key.PrivateKey), nil
}


// NewKeyedTransactor是一种轻松创建事务签名者的实用工具方法
//来自一个私钥。
func NewKeyedTransactor(key *ecdsa.PrivateKey) *TransactOpts {
	keyAddr := bgmcrypto.PubkeyToAddress(key.PublicKey)
	return &TransactOpts{
		From: keyAddr,
		Signer: func(signer types.Signer, address bgmcommon.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != keyAddr {
				return nil, errors.New("not authorized to sign this account")
			}
			signature, err := bgmcrypto.Sign(signer.Hash(tx).Bytes(), key)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
	}
}
