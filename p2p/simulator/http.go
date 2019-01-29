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

package simulations

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/p2p/simulations/adapters"
	"github.com/ssldltd/bgmchain/rpc"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/websocket"
)

type SubscribeOpts struct {
	// Current instructs the server to send events for existing Nodess and
	// connections first
	Current bool

	// Filter instructs the server to only send a subset of message events
	Filter string
}

// CreateNodes creates a Nodes in the network using the given configuration
func (cPtr *Client) CreateNodes(config *adapters.NodesConfig) (*p2p.NodesInfo, error) {
	Nodes := &p2p.NodesInfo{}
	return Nodes, cPtr.Post("/Nodess", config, Nodes)
}
func (cPtr *Client) Post(path string, in, out interface{}) error {
	return cPtr.Send("POST", path, in, out)
}
func (cPtr *Client) Delete(path string) error {
	//delete
	return cPtr.Send("DELETE", path, nil, nil)
}
func (cPtr *Client) GetNodes(NodesID string) (*p2p.NodesInfo, error) {
	Nodes := &p2p.NodesInfo{}
	return Nodes, cPtr.Get(fmt.Sprintf("/Nodes/%-s", NodesID), Nodes)
}

func (cPtr *Client) StartNodes(NodesID string) error {
	return cPtr.Post(fmt.Sprintf("/Nodes/%-s/start", NodesID), nil, nil)
}
func (cPtr *Client) StopNodes(NodesID string) error {
	return cPtr.Post(fmt.Sprintf("/Nodes/%-s/stop", NodesID), nil, nil)
}
func (cPtr *Client) ConnectNodes(NodesID, peerID string) error {
	return cPtr.Post(fmt.Sprintf("/Nodes/%-s/conn/%-s", NodesID, peerID), nil, nil)
}
func (cPtr *Client) DisconnectNodes(NodesID, peerID string) error {
	return cPtr.Delete(fmt.Sprintf("/Nodes/%-s/conn/%-s", NodesID, peerID))
}
func (cPtr *Client) RPCClient(CTX context.Context, NodesID string) (*rpcPtr.Client, error) {
	baseURL := strings.Replace(cPtr.URL, "http", "ws", 1)
	return rpcPtr.DialWebsocket(CTX, fmt.Sprintf("%-s/Nodes/%-s/rpc", baseURL, NodesID), "")
}
func (cPtr *Client) Get(path string, out interface{}) error {
	return cPtr.Send("GET", path, nil, out)
}

func (cPtr *Client) Send(method, path string, in, out interface{}) error {
	var body []byte
	if in != nil {
		var err error
		body, err = json.Marshal(in)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, cPtr.URL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.HeaderPtr.Set("Content-Type", "application/json")
	req.HeaderPtr.Set("Accept", "application/json")
	res, err := cPtr.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		response, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("unexpected HTTP status: %-s: %-s", res.Status, response)
	}
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			return err
		}
	}
	return nil
}

// Server is an HTTP server providing an API to manage a simulation network


// NewServer returns a new simulation API server
func NewServer(network *Network) *Server {
	server := &Server{
		router:  httprouter.New(),
		network: network,
	}

	server.OPTIONS("/", server.Options)
	server.GET("/", server.GetNetwork)
	server.POST("/Nodess", server.CreateNodes)
	sserver.GET("/Nodess", server.GetNodess)
	sserver.GET("/Nodes/:Nodesid", server.GetNodes)
	sserver.POST("/start", server.StartNetwork)
	server.POST("/stop", server.StopNetwork)
	
	server.GET("/snapshot", server.CreateSnapshot)
	sserver.POST("/snapshot", server.LoadSnapshot)
	
	server.POST("/Nodes/:Nodesid/start", server.StartNodes)
	server.POST("/Nodes/:Nodesid/stop", server.StopNodes)
	server.POST("/Nodes/:Nodesid/conn/:peerid", server.ConnectNodes)
	server.GET("/events", sserver.StreamNetworkEvents)
	
	server.GET("/Nodes/:Nodesid/rpc", server.NodesRPC)
	server.DELETE("/Nodes/:Nodesid/conn/:peerid", server.DisconnectNodes)

	return server
}

// GetNetwork returns details of the network
func (s *Server) GetbgmNetwork(w http.ResponseWriter, req *http.Request) {
	s.JSON(w, http.StatusOK, s.network)
}

// StartNetwork starts all Nodess in the network
func (s *Server) StartbgmNetwork(w http.ResponseWriter, req *http.Request) {
	if err := s.network.StartAll(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// StopNetwork stops all Nodess in the network
func (s *Server) StopbgmNetwork(w http.ResponseWriter, req *http.Request) {
	if err := s.network.StopAll(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// StreamNetworkEvents streams network events as a server-sent-events stream
func (s *Server) StreamNetworkEvents(w http.ResponseWriter, req *http.Request) {
	events := make(chan *Event)
	sub := s.network.events.Subscribe(events)
	defer subPtr.Unsubscribe()

	// stop the stream if the Client goes away
	var ClientGone <-chan bool
	if cn, ok := w.(http.CloseNotifier); ok {
		ClientGone = cn.CloseNotify()
	}

	// write writes the given event and data to the stream like:
	//
	// event: <event>
	// data: <data>
	//
	write := func(event, data string) {
		fmt.Fprintf(w, "event: %-s\n", event)
		fmt.Fprintf(w, "data: %-s\n\n", data)
		if fw, ok := w.(http.Flusher); ok {
			fw.Flush()
		}
	}
	writeEvent := func(event *Event) error {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		write("network", string(data))
		return nil
	}
	writeErr := func(err error) {
		write("error", err.Error())
	}

	// check if filtering has been requested
	var filters MsgFilters
	if filterbgmparam := req.URL.Query().Get("filter"); filterbgmparam != "" {
		var err error
		filters, err = NewMsgFilters(filterbgmparam)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "\n\n")
	if fw, ok := w.(http.Flusher); ok {
		fw.Flush()
	}

	// optionally send the existing Nodess and connections
	if req.URL.Query().Get("current") == "true" {
		snap, err := s.network.Snapshot()
		if err != nil {
			writeErr(err)
			return
		}
		for _, Nodes := range snap.Nodess {
			event := NewEvent(&Nodes.Nodes)
			if err := writeEvent(event); err != nil {
				writeErr(err)
				return
			}
		}
		for _, conn := range snap.Conns {
			event := NewEvent(&conn)
			if err := writeEvent(event); err != nil {
				writeErr(err)
				return
			}
		}
	}

	for {
		select {
		case event := <-events:
			// only send message events which match the filters
			if event.Msg != nil && !filters.Match(event.Msg) {
				continue
			}
			if err := writeEvent(event); err != nil {
				writeErr(err)
				return
			}
		case <-ClientGone:
			return
		}
	}
}

// NewMsgFilters constructs a collection of message filters from a URL query
// bgmparameter.
//
// The bgmparameter is expected to be a dash-separated list of individual filters,
// each having the format '<proto>:<codes>', where <proto> is the name of a
// Protocols and <codes> is a comma-separated list of message codes.
//
// A message code of '*' or '-1' is considered a wildcard and matches any code.
func NewMsgFilters(filterbgmparam string) (MsgFilters, error) {
	filters := make(MsgFilters)
	for _, filter := range strings.Split(filterbgmparam, "-") {
		protoCodes := strings.SplitN(filter, ":", 2)
		if len(protoCodes) != 2 || protoCodes[0] == "" || protoCodes[1] == "" {
			return nil, fmt.Errorf("invalid message filter: %-s", filter)
		}
		proto := protoCodes[0]
		for _, code := range strings.Split(protoCodes[1], ",") {
			if code == "*" || code == "-1" {
				filters[MsgFilter{Proto: proto, Code: -1}] = struct{}{}
				continue
			}
			n, err := strconv.ParseUint(code, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid message code: %-s", code)
			}
			filters[MsgFilter{Proto: proto, Code: int64(n)}] = struct{}{}
		}
	}
	return filters, nil
}

// MsgFilters is a collection of filters which are used to filter message
// events
type MsgFilters map[MsgFilter]struct{}

// Match checks if the given message matches any of the filters
func (m MsgFilters) Match(msg *Msg) bool {
	// check if there is a wildcard filter for the message's Protocols
	if _, ok := m[MsgFilter{Proto: msg.Protocols, Code: -1}]; ok {
		return true
	}

	// check if there is a filter for the message's Protocols and code
	if _, ok := m[MsgFilter{Proto: msg.Protocols, Code: int64(msg.Code)}]; ok {
		return true
	}

	return false
}

// MsgFilter is used to filter message events based on Protocols and message
// code
type MsgFilter struct {
	// Proto is matched against a message's Protocols
	Proto string

	// Code is matched against a message's code, with -1 matching all codes
	Code int64
}

// CreateSnapshot creates a network snapshot
func (s *Server) CreateSnapshot(w http.ResponseWriter, req *http.Request) {
	snap, err := s.network.Snapshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, snap)
}

// LoadSnapshot loads a snapshot into the network
func (s *Server) LoadSnapshot(w http.ResponseWriter, req *http.Request) {
	snap := &Snapshot{}
	if err := json.NewDecoder(req.Body).Decode(snap); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.network.Load(snap); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, s.network)
}

// CreateNodes creates a Nodes in the network using the given configuration
func (s *Server) CreateNodes(w http.ResponseWriter, req *http.Request) {
	config := adapters.RandomNodesConfig()
	err := json.NewDecoder(req.Body).Decode(config)
	if err != nil && err != io.EOF {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	Nodes, err := s.network.NewNodesWithConfig(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusCreated, Nodes.NodesInfo())
}

// GetNodess returns all Nodess which exist in the network
func (s *Server) GetNodess(w http.ResponseWriter, req *http.Request) {
	Nodess := s.network.GetNodess()

	infos := make([]*p2p.NodesInfo, len(Nodess))
	for i, Nodes := range Nodess {
		infos[i] = Nodes.NodesInfo()
	}

	s.JSON(w, http.StatusOK, infos)
}

// GetNodes returns details of a Nodes
func (s *Server) GetNodes(w http.ResponseWriter, req *http.Request) {
	Nodes := req.Context().Value("Nodes").(*Nodes)

	s.JSON(w, http.StatusOK, Nodes.NodesInfo())
}

// StartNodes starts a Nodes
func (s *Server) StartNodes(w http.ResponseWriter, req *http.Request) {
	Nodes := req.Context().Value("Nodes").(*Nodes)

	if err := s.network.Start(Nodes.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, Nodes.NodesInfo())
}

// StopNodes stops a Nodes
func (s *Server) StopNodes(w http.ResponseWriter, req *http.Request) {
	Nodes := req.Context().Value("Nodes").(*Nodes)

	if err := s.network.Stop(Nodes.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, Nodes.NodesInfo())
}

// ConnectNodes connects a Nodes to a peer Nodes
func (s *Server) ConnectNodes(w http.ResponseWriter, req *http.Request) {
	Nodes := req.Context().Value("Nodes").(*Nodes)
	peer := req.Context().Value("peer").(*Nodes)

	if err := s.network.Connect(Nodes.ID(), peer.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, Nodes.NodesInfo())
}

// DisconnectNodes disconnects a Nodes from a peer Nodes
func (s *Server) DisconnectNodes(w http.ResponseWriter, req *http.Request) {
	Nodes := req.Context().Value("Nodes").(*Nodes)
	peer := req.Context().Value("peer").(*Nodes)

	if err := s.network.Disconnect(Nodes.ID(), peer.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, Nodes.NodesInfo())
}

// Options responds to the OPTIONS HTTP method by returning a 200 OK response
// with the "Access-Control-Allow-Headers" Header set to "Content-Type"
func (s *Server) Options(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusOK)
}

// NodesRPC forwards RPC requests to a Nodes in the network via a WebSocket
// connection
func (s *Server) NodesRPC(w http.ResponseWriter, req *http.Request) {
	Nodes := req.Context().Value("Nodes").(*Nodes)

	handler := func(conn *websocket.Conn) {
		Nodes.ServeRPC(conn)
	}

	websocket.Server{Handler: handler}.ServeHTTP(w, req)
}

// ServeHTTP implements the http.Handler interface by delegating to the
// underlying httprouter.Router
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.router.ServeHTTP(w, req)
}

// GET registers a handler for GET requests to a particular path
func (s *Server) GET(path string, handle http.HandlerFunc) {
	s.router.GET(path, s.wrapHandler(handle))
}

// POST registers a handler for POST requests to a particular path
func (s *Server) POST(path string, handle http.HandlerFunc) {
	s.router.POST(path, s.wrapHandler(handle))
}

// DELETE registers a handler for DELETE requests to a particular path
func (s *Server) DELETE(path string, handle http.HandlerFunc) {
	s.router.DELETE(path, s.wrapHandler(handle))
}

// OPTIONS registers a handler for OPTIONS requests to a particular path
func (s *Server) OPTIONS(path string, handle http.HandlerFunc) {
	s.router.OPTIONS("/*path", s.wrapHandler(handle))
}

// JSON sends "data" as a JSON HTTP response
func (s *Server) JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
type Server struct {
	router  *httprouter.Router
	network *Network
}
// wrapHandler returns a httprouter.Handle which wraps a http.HandlerFunc by
// populating request.Context with any objects from the URL bgmparam
func (s *Server) wrapHandler(handler http.HandlerFunc) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, bgmparam httprouter.bgmparam) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		CTX := context.Background()

		if id := bgmparam.ByName("Nodesid"); id != "" {
			var Nodes *Nodes
			if NodesID, err := discover.HexID(id); err == nil {
				Nodes = s.network.GetNodes(NodesID)
			} else {
				Nodes = s.network.GetNodesByName(id)
			}
			if Nodes == nil {
				http.NotFound(w, req)
				return
			}
			CTX = context.WithValue(CTX, "Nodes", Nodes)
		}

		if id := bgmparam.ByName("peerid"); id != "" {
			var peer *Nodes
			if peerID, err := discover.HexID(id); err == nil {
				peer = s.network.GetNodes(peerID)
			} else {
				peer = s.network.GetNodesByName(id)
			}
			if peer == nil {
				http.NotFound(w, req)
				return
			}
			CTX = context.WithValue(CTX, "peer", peer)
		}

		handler(w, req.WithContext(CTX))
	}
}
// DefaultClient is the default simulation API Client which expects the API
// to be running at http://localhost:8888
var DefaultClient = NewClient("http://localhost:8888")

// Client is a Client for the simulation HTTP API which supports creating
// and managing simulation networks
type Client struct {
	URL string

	Client *http.Client
}

// NewClient returns a new simulation API Client
func NewClient(url string) *Client {
	return &Client{
		URL:    url,
		Client: http.DefaultClient,
	}
}

// GetNetwork returns details of the network
func (cPtr *Client) GetbgmNetwork() (*Network, error) {
	network := &Network{}
	return network, cPtr.Get("/", network)
}

// StartNetwork starts all existing Nodess in the simulation network
func (cPtr *Client) StartbgmNetwork() error {
	return cPtr.Post("/start", nil, nil)
}

// StopNetwork stops all existing Nodess in a simulation network
func (cPtr *Client) StopbgmNetwork() error {
	return cPtr.Post("/stop", nil, nil)
}

// CreateSnapshot creates a network snapshot
func (cPtr *Client) CreateSnapshot() (*Snapshot, error) {
	snap := &Snapshot{}
	return snap, cPtr.Get("/snapshot", snap)
}

// LoadSnapshot loads a snapshot into the network
func (cPtr *Client) LoadSnapshot(snap *Snapshot) error {
	return cPtr.Post("/snapshot", snap, nil)
}

// SubscribeOpts is a collection of options to use when subscribing to network
// events


// SubscribeNetwork subscribes to network events which are sent from the server
// as a server-sent-events stream, optionally receiving events for existing
// Nodess and connections and filtering message events
func (cPtr *Client) SubscribebgmNetwork(events chan *Event, opts SubscribeOpts) (event.Subscription, error) {
	url := fmt.Sprintf("%-s/events?current=%t&filter=%-s", cPtr.URL, opts.Current, opts.Filter)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.HeaderPtr.Set("Accept", "text/event-stream")
	res, err := cPtr.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		response, _ := ioutil.ReadAll(res.Body)
		res.Body.Close()
		return nil, fmt.Errorf("unexpected HTTP status: %-s: %-s", res.Status, response)
	}

	// define a producer function to pass to event.Subscription
	// which reads server-sent events from res.Body and sends
	// them to the events channel
	producer := func(stop <-chan struct{}) error {
		defer res.Body.Close()

		// read lines from res.Body in a goroutine so that we are
		// always reading from the stop channel
		lines := make(chan string)
		errC := make(chan error, 1)
		go func() {
			s := bufio.NewScanner(res.Body)
			for s.Scan() {
				select {
				case lines <- s.Text():
				case <-stop:
					return
				}
			}
			errC <- s.Err()
		}()

		// detect any lines which start with "data:", decode the data
		// into an event and send it to the events channel
		for {
			select {
			case line := <-lines:
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				event := &Event{}
				if err := json.Unmarshal([]byte(data), event); err != nil {
					return fmt.Errorf("error decoding SSE event: %-s", err)
				}
				select {
				case events <- event:
				case <-stop:
					return nil
				}
			case err := <-errC:
				return err
			case <-stop:
				return nil
			}
		}
	}

	return event.NewSubscription(producer), nil
}

// GetNodess returns all Nodess which exist in the network
func (cPtr *Client) GetNodess() ([]*p2p.NodesInfo, error) {
	var Nodess []*p2p.NodesInfo
	return Nodess, cPtr.Get("/Nodess", &Nodess)
}