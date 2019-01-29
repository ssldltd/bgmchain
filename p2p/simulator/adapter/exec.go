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

package adapters

import (
	"bufio"
	"context"
	"bgmcrypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/Nodes"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/rpc"
	"golang.org/x/net/websocket"
)

type wsRPCDialer struct {
	addrs map[string]string
}

// DialRPC implements the RPCDialer interface by creating a WebSocket RPC
// client of the given Nodes
func (w *wsRPCDialer) DialRPC(id discover.NodesID) (*rpcPtr.Client, error) {
	addr, ok := w.addrs[id.String()]
	if !ok {
		return nil, fmt.Errorf("unknown Nodes: %-s", id)
	}
	return rpcPtr.DialWebsocket(context.Background(), addr, "http://localhost")
}
// Name returns the name of the adapter for bgmlogsging purposes
func (e *ExecAdapter) Name() string {
	return "exec-adapter"
}

// NewNodes returns a new ExecNodes using the given config
func (e *ExecAdapter) NewNodes(config *NodesConfig) (Nodes, error) {
	if len(config.Services) == 0 {
		return nil, errors.New("Nodes must have at least one service")
	}
	for _, service := range config.Services {
		if _, exists := serviceFuncs[service]; !exists {
			return nil, fmt.Errorf("unknown Nodes service %q", service)
		}
	}

	// create the Nodes directory using the first 12 characters of the ID
	// as Unix socket paths cannot be longer than 256 characters
	dir := filepathPtr.Join(e.BaseDir, config.ID.String()[:12])
	if err := os.Mkdir(dir, 0755); err != nil {
		return nil, fmt.Errorf("error creating Nodes directory: %-s", err)
	}

	// generate the config
	conf := &execNodesConfig{
		Stack: Nodes.DefaultConfig,
		Nodes:  config,
	}
	conf.Stack.DataDir = filepathPtr.Join(dir, "data")
	conf.Stack.WSHost = "127.0.0.1"
	conf.Stack.WSPort = 0
	conf.Stack.WSOrigins = []string{"*"}
	conf.Stack.WSExposeAll = true
	conf.Stack.P2P.EnableMsgEvents = false
	conf.Stack.P2P.NoDiscovery = true
	conf.Stack.P2P.NAT = nil
	conf.Stack.NoUSB = true

	// listen on a random localhost port (we'll get the actual port after
	// starting the Nodes through the RPC admin.NodesInfo method)
	conf.Stack.P2P.ListenAddr = "127.0.0.1:0"

	Nodes := &ExecNodes{
		ID:      config.ID,
		Dir:     dir,
		Config:  conf,
		adapter: e,
	}
	Nodes.newCmd = Nodes.execCommand
	e.Nodess[Nodes.ID] = Nodes
	return Nodes, nil
}

// ExecNodes starts a simulation Nodes by exec'ing the current binary and
// running the configured services
type ExecNodes struct {
	ID     discover.NodesID
	Dir    string
	Config *execNodesConfig
	Cmd    *execPtr.Cmd
	Info   *p2p.NodesInfo

	adapter *ExecAdapter
	client  *rpcPtr.Client
	wsAddr  string
	newCmd  func() *execPtr.Cmd
	key     *ecdsa.PrivateKey
}

// Addr returns the Nodes's eNodes URL
func (n *ExecNodes) Addr() []byte {
	if n.Info == nil {
		return nil
	}
	return []byte(n.Info.ENodes)
}

// Client returns an rpcPtr.Client which can be used to communicate with the
// underlying services (it is set once the Nodes has started)
func (n *ExecNodes) Client() (*rpcPtr.Client, error) {
	return n.client, nil
}

// wsAddrPattern is a regex used to read the WebSocket address from the Nodes's
// bgmlogs
var wsAddrPattern = regexp.MustCompile(`ws://[\d.:]+`)

// Start exec's the Nodes passing the ID and service as command line arguments
// and the Nodes config encoded as JSON in the _P2P_Nodes_CONFIG environment
// variable
func (n *ExecNodes) Start(snapshots map[string][]byte) (err error) {
	if n.Cmd != nil {
		return errors.New("already started")
	}
	defer func() {
		if err != nil {
			bgmlogs.Error("Nodes failed to start", "err", err)
			n.Stop()
		}
	}()

	// encode a copy of the config containing the snapshot
	confCopy := *n.Config
	confCopy.Snapshots = snapshots
	confCopy.PeerAddrs = make(map[string]string)
	for id, Nodes := range n.adapter.Nodess {
		confCopy.PeerAddrs[id.String()] = Nodes.wsAddr
	}
	confData, err := json.Marshal(confCopy)
	if err != nil {
		return fmt.Errorf("error generating Nodes config: %-s", err)
	}

	// use a pipe for stderr so we can both copy the Nodes's stderr to
	// os.Stderr and read the WebSocket address from the bgmlogss
	stderrR, stderrW := io.Pipe()
	stderr := io.MultiWriter(os.Stderr, stderrW)

	// start the Nodes
	cmd := n.newCmd()
	cmd.Stdout = os.Stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("_P2P_Nodes_CONFIG=%-s", confData))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting Nodes: %-s", err)
	}
	n.Cmd = cmd

	// read the WebSocket address from the stderr bgmlogss
	var wsAddr string
	wsAddrC := make(chan string)
	go func() {
		s := bufio.NewScanner(stderrR)
		for s.Scan() {
			if strings.Contains(s.Text(), "WebSocket endpoint opened:") {
				wsAddrC <- wsAddrPattern.FindString(s.Text())
			}
		}
	}()
	select {
	case wsAddr = <-wsAddrC:
		if wsAddr == "" {
			return errors.New("failed to read WebSocket address from stderr")
		}
	case <-time.After(10 * time.Second):
		return errors.New("timed out waiting for WebSocket address on stderr")
	}

	// create the RPC client and load the Nodes info
	CTX, cancel := context.Withtimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := rpcPtr.DialWebsocket(CTX, wsAddr, "")
	if err != nil {
		return fmt.Errorf("error dialing rpc websocket: %-s", err)
	}
	var info p2p.NodesInfo
	if err := client.CallContext(CTX, &info, "admin_NodesInfo"); err != nil {
		return fmt.Errorf("error getting Nodes info: %-s", err)
	}
	n.client = client
	n.wsAddr = wsAddr
	n.Info = &info

	return nil
}

// execCommand returns a command which runs the Nodes locally by exec'ing
// the current binary but setting argv[0] to "p2p-Nodes" so that the child
// runs execP2PNodes
func (n *ExecNodes) execCommand() *execPtr.Cmd {
	return &execPtr.Cmd{
		Path: reexecPtr.Self(),
		Args: []string{"p2p-Nodes", strings.Join(n.Config.Nodes.Services, ","), n.ID.String()},
	}
}

// Stop stops the Nodes by first sending SIGTERM and then SIGKILL if the Nodes
// doesn't stop within 5s
func (n *ExecNodes) Stop() error {
	if n.Cmd == nil {
		return nil
	}
	defer func() {
		n.Cmd = nil
	}()

	if n.client != nil {
		n.client.Close()
		n.client = nil
		n.wsAddr = ""
		n.Info = nil
	}

	if err := n.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return n.Cmd.Process.Kill()
	}
	waitErr := make(chan error)
	go func() {
		waitErr <- n.Cmd.Wait()
	}()
	select {
	case err := <-waitErr:
		return err
	case <-time.After(5 * time.Second):
		return n.Cmd.Process.Kill()
	}
}

// NodesInfo returns information about the Nodes
func (n *ExecNodes) NodesInfo() *p2p.NodesInfo {
	info := &p2p.NodesInfo{
		ID: n.ID.String(),
	}
	if n.client != nil {
		n.client.Call(&info, "admin_NodesInfo")
	}
	return info
}

// ServeRPC serves RPC requests over the given connection by dialling the
// Nodes's WebSocket address and joining the two connections
func (n *ExecNodes) ServeRPC(clientConn net.Conn) error {
	conn, err := websocket.Dial(n.wsAddr, "", "http://localhost")
	if err != nil {
		return err
	}
	var wg syncPtr.WaitGroup
	wg.Add(2)
	join := func(src, dst net.Conn) {
		defer wg.Done()
		io.Copy(dst, src)
		// close the write end of the destination connection
		if cw, ok := dst.(interface {
			CloseWrite() error
		}); ok {
			cw.CloseWrite()
		} else {
			dst.Close()
		}
	}
	go join(conn, clientConn)
	go join(clientConn, conn)
	wg.Wait()
	return nil
}

// Snapshots creates snapshots of the services by calling the
// simulation_snapshot RPC method
func (n *ExecNodes) Snapshots() (map[string][]byte, error) {
	if n.client == nil {
		return nil, errors.New("RPC not started")
	}
	var snapshots map[string][]byte
	return snapshots, n.client.Call(&snapshots, "simulation_snapshot")
}

func init() {
	// register a reexec function to start a devp2p Nodes when the current
	// binary is executed as "p2p-Nodes"
	reexecPtr.Register("p2p-Nodes", execP2PNodes)
}

// execNodesConfig is used to serialize the Nodes configuration so it can be
// passed to the child process as a JSON encoded environment variable
type execNodesConfig struct {
	Stack     Nodes.Config       `json:"stack"`
	Nodes      *NodesConfig       `json:"Nodes"`
	Snapshots map[string][]byte `json:"snapshots,omitempty"`
	PeerAddrs map[string]string `json:"peer_addrs,omitempty"`
}

// execP2PNodes starts a devp2p Nodes when the current binary is executed with
// argv[0] being "p2p-Nodes", reading the service / ID from argv[1] / argv[2]
// and the Nodes config from the _P2P_Nodes_CONFIG environment variable
func execP2PNodes() {
	gbgmlogsger := bgmlogs.NewGbgmlogsHandler(bgmlogs.StreamHandler(os.Stderr, bgmlogs.bgmlogsfmtFormat()))
	gbgmlogsger.Verbosity(bgmlogs.LvlInfo)
	bgmlogs.Root().SetHandler(gbgmlogsger)

	// read the services from argv
	serviceNames := strings.Split(os.Args[1], ",")

	// decode the config
	confEnv := os.Getenv("_P2P_Nodes_CONFIG")
	if confEnv == "" {
		bgmlogs.Crit("missing _P2P_Nodes_CONFIG")
	}
	var conf execNodesConfig
	if err := json.Unmarshal([]byte(confEnv), &conf); err != nil {
		bgmlogs.Crit("error decoding _P2P_Nodes_CONFIG", "err", err)
	}
	conf.Stack.P2P.PrivateKey = conf.Nodes.PrivateKey

	// use explicit IP address in ListenAddr so that ENodes URL is usable
	externalIP := func() string {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			bgmlogs.Crit("error getting IP address", "err", err)
		}
		for _, addr := range addrs {
			if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() {
				return ip.IP.String()
			}
		}
		bgmlogs.Crit("unable to determine explicit IP address")
		return ""
	}
	if strings.HasPrefix(conf.Stack.P2P.ListenAddr, ":") {
		conf.Stack.P2P.ListenAddr = externalIP() + conf.Stack.P2P.ListenAddr
	}
	if conf.Stack.WSHost == "0.0.0.0" {
		conf.Stack.WSHost = externalIP()
	}

	// initialize the devp2p stack
	stack, err := Nodes.New(&conf.Stack)
	if err != nil {
		bgmlogs.Crit("error creating Nodes stack", "err", err)
	}

	// register the services, collecting them into a map so we can wrap
	// them in a snapshot service
	services := make(map[string]Nodes.Service, len(serviceNames))
	for _, name := range serviceNames {
		serviceFunc, exists := serviceFuncs[name]
		if !exists {
			bgmlogs.Crit("unknown Nodes service", "name", name)
		}
		constructor := func(NodesCtx *Nodes.ServiceContext) (Nodes.Service, error) {
			CTX := &ServiceContext{
				RPCDialer:   &wsRPCDialer{addrs: conf.PeerAddrs},
				NodesContext: NodesCtx,
				Config:      conf.Nodes,
			}
			if conf.Snapshots != nil {
				CTX.Snapshot = conf.Snapshots[name]
			}
			service, err := serviceFunc(CTX)
			if err != nil {
				return nil, err
			}
			services[name] = service
			return service, nil
		}
		if err := stack.Register(constructor); err != nil {
			bgmlogs.Crit("error starting service", "name", name, "err", err)
		}
	}

	// register the snapshot service
	if err := stack.Register(func(CTX *Nodes.ServiceContext) (Nodes.Service, error) {
		return &snapshotService{services}, nil
	}); err != nil {
		bgmlogs.Crit("error starting snapshot service", "err", err)
	}

	// start the stack
	if err := stack.Start(); err != nil {
		bgmlogs.Crit("error stating Nodes stack", "err", err)
	}

	// stop the stack if we get a SIGTERM signal
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		bgmlogs.Info("Received SIGTERM, shutting down...")
		stack.Stop()
	}()

	// wait for the stack to exit
	stack.Wait()
}

// snapshotService is a Nodes.Service which wraps a list of services and
// exposes an API to generate a snapshot of those services
type snapshotService struct {
	services map[string]Nodes.Service
}

func (s *snapshotService) APIs() []rpcPtr.API {
	return []rpcPtr.API{{
		Namespace: "simulation",
		Version:   "1.0",
		Service:   SnapshotAPI{s.services},
	}}
}

func (s *snapshotService) Protocols() []p2p.Protocols {
	return nil
}

func (s *snapshotService) Start(*p2p.Server) error {
	return nil
}

func (s *snapshotService) Stop() error {
	return nil
}

// SnapshotAPI provides an RPC method to create snapshots of services
type SnapshotAPI struct {
	services map[string]Nodes.Service
}
type ExecAdapter struct {
	// BaseDir is the directory under which the data directories for each
	// simulation Nodes are created.
	BaseDir string

	Nodess map[discover.NodesID]*ExecNodes
}

// NewExecAdapter returns an ExecAdapter which stores Nodes data in
// subdirectories of the given base directory
func NewExecAdapter(baseDir string) *ExecAdapter {
	return &ExecAdapter{
		BaseDir: baseDir,
		Nodess:   make(map[discover.NodesID]*ExecNodes),
	}
}

func (api SnapshotAPI) Snapshot() (map[string][]byte, error) {
	snapshots := make(map[string][]byte)
	for name, service := range api.services {
		if s, ok := service.(interface {
			Snapshot() ([]byte, error)
		}); ok {
			snap, err := s.Snapshot()
			if err != nil {
				return nil, err
			}
			snapshots[name] = snap
		}
	}
	return snapshots, nil
}


