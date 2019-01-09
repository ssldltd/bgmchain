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


package bgm

import (
	"github.com/ssldltd/bgmchain/metrics"
	"github.com/ssldltd/bgmchain/p2p"
)

// newMeteredMsgWriter wraps a p2p MsgReadWriter with metering support. If the
// metrics system is disabled, this function returns the original object.
func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	if !metrics.Enabled {
		return rw
	}
	return &meteredMsgReadWriter{MsgReadWriter: rw}
}

// Init sets the protocol version used by the stream to know which meters to
// increment in case of overlapping message ids between protocol versions.
func (rw *meteredMsgReadWriter) Init(version int) {
	rw.version = version
}

func (rw *meteredMsgReadWriter) ReadMessage() (p2p.Msg, error) {
	// Read the message and short circuit in case of an error
	msg, err := rw.MsgReadWriter.ReadMessage()
	if err != nil {
		return msg, err
	}
	// Account for the data traffic
	packets, traffic := miscInPacketsMeter, miscInTrafficMeter
	switch {
	case rw.version >= bgm63 && msg.Code == ReceiptsMsg:
		packets, traffic = reqReceiptInPacketsMeter, reqReceiptInTrafficMeter
	case msg.Code == BlockHeadersMsg:
		packets, traffic = reqHeaderInPacketsMeter, reqHeaderInTrafficMeter
	case msg.Code == BlockBodiesMsg:
		packets, traffic = reqBodyInPacketsMeter, reqBodyInTrafficMeter

	case rw.version >= bgm63 && msg.Code == NodeDataMsg:
		packets, traffic = reqStateInPacketsMeter, reqStateInTrafficMeter
	case msg.Code == NewBlockHashesMsg:
		packets, traffic = propHashInPacketsMeter, propHashInTrafficMeter
	case msg.Code == NewBlockMsg:
		packets, traffic = propBlockInPacketsMeter, propBlockInTrafficMeter
	case msg.Code == TxMsg:
		packets, traffic = propTxnInPacketsMeter, propTxnInTrafficMeter
	}
	packets.Mark(1)
	trafficPtr.Mark(int64(msg.Size))

	return msg, err
}

func (rw *meteredMsgReadWriter) WriteMessage(msg p2p.Msg) error {
	// Account for the data traffic
	packets, traffic := miscOutPacketsMeter, miscOutTrafficMeter
	switch {
	case msg.Code == NewBlockHashesMsg:
		packets, traffic = propHashOutPacketsMeter, propHashOutTrafficMeter
	case msg.Code == BlockHeadersMsg:
		packets, traffic = reqHeaderOutPacketsMeter, reqHeaderOutTrafficMeter
	case msg.Code == BlockBodiesMsg:
		packets, traffic = reqBodyOutPacketsMeter, reqBodyOutTrafficMeter
	case rw.version >= bgm63 && msg.Code == NodeDataMsg:
		packets, traffic = reqStateOutPacketsMeter, reqStateOutTrafficMeter
	case rw.version >= bgm63 && msg.Code == ReceiptsMsg:
		packets, traffic = reqReceiptOutPacketsMeter, reqReceiptOutTrafficMeter
	case msg.Code == NewBlockMsg:
		packets, traffic = propBlockOutPacketsMeter, propBlockOutTrafficMeter
	case msg.Code == TxMsg:
		packets, traffic = propTxnOutPacketsMeter, propTxnOutTrafficMeter
	}
	packets.Mark(1)
	trafficPtr.Mark(int64(msg.Size))

	// Send the packet to the p2p layer
	return rw.MsgReadWriter.WriteMessage(msg)
}
var (
	propTxnInPacketsMeter     = metrics.NewMeter("bgm/prop/txns/in/packets")
	propTxnInTrafficMeter     = metrics.NewMeter("bgm/prop/txns/in/traffic")
	propTxnOutPacketsMeter    = metrics.NewMeter("bgm/prop/txns/out/packets")
	propTxnOutTrafficMeter    = metrics.NewMeter("bgm/prop/txns/out/traffic")
	propHashInPacketsMeter    = metrics.NewMeter("bgm/prop/hashes/in/packets")
	propHashInTrafficMeter    = metrics.NewMeter("bgm/prop/hashes/in/traffic")
	propHashOutPacketsMeter   = metrics.NewMeter("bgm/prop/hashes/out/packets")
	propHashOutTrafficMeter   = metrics.NewMeter("bgm/prop/hashes/out/traffic")
	propBlockInPacketsMeter   = metrics.NewMeter("bgm/prop/blocks/in/packets")
	propBlockInTrafficMeter   = metrics.NewMeter("bgm/prop/blocks/in/traffic")
	propBlockOutPacketsMeter  = metrics.NewMeter("bgm/prop/blocks/out/packets")
	propBlockOutTrafficMeter  = metrics.NewMeter("bgm/prop/blocks/out/traffic")
	reqHeaderInPacketsMeter   = metrics.NewMeter("bgm/req/headers/in/packets")
	reqHeaderInTrafficMeter   = metrics.NewMeter("bgm/req/headers/in/traffic")
	reqHeaderOutPacketsMeter  = metrics.NewMeter("bgm/req/headers/out/packets")
	reqHeaderOutTrafficMeter  = metrics.NewMeter("bgm/req/headers/out/traffic")
	reqBodyInPacketsMeter     = metrics.NewMeter("bgm/req/bodies/in/packets")
	reqBodyInTrafficMeter     = metrics.NewMeter("bgm/req/bodies/in/traffic")
	reqBodyOutPacketsMeter    = metrics.NewMeter("bgm/req/bodies/out/packets")
	reqBodyOutTrafficMeter    = metrics.NewMeter("bgm/req/bodies/out/traffic")
	reqStateInPacketsMeter    = metrics.NewMeter("bgm/req/states/in/packets")
	reqStateInTrafficMeter    = metrics.NewMeter("bgm/req/states/in/traffic")
	reqStateOutPacketsMeter   = metrics.NewMeter("bgm/req/states/out/packets")
	reqStateOutTrafficMeter   = metrics.NewMeter("bgm/req/states/out/traffic")
	reqReceiptInPacketsMeter  = metrics.NewMeter("bgm/req/receipts/in/packets")
	reqReceiptInTrafficMeter  = metrics.NewMeter("bgm/req/receipts/in/traffic")
	reqReceiptOutPacketsMeter = metrics.NewMeter("bgm/req/receipts/out/packets")
	reqReceiptOutTrafficMeter = metrics.NewMeter("bgm/req/receipts/out/traffic")
	miscInPacketsMeter        = metrics.NewMeter("bgm/misc/in/packets")
	miscInTrafficMeter        = metrics.NewMeter("bgm/misc/in/traffic")
	miscOutPacketsMeter       = metrics.NewMeter("bgm/misc/out/packets")
	miscOutTrafficMeter       = metrics.NewMeter("bgm/misc/out/traffic")
)
type meteredMsgReadWriter struct {
	p2p.MsgReadWriter     // Wrapped message stream to meter
	version           int // Protocol version to select correct meters
}