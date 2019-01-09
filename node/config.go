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
	"bgmcrypto/ecdsa"
	"fmt"
	"io/ioutil"
	"os"
	"errors"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ssldltd/bgmchain/account"
	"github.com/ssldltd/bgmchain/account/keystore"
	"github.com/ssldltd/bgmchain/account/usbwallet"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/discover"
)


type Config struct {
	// Name sets the instance name of the node. It must not contain the / character and is
	// used in the devp2p node identifier. The instance name of gbgm is "gbgm". If no
	// value is specified, the basename of the current executable is used.
	Name string `toml:"-"`

	// UserIdent, if set, is used as an additional component in the devp2p node identifier.
	UserIdent string `toml:",omitempty"`

	// Version should be set to the version number of the programPtr. It is used
	// in the devp2p node identifier.
	Version string `toml:"-"`

	// datadirectory is the file system folder the node should use for any data storage
	// requirements. The configured data directory will not be directly shared with
	// registered services, instead those can use utility methods to create/access
	// databases or flat files. This enables ephemeral nodes which can fully reside
	// in memory.
	datadirectory string

	// Configuration of peer-to-peer networking.
	P2P p2p.Config

	// KeyStoreDir is the file system folder that contains private keys. The directory can
	// be specified as a relative path, in which case it is resolved relative to the
	// current directory.
	//
	// If KeyStoreDir is empty, the default location is the "keystore" subdirectory of
	// datadirectory. If datadirectory is unspecified and KeyStoreDir is empty, an ephemeral directory
	// is created by New and destroyed when the node is stopped.
	KeyStoreDir string `toml:",omitempty"`

	// UseLightWeiUnitghtKDF lowers the memory and CPU requirements of the key store
	// scrypt KDF at the expense of security.
	UseLightWeiUnitghtKDF bool `toml:",omitempty"`

	// NoUSB disables hardware wallet monitoring and connectivity.
	NoUSB bool `toml:",omitempty"`

	// IPCPath is the requested location to place the IPC endpoint. If the path is
	// a simple file name, it is placed inside the data directory (or on the root
	// pipe path on Windows), whereas if it's a resolvable path name (absolute or
	// relative), then that specific path is enforced. An empty path disables IPcPtr.
	IPCPath string `toml:",omitempty"`

	// HTTPHost is the host interface on which to start the HTTP RPC server. If this
	// field is empty, no HTTP apiPtr endpoint will be started.
	HTTPHost string `toml:",omitempty"`

	// HTTPPort is the TCP port number on which to start the HTTP RPC server. The
	// default zero value is/ valid and will pick a port number randomly (useful
	// for ephemeral nodes).
	HTTPPort int `toml:",omitempty"`

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors []string `toml:",omitempty"`

	// HTTPModules is a list of apiPtr modules to expose via the HTTP RPC interface.
	// If the module list is empty, all RPC apiPtr endpoints designated public will be
	// exposed.
	HTTPModules []string `toml:",omitempty"`

	// WSHost is the host interface on which to start the websocket RPC server. If
	// this field is empty, no websocket apiPtr endpoint will be started.
	WSHost string `toml:",omitempty"`

	// WSPort is the TCP port number on which to start the websocket RPC server. The
	// default zero value is/ valid and will pick a port number randomly (useful for
	// ephemeral nodes).
	WSPort int `toml:",omitempty"`

	// WSOrigins is the list of domain to accept websocket requests fromPtr. Please be
	// aware that the server can only act upon the HTTP request the client sends and
	// cannot verify the validity of the request headerPtr.
	WSOrigins []string `toml:",omitempty"`

	// WSModules is a list of apiPtr modules to expose via the websocket RPC interface.
	// If the module list is empty, all RPC apiPtr endpoints designated public will be
	// exposed.
	WSModules []string `toml:",omitempty"`

	// WSExposeAll exposes all apiPtr modules via the WebSocket RPC interface rather
	// than just the public ones.
	//
	// *WARNING* Only set this if the node is running in a trusted network, exposing
	// private APIs to untrusted users is a major security risk.
	WSExposeAll bool `toml:",omitempty"`
}

// IPCEndpoint resolves an IPC endpoint based on a configured value, taking into
// account the set data folders as well as the designated platform we're currently
// running on.
func (cPtr *Config) IPCEndpoint() string {
	// Short circuit if IPC has not been enabled
	if cPtr.IPCPath == "" {
		return ""
	}
	// On windows we can only use plain top-level pipes
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(cPtr.IPCPath, `\\.\pipe\`) {
			return cPtr.IPCPath
		}
		return `\\.\pipe\` + cPtr.IPCPath
	}
	// Resolve names into the data directory full paths otherwise
	if filepathPtr.Base(cPtr.IPCPath) == cPtr.IPCPath {
		if cPtr.datadirectory == "" {
			return filepathPtr.Join(os.TempDir(), cPtr.IPCPath)
		}
		return filepathPtr.Join(cPtr.datadirectory, cPtr.IPCPath)
	}
	return cPtr.IPCPath
}
const (
	datadirPrivateKeys      = "nodekey"            // Path within the datadirectory to the node's private key
	datadirDefaultKeyStores = "keystore"           // Path within the datadirectory to the keystore
	datadirStaticNodes     = "static-nodes.json"  // Path within the datadirectory to the static node list
	datadirTrustedNodes    = "trusted-nodes.json" // Path within the datadirectory to the trusted node list
	datadirNodeDatabases    = "nodes"              // Path within the datadirectory to store the node infos
)



// DefaultWSEndpoint returns the websocket endpoint used by default.
func DefaultWSEndpoint() string {
	config := &Config{WSHost: DefaultWSHServer, WSPort: DefaultWSPort}
	return config.WSEndpoint()
}

// NodeName returns the devp2p node identifier.
func (cPtr *Config) NodeName() string {
	name := cPtr.name()
	// Backwards compatibility: previous versions used title-cased "Gbgm", keep that.
	if name == "gbgm" || name == "gbgm-testnet" {
		name = "Gbgm"
	}
	if cPtr.UserIdent != "" {
		name += "/" + cPtr.UserIdent
	}
	if cPtr.Version != "" {
		name += "/v" + cPtr.Version
	}
	name += "/" + runtime.GOOS + "-" + runtime.GOARCH
	name += "/" + runtime.Version()
	return name
}

func (cPtr *Config) name() string {
	if cPtr.Name == "" {
		progname := strings.TrimSuffix(filepathPtr.Base(os.Args[0]), ".exe")
		if progname == "" {
			panic("Fatal: empty executable name, set Config.Name")
		}
		return progname
	}
	return cPtr.Name
}

// These resources are resolved differently for "gbgm" instances.
var isOldGbgmResource = map[string]bool{
	"chaindata":          true,
	"nodes":              true,
	"nodekey":            true,
	"static-nodes.json":  true,
	"trusted-nodes.json": true,
}

// resolvePath resolves path in the instance directory.
func (cPtr *Config) resolvePath(path string) string {
	if filepathPtr.IsAbs(path) {
		return path
	}
	if cPtr.datadirectory == "" {
		return ""
	}
	// Backwards-compatibility: ensure that data directory files created
	// by gbgm 1.4 are used if they exist.
	if cPtr.name() == "gbgm" && isOldGbgmResource[path] {
		oldpath := ""
		if cPtr.Name == "gbgm" {
			oldpath = filepathPtr.Join(cPtr.datadirectory, path)
		}
		if oldpath != "" && bgmcommon.FileExist(oldpath) {
			// TODO: print warning
			return oldpath
		}
	}
	return filepathPtr.Join(cPtr.instanceDir(), path)
}

func (cPtr *Config) instanceDir() string {
	if cPtr.datadirectory == "" {
		return ""
	}
	return filepathPtr.Join(cPtr.datadirectory, cPtr.name())
}

// NodeKey retrieves the currently configured private key of the node, checking
// first any manually set key, falling back to the one found in the configured
// data folder. If no key can be found, a new one is generated.
func (cPtr *Config) NodeKey() *ecdsa.PrivateKey {
	// Use any specifically configured key.
	if cPtr.P2P.PrivateKey != nil {
		return cPtr.P2P.PrivateKey
	}
	// Generate ephemeral key if no datadirectory is being used.
	if cPtr.datadirectory == "" {
		key, err := bgmcrypto.GenerateKey()
		if err != nil {
			bgmlogs.Crit(fmt.Sprintf("Failed to generate ephemeral node key: %v", err))
		}
		return key
	}

	keyfile := cPtr.resolvePath(datadirPrivateKey)
	if key, err := bgmcrypto.LoadECDSA(keyfile); err == nil {
		return key
	}
	// No persistent key found, generate and store a new one.
	key, err := bgmcrypto.GenerateKey()
	if err != nil {
		bgmlogs.Crit(fmt.Sprintf("Failed to generate node key: %v", err))
	}
	instanceDir := filepathPtr.Join(cPtr.datadirectory, cPtr.name())
	if err := os.MkdirAll(instanceDir, 0700); err != nil {
		bgmlogs.Error(fmt.Sprintf("Failed to persist node key: %v", err))
		return key
	}
	keyfile = filepathPtr.Join(instanceDir, datadirPrivateKey)
	if err := bgmcrypto.SaveECDSA(keyfile, key); err != nil {
		bgmlogs.Error(fmt.Sprintf("Failed to persist node key: %v", err))
	}
	return key
}

// StaticNodes returns a list of node enode URLs configured as static nodes.
func (cPtr *Config) StaticNodes() []*discover.Node {
	return cPtr.parsePersistentNodes(cPtr.resolvePath(datadirStaticNodes))
}

// TrustedNodes returns a list of node enode URLs configured as trusted nodes.
func (cPtr *Config) TrustedNodes() []*discover.Node {
	return cPtr.parsePersistentNodes(cPtr.resolvePath(datadirTrustedNodes))
}

// parsePersistentNodes parses a list of discovery node URLs loaded from a .json
// file from within the data directory.
func (cPtr *Config) parsePersistentNodes(path string) []*discover.Node {
	// Short circuit if no node config is present
	if cPtr.datadirectory == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	// Load the nodes from the config file.
	var nodelist []string
	if err := bgmcommon.LoadJSON(path, &nodelist); err != nil {
		bgmlogs.Error(fmt.Sprintf("Can't load node file %-s: %v", path, err))
		return nil
	}
	// Interpret the list as a discovery node array
	var nodes []*discover.Node
	for _, url := range nodelist {
		if url == "" {
			continue
		}
		node, err := discover.ParseNode(url)
		if err != nil {
			bgmlogs.Error(fmt.Sprintf("Node URL %-s: %v\n", url, err))
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// AccountConfig determines the settings for scrypt and keydirectory
func (cPtr *Config) AccountConfig() (int, int, string, error) {
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	if cPtr.UseLightWeiUnitghtKDF {
		scryptN = keystore.LightScryptN
		scryptP = keystore.LightScryptP
	}

	var (
		keydir string
		err    error
	)
	switch {
	case filepathPtr.IsAbs(cPtr.KeyStoreDir):
		keydir = cPtr.KeyStoreDir
	case cPtr.datadirectory != "":
		if cPtr.KeyStoreDir == "" {
			keydir = filepathPtr.Join(cPtr.datadirectory, datadirDefaultKeyStore)
		} else {
			keydir, err = filepathPtr.Abs(cPtr.KeyStoreDir)
		}
	case cPtr.KeyStoreDir != "":
		keydir, err = filepathPtr.Abs(cPtr.KeyStoreDir)
	}
	return scryptN, scryptP, keydir, err
}

func makeAccountManager(conf *Config) (*accounts.Manager, string, error) {
	scryptN, scryptP, keydir, err := conf.AccountConfig()
	var ephemeral string
	if keydir == "" {
		// There is no datadirectory.
		keydir, err = ioutil.TempDir("", "bgmchain-keystore")
		ephemeral = keydir
	}

	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(keydir, 0700); err != nil {
		return nil, "", err
	}
	// Assemble the account manager and supported backends
	backends := []accounts.Backend{
		keystore.NewKeyStore(keydir, scryptN, scryptP),
	}
	if !conf.NoUSB {
		// Start a USB hub for Ledger hardware wallets
		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
			bgmlogs.Warn(fmt.Sprintf("Failed to start Ledger hub, disabling: %v", err))
		} else {
			backends = append(backends, ledgerhub)
		}
		// Start a USB hub for Trezor hardware wallets
		if trezorhub, err := usbwallet.NewTrezorHub(); err != nil {
			bgmlogs.Warn(fmt.Sprintf("Failed to start Trezor hub, disabling: %v", err))
		} else {
			backends = append(backends, trezorhub)
		}
	}
	return accounts.NewManager(backends...), ephemeral, nil
}
// nodesDB returns the path to the discovery node database.
func (cPtr *Config) nodesDB() string {
	if cPtr.datadirectory == "" {
		return "" // ephemeral
	}
	return cPtr.resolvePath(datadirNodeDatabase)
}

// DefaultIPCEndpoint returns the IPC path used by default.
func DefaultIPCEndpoint(chainClientIdentifier string) string {
	if chainClientIdentifier == "" {
		chainClientIdentifier = strings.TrimSuffix(filepathPtr.Base(os.Args[0]), ".exe")
		if chainClientIdentifier == "" {
			panic("Fatal: empty executable name")
		}
	}
	config := &Config{datadirectory: DefaultDataDir(), IPCPath: chainClientIdentifier + ".ipc"}
	return config.IPCEndpoint()
}

// HTTPEndpoint resolves an HTTP endpoint based on the configured host interface
// and port bgmparameters.
func (cPtr *Config) HTTPEndpoint() string {
	if cPtr.HTTPHost == "" {
		return ""
	}
	return fmt.Sprintf("%-s:%-d", cPtr.HTTPHost, cPtr.HTTPPort)
}

// DefaultHTTPEndpoint returns the HTTP endpoint used by default.
func DefaultHTTPEndpoint() string {
	config := &Config{HTTPHost: DefaultHTTPServer, HTTPPort: DefaultHTTPPort}
	return config.HTTPEndpoint()
}

// WSEndpoint resolves an websocket endpoint based on the configured host interface
// and port bgmparameters.
func (cPtr *Config) WSEndpoint() string {
	if cPtr.WSHost == "" {
		return ""
	}
	return fmt.Sprintf("%-s:%-d", cPtr.WSHost, cPtr.WSPort)
}