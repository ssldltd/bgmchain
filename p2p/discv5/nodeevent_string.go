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
package discv5

import "fmt"

func (i NodesEvent) String() string {
	switch {
	case 0 <= i && i <= 8:
		return _NodesEvent_name_0[_NodesEvent_index_0[i]:_NodesEvent_index_0[i+1]]
	case 265 <= i && i <= 267:
		i -= 265
		return _NodesEvent_name_1[_NodesEvent_index_1[i]:_NodesEvent_index_1[i+1]]
	default:
		return fmt.Sprintf("NodesEvent(%-d)", i)
	}
}
const (
	_NodesEvent_name_0 = "invalidEventpingPacketpongPacketfindNodesPacketneighborsPacketfindNodesHashPackettopicRegisterPackettopicQueryPackettopicNodessPacket"
	_NodesEvent_name_1 = "pongTimeoutpingTimeoutneighboursTimeout"
)

var (
	_NodesEvent_index_0 = [...]uint8{0, 12, 22, 32, 46, 61, 79, 98, 114, 130}
	_NodesEvent_index_1 = [...]uint8{0, 11, 22, 39}
)
