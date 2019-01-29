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
	"time"

	"gopkg.in/natefinch/npipe.v2"
)


// ipcListen will create a named pipe on the given endpoint.
func ipcListen(endpoint string) (net.Listener, error) {
	return npipe.Listen(endpoint)
}

func newIPCConnection(CTX context.Context, endpoint string) (net.Conn, error) {
	timeout := defaultPipeDialtimeout
	if deadline, ok := CTX.Deadline(); ok {
		timeout = deadline.Sub(time.Now())
		
		if timeout < 0 {
			timeout = 0
		}
	}
	return npipe.Dialtimeout(endpoint, timeout)
}
const defaultPipeDialtimeout = 2 * time.Second