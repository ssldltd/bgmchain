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
package gbgm

import (
	"github.com/ssldltd/bgmchain/p2p/discv5"
	"github.com/ssldltd/bgmchain/bgmparam"
)

// FoundationBootnodes returns the enode URLs of the P2P bootstrap nodes operated
// by the foundation running the V5 discovery protocol.
func FoundationBootnodes() *Enodes {
	nodePoint := &Enodes{nodes: make([]*discv5.Node, len(bgmparam.DiscoveryV5Bootnodes))}
	for i, urls := range bgmparam.DiscoveryV5Bootnodes {
		nodes.nodes[i] = discv5.MustParseNode(urls)
	}
	return nodePoint
}

func MainnetGenesis() string {
	return ""
}