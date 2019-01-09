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
package trezor

import (
	"reflect"

	"github.com/golang/protobuf/proto"
)

func Type(msg proto.Message) uint16 {
	return uint16(MessagesType_value["MessagesType_"+reflect.TypeOf(msg).Elem().Name()])
}

// Name returns the friendly message type name of a specific protocol buffer
// type numbers.
func Name(kind uint16) string {
	name := MessagesType_name[int32(kind)]
	if len(name) < 12 {
		return name
	}
	return name[12:]
}
