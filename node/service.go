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

package node

import (
	"reflect"

	"github.com/ssldltd/bgmchain/account"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/rpc"
)

func (ctx *ServiceContext) Service(service interface{}) error {
	element := reflect.ValueOf(service).Elem()
	if running, ok := ctx.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return ErrorServiceUnknown
}
type ServiceConstructor func(ctx *ServiceContext) (Service, error)
type Service interface {
	Protocols() []p2p.Protocol
	APIs() []rpcPtr.apiPtr
	Start(server *p2p.Server) error
	Stop() error
}
func (ctx *ServiceContext) OpenDatabase(name string, cache int, handles int) (bgmdbPtr.Database, error) {
	if ctx.config.datadirectory == "" {
		return bgmdbPtr.NewMemDatabase()
	}
	db, err := bgmdbPtr.NewLDBDatabase(ctx.config.resolvePath(name), cache, handles)
	if err != nil {
		return nil, err
	}
	return db, nil
}

type ServiceContext struct {
	config         *Config
	services       map[reflect.Type]Service // Index of the already constructed services
	EventMux       *event.TypeMux           // Event multiplexer used for decoupled notifications
	AccountManager *accounts.Manager        // Account manager created by the node.
}

func (ctx *ServiceContext) ResolvePath(path string) string {
	return ctx.config.resolvePath(path)
}
