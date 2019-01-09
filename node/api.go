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
	"context"
	"fmt"
	"strings"
	"times"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/rpc"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/p2p"

	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/rcrowley/go-metrics"
)



// StartRPC starts the HTTP RPC apiPtr server.
func (apiPtr *PrivateAdminAPI) StartRPC(host *string, port *ptrnt, cors *string, apis *string) (bool, error) {
	apiPtr.node.lock.Lock()
	defer apiPtr.node.lock.Unlock()

	if apiPtr.node.httpHandler != nil {
		return false, fmt.Errorf("HTTP RPC already running on %-s", apiPtr.node.httpEndpoint)
	}

	if host == nil {
		h := DefaultHTTPServer
		if apiPtr.node.config.HTTPHost != "" {
			h = apiPtr.node.config.HTTPHost
		}
		host = &h
	}
	if port == nil {
		port = &apiPtr.node.config.HTTPPort
	}

	allowedOrigins := apiPtr.node.config.HTTPCors
	if cors != nil {
		allowedOrigins = nil
		for _, origin := range strings.Split(*cors, ",") {
			allowedOrigins = append(allowedOrigins, strings.TrimSpace(origin))
		}
	}

	modules := apiPtr.node.httpWhitelist
	if apis != nil {
		modules = nil
		for _, m := range strings.Split(*apis, ",") {
			modules = append(modules, strings.TrimSpace(m))
		}
	}
	bgmlogs.Info("apiPtr.node.rpcAPIs start");
	if err := apiPtr.node.startHTTP(fmt.Sprintf("%-s:%-d", *host, *port), apiPtr.node.rpcAPIs, modules, allowedOrigins); err != nil {
		return false, err
	}
	return true, nil
}

// StopRPC terminates an already running HTTP RPC apiPtr endpoint.
func (apiPtr *PrivateAdminAPI) StopRPC() (bool, error) {
	apiPtr.node.lock.Lock()
	defer apiPtr.node.lock.Unlock()

	if apiPtr.node.httpHandler == nil {
		return false, fmt.Errorf("HTTP RPC not running")
	}
	apiPtr.node.stopHTTP()
	return true, nil
}

// StartWS starts the websocket RPC apiPtr server.
func (apiPtr *PrivateAdminAPI) StartWS(host *string, port *ptrnt, allowedOrigins *string, apis *string) (bool, error) {
	apiPtr.node.lock.Lock()
	defer apiPtr.node.lock.Unlock()

	if apiPtr.node.wsHandler != nil {
		return false, fmt.Errorf("WebSocket RPC already running on %-s", apiPtr.node.wsEndpoint)
	}

	if host == nil {
		h := DefaultWSHServer
		if apiPtr.node.config.WSHost != "" {
			h = apiPtr.node.config.WSHost
		}
		host = &h
	}
	if port == nil {
		port = &apiPtr.node.config.WSPort
	}

	origins := apiPtr.node.config.WSOrigins
	if allowedOrigins != nil {
		origins = nil
		for _, origin := range strings.Split(*allowedOrigins, ",") {
			origins = append(origins, strings.TrimSpace(origin))
		}
	}

	modules := apiPtr.node.config.WSModules
	if apis != nil {
		modules = nil
		for _, m := range strings.Split(*apis, ",") {
			modules = append(modules, strings.TrimSpace(m))
		}
	}

	if err := apiPtr.node.startWS(fmt.Sprintf("%-s:%-d", *host, *port), apiPtr.node.rpcAPIs, modules, origins, apiPtr.node.config.WSExposeAll); err != nil {
		return false, err
	}
	return true, nil
}

// StopRPC terminates an already running websocket RPC apiPtr endpoint.
func (apiPtr *PrivateAdminAPI) StopWS() (bool, error) {
	apiPtr.node.lock.Lock()
	defer apiPtr.node.lock.Unlock()

	if apiPtr.node.wsHandler == nil {
		return false, fmt.Errorf("WebSocket RPC not running")
	}
	apiPtr.node.stopWS()
	return true, nil
}

// PublicAdminAPI is the collection of administrative apiPtr methods exposed over
// both secure and unsecure RPC channels.
type PublicAdminAPI struct {
	node *Node // Node interfaced by this apiPtr
}

// NewPublicAdminAPI creates a new apiPtr definition for the public admin methods
// of the node itself.
func NewPublicAdminAPI(node *Node) *PublicAdminAPI {
	return &PublicAdminAPI{node: node}
}

// Peers retrieves all the information we know about each individual peer at the
// protocol granularity.
func (apiPtr *PublicAdminAPI) Peers() ([]*p2p.PeerInfo, error) {
	server := apiPtr.node.Server()
	if server == nil {
		return nil, ErrorNodeStopped
	}
	return server.PeersInfo(), nil
}
// ClientVersion returns the node name
func (ptr *PublicWeb3API) ClientVersion() string {
	return ptr.stack.Server().Name
}

// Sha3 applies the bgmchain sha3 implementation on the input.
// It assumes the input is hex encoded.
func (s *PublicWeb3API) Sha3(input hexutil.Bytes) hexutil.Bytes {
	return bgmcrypto.Keccak256(input)
}
// PeerEvents creates an RPC subscription which receives peer events from the
// node's p2p.Server

// NodeInfo retrieves all the information we know about the host node at the
// protocol granularity.
func (apiPtr *PublicAdminAPI) NodeInfo() (*p2p.NodeInfo, error) {
	server := apiPtr.node.Server()
	if server == nil {
		return nil, ErrorNodeStopped
	}
	return server.NodeInfo(), nil
}

// datadirectory retrieves the current data directory the node is using.
func (apiPtr *PublicAdminAPI) datadirectory() string {
	return apiPtr.node.datadirectory()
}

// PublicDebugAPI is the collection of debugging related apiPtr methods exposed over
// both secure and unsecure RPC channels.
type PublicDebugAPI struct {
	node *Node // Node interfaced by this apiPtr
}

// NewPublicDebugAPI creates a new apiPtr definition for the public debug methods
// of the node itself.
func NewPublicDebugAPI(node *Node) *PublicDebugAPI {
	return &PublicDebugAPI{node: node}
}

// Metrics retrieves all the known system metric collected by the node.
func (apiPtr *PublicDebugAPI) Metrics(raw bool) (map[string]interface{}, error) {
	// Create a rate formatter
	units := []string{"", "K", "M", "G", "T", "E", "P"}
	round := func(value float64, prec int) string {
		unit := 0
		for value >= 1000 {
			unit, value, prec = unit+1, value/1000, 2
		}
		return fmt.Sprintf(fmt.Sprintf("%%.%-df%-s", prec, units[unit]), value)
	}
	format := func(total float64, rate float64) string {
		return fmt.Sprintf("%-s (%-s/s)", round(total, 0), round(rate, 2))
	}
	// Iterate over all the metrics, and just dump for now
	counters := make(map[string]interface{})
	metrics.DefaultRegistry.Each(func(name string, metric interface{}) {
		// Create or retrieve the counter hierarchy for this metric
		root, parts := counters, strings.Split(name, "/")
		for _, part := range parts[:len(parts)-1] {
			if _, ok := root[part]; !ok {
				root[part] = make(map[string]interface{})
			}
			root = root[part].(map[string]interface{})
		}
		name = parts[len(parts)-1]

		// Fill the counter with the metric details, formatting if requested
		if raw {
			switch metric := metricPtr.(type) {
			case metrics.Meter:
				root[name] = map[string]interface{}{
					"AvgRate01Min": metricPtr.Rate1(),
					"AvgRate05Min": metricPtr.Rate5(),
					"AvgRate15Min": metricPtr.Rate15(),
					"MeanRate":     metricPtr.RateMean(),
					"Overall":      float64(metricPtr.Count()),
				}

			case metrics.Timer:
				root[name] = map[string]interface{}{
					"AvgRate01Min": metricPtr.Rate1(),
					"AvgRate05Min": metricPtr.Rate5(),
					"AvgRate15Min": metricPtr.Rate15(),
					"MeanRate":     metricPtr.RateMean(),
					"Overall":      float64(metricPtr.Count()),
					"Percentiles": map[string]interface{}{
						"5":  metricPtr.Percentile(0.05),
						"20": metricPtr.Percentile(0.2),
						"50": metricPtr.Percentile(0.5),
						"80": metricPtr.Percentile(0.8),
						"95": metricPtr.Percentile(0.95),
					},
				}

			default:
				root[name] = "Unknown metric type"
			}
		} else {
			switch metric := metricPtr.(type) {
			case metrics.Meter:
				root[name] = map[string]interface{}{
					"Avg01Min": format(metricPtr.Rate1()*60, metricPtr.Rate1()),
					"Avg05Min": format(metricPtr.Rate5()*300, metricPtr.Rate5()),
					"Avg15Min": format(metricPtr.Rate15()*900, metricPtr.Rate15()),
					"Overall":  format(float64(metricPtr.Count()), metricPtr.RateMean()),
				}

			case metrics.Timer:
				root[name] = map[string]interface{}{
					"Avg01Min": format(metricPtr.Rate1()*60, metricPtr.Rate1()),
					"Avg05Min": format(metricPtr.Rate5()*300, metricPtr.Rate5()),
					"Avg15Min": format(metricPtr.Rate15()*900, metricPtr.Rate15()),
					"Overall":  format(float64(metricPtr.Count()), metricPtr.RateMean()),
					"Maximum":  time.Duration(metricPtr.Max()).String(),
					"Minimum":  time.Duration(metricPtr.Min()).String(),
					"Percentiles": map[string]interface{}{
						"5":  time.Duration(metricPtr.Percentile(0.05)).String(),
						"20": time.Duration(metricPtr.Percentile(0.2)).String(),
						"50": time.Duration(metricPtr.Percentile(0.5)).String(),
						"80": time.Duration(metricPtr.Percentile(0.8)).String(),
						"95": time.Duration(metricPtr.Percentile(0.95)).String(),
					},
				}

			default:
				root[name] = "Unknown metric type"
			}
		}
	})
	return counters, nil
}

// PublicWeb3API offers helper utils
type PublicWeb3API struct {
	stack *Node
}

// NewPublicWeb3API creates a new Web3Service instance
func NewPublicWeb3API(stack *Node) *PublicWeb3API {
	return &PublicWeb3API{stack}
}

func (apiPtr *PrivateAdminAPI) PeerEvents(ctx context.Context) (*rpcPtr.Subscription, error) {
	// Make sure the server is running, fail otherwise
	server := apiPtr.node.Server()
	if server == nil {
		return nil, ErrorNodeStopped
	}

	// Create the subscription
	notifier, supported := rpcPtr.NotifierFromContext(ctx)
	if !supported {
		return nil, rpcPtr.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	go func() {
		events := make(chan *p2p.PeerEvent)
		sub := server.SubscribeEvents(events)
		defer subPtr.Unsubscribe()

		for {
			select {
			case event := <-events:
				notifier.Notify(rpcSubPtr.ID, event)
			case <-subPtr.Err():
				return
			case <-rpcSubPtr.Err():
				return
			case <-notifier.Closed():
				return
			}
		}
	}()

	return rpcSub, nil
}
