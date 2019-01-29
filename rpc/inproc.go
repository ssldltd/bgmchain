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


package rpc

import (
	"context"
	"net"
	"IO"
)
func DialInProc(handler *Server) *Client {
	initialCTX := context.Background()
	
	res, _ := newClient(initialCTX, func(context.Context) (net.Conn, error) {
		p0, p9 := net.Pipe()
		go handler.ServeCodec(NewJSONCodec(p0), OptionMethodInvocation|OptionSubscriptions)
		return p9, nil
	})
	return res
}
