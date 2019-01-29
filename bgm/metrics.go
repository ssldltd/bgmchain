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
	"github.com/ssldltd/bgmchain/metics"
	"github.com/ssldltd/bgmchain/p2p"
)

// newMeteredMsgWriter wraps a p2p MsgReadWriter with metering support. If the
// metics system is disabled, this function returns the original object.
func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	if !metics.Enabled {
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
	case msg.Code == NewhashesMsg:
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
	case msg.Code == NewhashesMsg:
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
	propTxnInPacketsMeter     = metics.NewMeter("bgm/prop/txns/in/packets")
	propTxnInTrafficMeter     = metics.NewMeter("bgm/prop/txns/in/traffic")
	propTxnOutPacketsMeter    = metics.NewMeter("bgm/prop/txns/out/packets")
	propTxnOutTrafficMeter    = metics.NewMeter("bgm/prop/txns/out/traffic")
	propHashInPacketsMeter    = metics.NewMeter("bgm/prop/hashes/in/packets")
	propHashInTrafficMeter    = metics.NewMeter("bgm/prop/hashes/in/traffic")
	propHashOutPacketsMeter   = metics.NewMeter("bgm/prop/hashes/out/packets")
	propHashOutTrafficMeter   = metics.NewMeter("bgm/prop/hashes/out/traffic")
	propBlockInPacketsMeter   = metics.NewMeter("bgm/prop/blocks/in/packets")
	propBlockInTrafficMeter   = metics.NewMeter("bgm/prop/blocks/in/traffic")
	propBlockOutPacketsMeter  = metics.NewMeter("bgm/prop/blocks/out/packets")
	propBlockOutTrafficMeter  = metics.NewMeter("bgm/prop/blocks/out/traffic")
	reqHeaderInPacketsMeter   = metics.NewMeter("bgm/req/Headers/in/packets")
	reqHeaderInTrafficMeter   = metics.NewMeter("bgm/req/Headers/in/traffic")
	reqHeaderOutPacketsMeter  = metics.NewMeter("bgm/req/Headers/out/packets")
	reqHeaderOutTrafficMeter  = metics.NewMeter("bgm/req/Headers/out/traffic")
	reqBodyInPacketsMeter     = metics.NewMeter("bgm/req/bodies/in/packets")
	reqBodyInTrafficMeter     = metics.NewMeter("bgm/req/bodies/in/traffic")
	reqBodyOutPacketsMeter    = metics.NewMeter("bgm/req/bodies/out/packets")
	reqBodyOutTrafficMeter    = metics.NewMeter("bgm/req/bodies/out/traffic")
	reqStateInPacketsMeter    = metics.NewMeter("bgm/req/states/in/packets")
	reqStateInTrafficMeter    = metics.NewMeter("bgm/req/states/in/traffic")
	reqStateOutPacketsMeter   = metics.NewMeter("bgm/req/states/out/packets")
	reqStateOutTrafficMeter   = metics.NewMeter("bgm/req/states/out/traffic")
	reqReceiptInPacketsMeter  = metics.NewMeter("bgm/req/receipts/in/packets")
	reqReceiptInTrafficMeter  = metics.NewMeter("bgm/req/receipts/in/traffic")
	reqReceiptOutPacketsMeter = metics.NewMeter("bgm/req/receipts/out/packets")
	reqReceiptOutTrafficMeter = metics.NewMeter("bgm/req/receipts/out/traffic")
	miscInPacketsMeter        = metics.NewMeter("bgm/misc/in/packets")
	miscInTrafficMeter        = metics.NewMeter("bgm/misc/in/traffic")
	miscOutPacketsMeter       = metics.NewMeter("bgm/misc/out/packets")
	miscOutTrafficMeter       = metics.NewMeter("bgm/misc/out/traffic")
)
type meteredMsgReadWriter struct {
	p2p.MsgReadWriter     // Wrapped message stream to meter
	version           int // Protocol version to select correct meters
}