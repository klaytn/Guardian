// Copyright 2019 The klaytn Authors
// This file is part of the klaytn library.
//
// The klaytn library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The klaytn library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the klaytn library. If not, see <http://www.gnu.org/licenses/>.

package node

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/klaytn/klaytn/cmd/utils"
	"github.com/klaytn/klaytn/crypto"
	"github.com/klaytn/klaytn/log"
	"github.com/klaytn/klaytn/networks/p2p"
	"github.com/klaytn/klaytn/networks/p2p/discover"
	"github.com/klaytn/klaytn/networks/p2p/nat"
	"github.com/klaytn/klaytn/networks/p2p/netutil"
	"github.com/urfave/cli/v2"
)

const (
	GenerateNodeKeySpecified = iota
	NoPrivateKeyPathSpecified
	NodeKeyDuplicated
	WriteOutAddress
	GoodToGo
)

type GuardianConfig struct {
	// Parameter variables
	networkID    uint64
	addr         string
	genKeyPath   string
	nodeKeyFile  string
	nodeKeyHex   string
	natFlag      string
	netrestrict  string
	writeAddress bool

	// Context
	restrictList *netutil.Netlist
	nodeKey      *ecdsa.PrivateKey
	natm         nat.Interface
	listenAddr   string

	serverConfig p2p.Config

	// Authorized Nodes are used as pre-configured nodes list which are only
	// bonded with this bootnode.
	AuthorizedNodes []*discover.Node

	// DataDir is the file system folder the node should use for any data storage
	// requirements. The configured data directory will not be directly shared with
	// registered services, instead those can use utility methods to create/access
	// databases or flat files. This enables ephemeral nodes which can fully reside
	// in memory.
	DataDir string

	// IPCPath is the requested location to place the IPC endpoint. If the path is
	// a simple file name, it is placed inside the data directory (or on the root
	// pipe path on Windows), whereas if it's a resolvable path name (absolute or
	// relative), then that specific path is enforced. An empty path disables IPC.
	IPCPath string `toml:",omitempty"`

	// Logger is a custom logger to use with the p2p.Server.
	Logger log.Logger `toml:",omitempty"`
}

func NewGuardianConfig(ctx *cli.Context) *GuardianConfig {
	return &GuardianConfig{
		// Config variables
		networkID:    ctx.Uint64(utils.NetworkIdFlag.Name),
		addr:         ctx.String(utils.BNAddrFlag.Name),
		genKeyPath:   ctx.String(utils.GenKeyFlag.Name),
		nodeKeyFile:  ctx.String(utils.NodeKeyFileFlag.Name),
		nodeKeyHex:   ctx.String(utils.NodeKeyHexFlag.Name),
		natFlag:      ctx.String(utils.NATFlag.Name),
		netrestrict:  ctx.String(utils.NetrestrictFlag.Name),
		writeAddress: ctx.Bool(utils.WriteAddressFlag.Name),

		IPCPath: "klay.ipc",
		DataDir: ctx.String(utils.DataDirFlag.Name),

		Logger: log.NewModuleLogger(log.CMDKBN),
	}

}

// splitAndTrim splits input separated by a comma
// and trims excessive white space from the substrings.
func SplitAndTrim(input string) []string {
	result := strings.Split(input, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}
	return result
}

func SetAuthorizedNodes(ctx *cli.Context, cfg *GuardianConfig) {
	if !ctx.IsSet(utils.AuthorizedNodesFlag.Name) {
		return
	}
	urls := ctx.String(utils.AuthorizedNodesFlag.Name)
	splitedUrls := strings.Split(urls, ",")
	cfg.AuthorizedNodes = make([]*discover.Node, 0, len(splitedUrls))
	for _, url := range splitedUrls {
		node, err := discover.ParseNode(url)
		if err != nil {
			logger.Error("URL is invalid", "kni", url, "err", err)
			continue
		}
		cfg.AuthorizedNodes = append(cfg.AuthorizedNodes, node)
	}
}

// setIPC creates an IPC path configuration from the set command line flags,
// returning an empty string if IPC was explicitly disabled, or the set path.
func SetIPC(ctx *cli.Context, cfg *GuardianConfig) {
	utils.CheckExclusive(ctx, utils.IPCDisabledFlag, utils.IPCPathFlag)
	switch {
	case ctx.Bool(utils.IPCDisabledFlag.Name):
		cfg.IPCPath = ""
	case ctx.IsSet(utils.IPCPathFlag.Name):
		cfg.IPCPath = ctx.String(utils.IPCPathFlag.Name)
	}
}

func SetP2PConfig(ctx *cli.Context, cfg *GuardianConfig) {
	utils.SetP2PConfig(ctx, &cfg.serverConfig)
}

func (cfg *GuardianConfig) CheckCMDState() int {
	if cfg.genKeyPath != "" {
		return GenerateNodeKeySpecified
	}
	if cfg.nodeKeyFile == "" && cfg.nodeKeyHex == "" {
		return NoPrivateKeyPathSpecified
	}
	if cfg.nodeKeyFile != "" && cfg.nodeKeyHex != "" {
		return NodeKeyDuplicated
	}
	if cfg.writeAddress {
		return WriteOutAddress
	}
	return GoodToGo
}

func (cfg *GuardianConfig) GenerateNodeKey() {
	nodeKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("could not generate key: %v", err)
	}
	if err = crypto.SaveECDSA(cfg.genKeyPath, nodeKey); err != nil {
		log.Fatalf("%v", err)
	}
	os.Exit(0)
}

func (cfg *GuardianConfig) DoWriteOutAddress() {
	err := cfg.ReadNodeKey()
	if err != nil {
		log.Fatalf("Failed to read node key: %v", err)
	}
	fmt.Printf("%v\n", discover.PubkeyID(&(cfg.nodeKey).PublicKey))
	os.Exit(0)
}

func (cfg *GuardianConfig) ReadNodeKey() error {
	var err error
	if cfg.nodeKeyFile != "" {
		cfg.nodeKey, err = crypto.LoadECDSA(cfg.nodeKeyFile)
		return err
	}
	if cfg.nodeKeyHex != "" {
		cfg.nodeKey, err = crypto.HexToECDSA(cfg.nodeKeyHex)
		return err
	}
	return nil
}

func (cfg *GuardianConfig) ValidateNetworkParameter() error {
	var err error
	if cfg.natFlag != "" {
		cfg.natm, err = nat.Parse(cfg.natFlag)
		if err != nil {
			return err
		}
	}

	if cfg.netrestrict != "" {
		cfg.restrictList, err = netutil.ParseNetlist(cfg.netrestrict)
		if err != nil {
			return err
		}
	}

	if cfg.addr[0] != ':' {
		cfg.listenAddr = ":" + cfg.addr
	} else {
		cfg.listenAddr = cfg.addr
	}

	return nil
}

// IPCEndpoint resolves an IPC endpoint based on a configured value, taking into
// account the set data folders as well as the designated platform we're currently
// running on.
func (c *GuardianConfig) IPCEndpoint() string {
	// Short circuit if IPC has not been enabled
	if c.IPCPath == "" {
		return ""
	}
	// On windows we can only use plain top-level pipes
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(c.IPCPath, `\\.\pipe\`) {
			return c.IPCPath
		}
		return `\\.\pipe\` + c.IPCPath
	}
	// Resolve names into the data directory full paths otherwise
	if filepath.Base(c.IPCPath) == c.IPCPath {
		if c.DataDir == "" {
			return filepath.Join(os.TempDir(), c.IPCPath)
		}
		return filepath.Join(c.DataDir, c.IPCPath)
	}
	return c.IPCPath
}

func DefaultIPCEndpoint(clientIdentifier string) string {
	if clientIdentifier == "" {
		clientIdentifier = strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
		if clientIdentifier == "" {
			panic("empty executable name")
		}
	}
	config := &GuardianConfig{DataDir: os.TempDir(), IPCPath: clientIdentifier + ".ipc"}
	return config.IPCEndpoint()
}
