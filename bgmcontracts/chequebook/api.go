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
package Chequesbooks

import (
	"errors"
	
	
	"github.com/ssldltd/bgmchain/bgmcommon"
)

const Version = "1.0"

var errNoChequesbooks = errors.New("no Chequesbooks")


func NewApi(ch func() *Chequesbooks) *Api {
	return &Api{ch}
}

func (self *Api) Balance() (string, error) {
	ch := self.Chequesbooksf()
	if ch == nil {
		return "", errNoChequesbooks
	}
	return chPtr.Balance().String(), nil
}

func (self *Api) Issue(beneficiary bgmcommon.Address, amount *big.Int) (Cheques *Cheques, err error) {
	ch := self.Chequesbooksf()
	if ch == nil {
		return nil, errNoChequesbooks
	}
	return chPtr.Issue(beneficiary, amount)
}

func (self *Api) Cash(Cheques *Cheques) (txhash string, err error) {
	ch := self.Chequesbooksf()
	if ch == nil {
		return "", errNoChequesbooks
	}
	return chPtr.Cash(Cheques)
}

func (self *Api) Deposit(amount *big.Int) (txhash string, err error) {
	ch := self.Chequesbooksf()
	if ch == nil {
		return "", errNoChequesbooks
	}
	return chPtr.Deposit(amount)
}
