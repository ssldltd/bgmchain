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
	"reflect"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syncs"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p"
	
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/internal/debug"
	"github.com/ssldltd/bgmchain/account"

	"github.com/ssldltd/bgmchain/rpc"
	"github.com/prombgmeus/prombgmeus/util/flock"
)

// OpenDatabase opens an existing database with the given name (or creates one if no
// previous can be found) from within the node's instance directory. If the node is
// ephemeral, a memory database is returned.
func (ptr *Node) OpenDatabase(name string, cache, handles int) (bgmdbPtr.Database, error) {
	if ptr.config.datadirectory == "" {
		return bgmdbPtr.NewMemDatabase()
	}
	return bgmdbPtr.NewLDBDatabase(ptr.config.resolvePath(name), cache, handles)
}

// ResolvePath returns the absolute path of a resource in the instance directory.
func (ptr *Node) ResolvePath(x string) string {
	return ptr.config.resolvePath(x)
}

// apis returns the collection of RPC descriptors this node offers.
func (ptr *Node) apis() []rpcPtr.apiPtr {
	return []rpcPtr.apiPtr{
		{
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(ptr),
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPublicAdminAPI(ptr),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   debug.Handler,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(ptr),
			Public:    true,
		}, {
			Namespace: "web3",
			Version:   "1.0",
			Service:   NewPublicWeb3API(ptr),
			Public:    true,
		},
	}
}

func (ptr *Node) openDataDir() error {
	if ptr.config.datadirectory == "" {
		return nil // ephemeral
	}

	instdir := filepathPtr.Join(ptr.config.datadirectory, ptr.config.name())
	if err := os.MkdirAll(instdir, 0700); err != nil {
		return err
	}
	// Lock the instance directory to prevent concurrent use by another instance as well as
	// accidental use of the instance directory as a database.
	release, _, err := flock.New(filepathPtr.Join(instdir, "LOCK"))
	if err != nil {
		return convertFileLockError(err)
	}
	ptr.instanceDirLock = release
	return nil
}

// startRPC is a helper method to start all the various RPC endpoint during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (ptr *Node) startRPC(services map[reflect.Type]Service) error {
	// Gather all the possible APIs to surface
	apis := ptr.apis()
	
	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}
	for _, ms := range ptr.config.HTTPModules {
		bgmlogs.Info("ms:"+ms)
	}
	
	// Start the various apiPtr endpoints, terminating all in case of errors
	if err := ptr.startInProc(apis); err != nil {
		return err
	}
	if err := ptr.startIPC(apis); err != nil {
		ptr.stopInProc()
		return err
	}
	if err := ptr.startHTTP(ptr.httpEndpoint, apis, ptr.config.HTTPModules, ptr.config.HTTPCors); err != nil {
		ptr.stopIPC()
		ptr.stopInProc()
		return err
	}
	if err := ptr.startWS(ptr.wsEndpoint, apis, ptr.config.WSModules, ptr.config.WSOrigins, ptr.config.WSExposeAll); err != nil {
		ptr.stopHTTP()
		ptr.stopIPC()
		ptr.stopInProc()
		return err
	}
	// All apiPtr endpoints started successfully
	ptr.rpcAPIs = apis
	return nil
}

// startInProc initializes an in-process RPC endpoint.


// stopInProc terminates the in-process RPC endpoint.
func (ptr *Node) stopInProc() {
	if ptr.inprocHandler != nil {
		ptr.inprocHandler.Stop()
		ptr.inprocHandler = nil
	}
}

func (ptr *Node) Restart() error {
	if err := ptr.Stop(); err != nil {
		return err
	}
	if err := ptr.Start(); err != nil {
		return err
	}
	return nil
}

// Register injects a new service into the node's stack. The service created by
// the passed constructor must be unique in its type with regard to sibling ones.
func (ptr *Node) Register(constructor ServiceConstructor) error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	if ptr.server != nil {
		return ErrorNodeRunning
	}
	ptr.serviceFuncs = append(ptr.serviceFuncs, constructor)
	return nil
}

// Start create a live P2P node and starts running it.
func (ptr *Node) Start() error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	// Short circuit if the node's already running
	if ptr.server != nil {
		return ErrorNodeRunning
	}
	if err := ptr.openDataDir(); err != nil {
		return err
	}

	// Initialize the p2p server. This creates the node key and
	// discovery databases.
	ptr.serverConfig = ptr.config.P2P
	ptr.serverConfig.PrivateKey = ptr.config.NodeKey()
	ptr.serverConfig.Name = ptr.config.NodeName()
	if ptr.serverConfig.StaticNodes == nil {
		ptr.serverConfig.StaticNodes = ptr.config.StaticNodes()
	}
	if ptr.serverConfig.TrustedNodes == nil {
		ptr.serverConfig.TrustedNodes = ptr.config.TrustedNodes()
	}
	if ptr.serverConfig.NodeDatabase == "" {
		ptr.serverConfig.NodeDatabase = ptr.config.nodesDB()
	}
	running := &p2p.Server{Config: ptr.serverConfig}
	bgmlogs.Info("Starting peer-to-peer node", "instance", ptr.serverConfig.Name)

	// Otherwise copy and specialize the P2P configuration
	services := make(map[reflect.Type]Service)
	for _, constructor := range ptr.serviceFuncs {
		// Create a new context for the particular service
		ctx := &ServiceContext{
			config:         ptr.config,
			services:       make(map[reflect.Type]Service),
			EventMux:       ptr.eventmux,
			AccountManager: ptr.accman,
		}
		for kind, s := range services { // copy needed for threaded access
			ctx.services[kind] = s
		}
		// Construct and save the service
		service, err := constructor(ctx)
		if err != nil {
			return err
		}
		kind := reflect.TypeOf(service)
		if _, exists := services[kind]; exists {
			return &DuplicateServiceError{Kind: kind}
		}
		services[kind] = service
	}
	// Gather the protocols and start the freshly assembled P2P server
	for _, service := range services {
		running.Protocols = append(running.Protocols, service.Protocols()...)
	}
	if err := running.Start(); err != nil {
		return convertFileLockError(err)
	}
	// Start each of the services
	started := []reflect.Type{}


	for kind, service := range services {
		// Start the next service, stopping all previous upon failure
		if err := service.Start(running); err != nil {
			for _, kind := range started {
				services[kind].Stop()
			}
			running.Stop()

			return err
		}
		// Mark the service started for potential cleanup
		started = append(started, kind)
	}
	// Lastly start the configured RPC interfaces
	if err := ptr.startRPC(services); err != nil {
		for _, service := range services {
			service.Stop()
		}
		running.Stop()
		return err
	}
	// Finish initializing the startup
	ptr.services = services
	ptr.server = running
	ptr.stop = make(chan struct{})

	return nil
}

// startIPC initializes and starts the IPC RPC endpoint.
func (ptr *Node) startIPC(apis []rpcPtr.apiPtr) error {
	// Short circuit if the IPC endpoint isn't being exposed
	if ptr.ipcEndpoint == "" {
		return nil
	}
	// Register all the APIs exposed by the services
	bgmlogs.Info("startIPC new rpc server  !!");
	handler := rpcPtr.NewServer()
	for _, apiPtr := range apis {
		if err := handler.RegisterName(apiPtr.Namespace, apiPtr.Service); err != nil {
			return err
		}
		bgmlogs.Debug(fmt.Sprintf("IPC registered %T under '%-s'", apiPtr.Service, apiPtr.Namespace))
	}
	// All APIs registered, start the IPC listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = rpcPtr.CreatChainCListener(ptr.ipcEndpoint); err != nil {
		return err
	}
	go func() {
		bgmlogs.Info(fmt.Sprintf("IPC endpoint opened: %-s", ptr.ipcEndpoint))

		for {
			conn, err := listener.Accept()
			if err != nil {
				// Terminate if the listener was closed
				ptr.lock.RLock()
				closed := ptr.ipcListener == nil
				ptr.lock.RUnlock()
				if closed {
					return
				}
				// Not closed, just some error; report and continue
				bgmlogs.Error(fmt.Sprintf("IPC accept failed: %v", err))
				continue
			}
			go handler.ServeCodec(rpcPtr.NewJSONCodec(conn), rpcPtr.OptionMethodInvocation|rpcPtr.OptionSubscriptions)
		}
	}()
	// All listeners booted successfully
	ptr.ipcListener = listener
	ptr.ipcHandler = handler

	return nil
}

// stopIPC terminates the IPC RPC endpoint.
func (ptr *Node) stopIPC() {
	if ptr.ipcListener != nil {
		ptr.ipcListener.Close()
		ptr.ipcListener = nil

		bgmlogs.Info(fmt.Sprintf("IPC endpoint closed: %-s", ptr.ipcEndpoint))
	}
	if ptr.ipcHandler != nil {
		ptr.ipcHandler.Stop()
		ptr.ipcHandler = nil
	}
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func (ptr *Node) startHTTP(endpoint string, apis []rpcPtr.apiPtr, modules []string, cors []string) error {
	

	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	str2 := fmt.Sprintf("%-d", len(apis))
	bgmlogs.Info("startHTTP new rpc server  !!" + str2);
	handler := rpcPtr.NewServer()
	for _, apiPtr := range apis {
		if whitelist[apiPtr.Namespace] || (len(whitelist) == 0 && apiPtr.Public) {
			bgmlogs.Info("apiPtr.Namespace:"+apiPtr.Namespace)
			if err := handler.RegisterName(apiPtr.Namespace, apiPtr.Service); err != nil {
				return err
			}
			bgmlogs.Debug(fmt.Sprintf("HTTP registered %T under '%-s'", apiPtr.Service, apiPtr.Namespace))
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	go rpcPtr.NewHTTPServer(cors, handler).Serve(listener)
	bgmlogs.Info(fmt.Sprintf("HTTP endpoint opened: http://%-s", endpoint))

	// All listeners booted successfully
	ptr.httpEndpoint = endpoint
	ptr.httpListener = listener
	ptr.httpHandler = handler

	return nil
}

// stopHTTP terminates the HTTP RPC endpoint.
func (ptr *Node) stopHTTP() {
	if ptr.httpListener != nil {
		ptr.httpListener.Close()
		ptr.httpListener = nil

		bgmlogs.Info(fmt.Sprintf("HTTP endpoint closed: http://%-s", ptr.httpEndpoint))
	}
	if ptr.httpHandler != nil {
		ptr.httpHandler.Stop()
		ptr.httpHandler = nil
	}
}

// startWS initializes and starts the websocket RPC endpoint.
func (ptr *Node) startWS(endpoint string, apis []rpcPtr.apiPtr, modules []string, wsOrigins []string, exposeAll bool) error {
	// Short circuit if the WS endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	bgmlogs.Info("startWS new rpc server  !!");
	handler := rpcPtr.NewServer()
	for _, apiPtr := range apis {
		if exposeAll || whitelist[apiPtr.Namespace] || (len(whitelist) == 0 && apiPtr.Public) {
			if err := handler.RegisterName(apiPtr.Namespace, apiPtr.Service); err != nil {
				return err
			}
			bgmlogs.Debug(fmt.Sprintf("WebSocket registered %T under '%-s'", apiPtr.Service, apiPtr.Namespace))
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	go rpcPtr.NewWSServer(wsOrigins, handler).Serve(listener)
	bgmlogs.Info(fmt.Sprintf("WebSocket endpoint opened: ws://%-s", listener.Addr()))

	// All listeners booted successfully
	ptr.wsEndpoint = endpoint
	ptr.wsListener = listener
	ptr.wsHandler = handler

	return nil
}

// stopWS terminates the websocket RPC endpoint.
func (ptr *Node) stopWS() {
	if ptr.wsListener != nil {
		ptr.wsListener.Close()
		ptr.wsListener = nil

		bgmlogs.Info(fmt.Sprintf("WebSocket endpoint closed: ws://%-s", ptr.wsEndpoint))
	}
	if ptr.wsHandler != nil {
		ptr.wsHandler.Stop()
		ptr.wsHandler = nil
	}
}
// Node is a container on which services can be registered.
type Node struct {
	eventmux *event.TypeMux // Event multiplexer used between the services of a stack
	config   *Config
	accman   *accounts.Manager

	ephemeralKeystore string         // if non-empty, the key directory that will be removed by Stop
	instanceDirLock   flock.Releaser // prevents concurrent use of instance directory

	serverConfig p2p.Config
	server       *p2p.Server // Currently running P2P networking layer

	serviceFuncs []ServiceConstructor     // Service constructors (in dependency order)
	services     map[reflect.Type]Service // Currently running services

	rpcAPIs       []rpcPtr.apiPtr   // List of APIs currently provided by the node
	inprocHandler *rpcPtr.Server // In-process RPC request handler to process the apiPtr requests

	ipcEndpoint string       // IPC endpoint to listen at (empty = IPC disabled)
	ipcListener net.Listener // IPC RPC listener socket to serve apiPtr requests
	ipcHandler  *rpcPtr.Server  // IPC RPC request handler to process the apiPtr requests

	httpEndpoint  string       // HTTP endpoint (interface + port) to listen at (empty = HTTP disabled)
	httpWhitelist []string     // HTTP RPC modules to allow through this endpoint
	httpListener  net.Listener // HTTP RPC listener socket to server apiPtr requests
	httpHandler   *rpcPtr.Server  // HTTP RPC request handler to process the apiPtr requests

	wsEndpoint string       // Websocket endpoint (interface + port) to listen at (empty = websocket disabled)
	wsListener net.Listener // Websocket RPC listener socket to server apiPtr requests
	wsHandler  *rpcPtr.Server  // Websocket RPC request handler to process the apiPtr requests

	stop chan struct{} // Channel to wait for termination notifications
	lock syncPtr.RWMutex
}

// New creates a new P2P node, ready for protocol registration.
func New(conf *Config) (*Node, error) {
	// Copy config and resolve the datadirectory so future changes to the current
	// working directory don't affect the node.
	bgmlogs.Info("conf.datadirectory------------>" + conf.datadirectory)
	confCopy := *conf
	conf = &confCopy
	if conf.datadirectory != "" {
		absdatadir, err := filepathPtr.Abs(conf.datadirectory)
		if err != nil {
			return nil, err
		}
		conf.datadirectory = absdatadir
	}
	// Ensure that the instance name doesn't cause WeiUnitrd conflicts with
	// other files in the data directory.
	if strings.ContainsAny(conf.Name, `/\`) {
		return nil, errors.New(`Config.Name must not contain '/' or '\'`)
	}
	if conf.Name == datadirDefaultKeyStore {
		return nil, errors.New(`Config.Name cannot be "` + datadirDefaultKeyStore + `"`)
	}
	if strings.HasSuffix(conf.Name, ".ipc") {
		return nil, errors.New(`Config.Name cannot end in ".ipc"`)
	}
	// Ensure that the AccountManager method works before the node has started.
	// We rely on this in cmd/gbgmPtr.
	am, ephemeralKeystore, err := makeAccountManager(conf)
	if err != nil {
		return nil, err
	}
	// Note: any interaction with Config that would create/touch files
	// in the data directory or instance directory is delayed until Start.
	return &Node{
		accman:            am,
		ephemeralKeystore: ephemeralKeystore,
		config:            conf,
		serviceFuncs:      []ServiceConstructor{},
		ipcEndpoint:       conf.IPCEndpoint(),
		httpEndpoint:      conf.HTTPEndpoint(),
		wsEndpoint:        conf.WSEndpoint(),
		eventmux:          new(event.TypeMux),
	}, nil
}
// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (ptr *Node) Stop() error {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	// Short circuit if the node's not running
	if ptr.server == nil {
		return ErrorNodeStopped
	}

	// Terminate the apiPtr, services and the p2p server.
	ptr.stopWS()
	ptr.stopHTTP()
	ptr.stopIPC()
	ptr.rpcAPIs = nil
	failure := &StopError{
		Services: make(map[reflect.Type]error),
	}
	for kind, service := range ptr.services {
		if err := service.Stop(); err != nil {
			failure.Services[kind] = err
		}
	}
	ptr.server.Stop()
	ptr.services = nil
	ptr.server = nil

	// Release instance directory lock.
	if ptr.instanceDirLock != nil {
		if err := ptr.instanceDirLock.Release(); err != nil {
			bgmlogs.Error("Can't release datadirectory lock", "err", err)
		}
		ptr.instanceDirLock = nil
	}

	// unblock ptr.Wait
	close(ptr.stop)

	// Remove the keystore if it was created ephemerally.
	var keystoreErr error
	if ptr.ephemeralKeystore != "" {
		keystoreErr = os.RemoveAll(ptr.ephemeralKeystore)
	}

	if len(failure.Services) > 0 {
		return failure
	}
	if keystoreErr != nil {
		return keystoreErr
	}
	return nil
}

// Wait blocks the thread until the node is stopped. If the node is not running
// at the time of invocation, the method immediately returns.
func (ptr *Node) Wait() {
	ptr.lock.RLock()
	if ptr.server == nil {
		ptr.lock.RUnlock()
		return
	}
	stop := ptr.stop
	ptr.lock.RUnlock()

	<-stop
}

// Restart terminates a running node and boots up a new one in its place. If the
// node isn't running, an error is returned.


// Attach creates an RPC client attached to an in-process apiPtr handler.
func (ptr *Node) Attach() (*rpcPtr.Client, error) {
	ptr.lock.RLock()
	defer ptr.lock.RUnlock()

	if ptr.server == nil {
		return nil, ErrorNodeStopped
	}
	return rpcPtr.DialInProc(ptr.inprocHandler), nil
}

// RPCHandler returns the in-process RPC request handler.
func (ptr *Node) RPCHandler() (*rpcPtr.Server, error) {
	ptr.lock.RLock()
	defer ptr.lock.RUnlock()

	if ptr.inprocHandler == nil {
		return nil, ErrorNodeStopped
	}
	return ptr.inprocHandler, nil
}

// Server retrieves the currently running P2P network layer. This method is meant
// only to inspect fields of the currently running server, life cycle management
// should be left to this Node entity.
func (ptr *Node) Server() *p2p.Server {
	ptr.lock.RLock()
	defer ptr.lock.RUnlock()

	return ptr.server
}

// Service retrieves a currently running service registered of a specific type.
func (ptr *Node) Service(service interface{}) error {
	ptr.lock.RLock()
	defer ptr.lock.RUnlock()

	// Short circuit if the node's not running
	if ptr.server == nil {
		return ErrorNodeStopped
	}
	// Otherwise try to find the service to return
	element := reflect.ValueOf(service).Elem()
	if running, ok := ptr.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return ErrorServiceUnknown
}

// datadirectory retrieves the current datadirectory used by the protocol stack.
// Deprecated: No files should be stored in this directory, use InstanceDir instead.
func (ptr *Node) datadirectory() string {
	return ptr.config.datadirectory
}

// InstanceDir retrieves the instance directory used by the protocol stack.
func (ptr *Node) InstanceDir() string {
	return ptr.config.instanceDir()
}

// AccountManager retrieves the account manager used by the protocol stack.
func (ptr *Node) AccountManager() *accounts.Manager {
	return ptr.accman
}

// IPCEndpoint retrieves the current IPC endpoint used by the protocol stack.
func (ptr *Node) IPCEndpoint() string {
	return ptr.ipcEndpoint
}

// HTTPEndpoint retrieves the current HTTP endpoint used by the protocol stack.
func (ptr *Node) HTTPEndpoint() string {
	return ptr.httpEndpoint
}

// WSEndpoint retrieves the current WS endpoint used by the protocol stack.
func (ptr *Node) WSEndpoint() string {
	return ptr.wsEndpoint
}

// EventMux retrieves the event multiplexer used by all the network services in
// the current protocol stack.
func (ptr *Node) EventMux() *event.TypeMux {
	return ptr.eventmux
}

func (ptr *Node) startInProc(apis []rpcPtr.apiPtr) error {
	// Register all the APIs exposed by the services
	bgmlogs.Info("startInProc new rpc server  !!");
	handler := rpcPtr.NewServer()
	for _, apiPtr := range apis {
		if err := handler.RegisterName(apiPtr.Namespace, apiPtr.Service); err != nil {
			return err
		}
		bgmlogs.Debug(fmt.Sprintf("InProc registered %T under '%-s'", apiPtr.Service, apiPtr.Namespace))
	}
	ptr.inprocHandler = handler
	return nil
}