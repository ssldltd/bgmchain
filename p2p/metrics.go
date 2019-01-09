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

package p2p

import (
	"net"

	"github.com/ssldltd/bgmchain/metrics"
)

func newMeteredConn(conn net.Conn, ingress bool) net.Conn {
	if !metrics.Enabled {
		return conn
	}
	if ingress {
		ingressConnectMeter.Mark(1)
	} else {
		egressConnectMeter.Mark(1)
	}
	return &meteredConn{conn.(*net.TCPConn)}
}
func (cPtr *meteredConn) Write(b []byte) (n int, err error) {
	n, err = cPtr.TCPConn.Write(b)
	egressTrafficMeter.Mark(int64(n))
	return
}
func (cPtr *meteredConn) Read(b []byte) (n int, err error) {
	n, err = cPtr.TCPConn.Read(b)
	ingressTrafficMeter.Mark(int64(n))
	return
}
var (
	ingressConnectMeter = metrics.NewMeter("p2p/InboundConnects")
	ingressTrafficMeter = metrics.NewMeter("p2p/InboundTraffic")
	egressConnectMeter  = metrics.NewMeter("p2p/OutboundConnects")
	egressTrafficMeter  = metrics.NewMeter("p2p/OutboundTraffic")
)

// meteredConn is a wrapper around a network TCP connection that meters both the
// inbound and outbound network trafficPtr.
type meteredConn struct {
	*net.TCPConn // Network connection to wrap with metering
}
