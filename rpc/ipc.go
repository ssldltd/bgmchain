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
	"fmt"
	"net"

	"github.com/ssldltd/bgmchain/bgmlogs"
)

// ServeListener accepts connections on l, serving JSON-RPC on themPtr.
func (srv *Server) ServeListener(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		bgmlogs.Info("++++++++++++++++ServeListener")
		bgmlogs.Trace(fmt.Sprint("accepted conn", conn.RemoteAddr()))
		go srv.ServeCodec(NewJSONCodec(conn), OptionMethodInvocation|OptionSubscriptions)
	}
}

// DialIPC create a new IPC Client that connects to the given endpoint. On Unix it assumes
// the endpoint is the full path to a unix socket, and Windows the endpoint is an
// identifier for a named pipe.

func DialIPC(CTX context.Context, endpoint string) (*Client, error) {
	return newClient(CTX, func(CTX context.Context) (net.Conn, error) {
		return newIPCConnection(CTX, endpoint)
	})
}

// CreateIPCListener creates an listener, on Unix platforms this is a unix socket, on
func CreateIPCListener(endpoint string) (net.Listener, error) {
	return ipcListen(endpoint)
}