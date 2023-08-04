// Modifications Copyright 2019 The klaytn Authors
// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.
//
// This file is derived from node/node.go (2018/06/04).
// Modified and improved for the klaytn development.

package node

import (
	"net"
	"sync"

	"github.com/klaytn/klaytn/log"
	"github.com/klaytn/klaytn/networks/p2p"
	"github.com/klaytn/klaytn/networks/p2p/nat"
	"github.com/klaytn/klaytn/networks/rpc"
	"github.com/klaytn/klaytn/node"
)

var logger = log.NewModuleLogger(log.Node)

// Node is a container on which services can be registered.
type Node struct {
	config *GuardianConfig

	server p2p.Server

	rpcAPIs       []rpc.API
	inprocHandler *rpc.Server // In-process RPC request handler to process the API requests

	ipcEndpoint string       // IPC endpoint to listen at (empty = IPC disabled)
	ipcListener net.Listener // IPC RPC listener socket to serve API requests
	ipcHandler  *rpc.Server  // IPC RPC request handler to process the API requests

	stop chan struct{} // Channel to wait for termination notifications
	lock sync.RWMutex

	logger log.Logger
}

// New creates a new P2P node, ready for protocol registration.
func New(conf *GuardianConfig) (*Node, error) {
	// Copy config and resolve the datadir so future changes to the current
	// working directory don't affect the node.
	confCopy := *conf
	conf = &confCopy

	// Note: any interaction with Config that would create/touch files
	// in the data directory or instance directory is delayed until Start.
	return &Node{
		config:      conf,
		ipcEndpoint: conf.IPCEndpoint(),
		logger:      conf.Logger,
	}, nil
}

func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.server != nil {
		return ErrNodeRunning
	}

	addr, err := net.ResolveUDPAddr("udp", n.config.listenAddr)
	if err != nil {
		log.Fatalf("Failed to ResolveUDPAddr: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Failed to ListenUDP: %v", err)
	}

	realaddr := conn.LocalAddr().(*net.UDPAddr)
	if n.config.natm != nil {
		if !realaddr.IP.IsLoopback() {
			go nat.Map(n.config.natm, nil, "udp", realaddr.Port, realaddr.Port, "Klaytn node discovery")
		}
		// TODO: react to external IP changes over time.
		if ext, err := n.config.natm.ExternalIP(); err == nil {
			realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
		}
	}

	n.server = p2p.NewServer(n.config.serverConfig)
	n.logger.Info("Starting peer-to-peer node", "instance", n.config.serverConfig.Name)

	if err := n.server.Start(); err != nil {
		return convertFileLockError(err)
	}

	n.appendAPIs(n.APIs())

	// Lastly start the configured RPC interfaces
	if err := n.startRPC(); err != nil {
		return err
	}

	// Finish initializing the startup
	n.stop = make(chan struct{})

	return nil
}

// startRPC is a helper method to start all the various RPC endpoint during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (n *Node) startRPC() error {
	apis := n.apis()
	// Start the various API endpoints, terminating all in case of errors
	if err := n.startInProc(apis); err != nil {
		return err
	}
	if err := n.startIPC(apis); err != nil {
		n.stopInProc()
		return err
	}

	// All API endpoints started successfully
	n.rpcAPIs = apis

	return nil
}

// startInProc initializes an in-process RPC endpoint.
func (n *Node) startInProc(apis []rpc.API) error {
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		n.logger.Debug("InProc registered", "service", api.Service, "namespace", api.Namespace)
	}
	n.inprocHandler = handler
	return nil
}

// stopInProc terminates the in-process RPC endpoint.
func (n *Node) stopInProc() {
	if n.inprocHandler != nil {
		n.inprocHandler.Stop()
		n.inprocHandler = nil
	}
}

// startIPC initializes and starts the IPC RPC endpoint.
func (n *Node) startIPC(apis []rpc.API) error {
	if n.ipcEndpoint == "" {
		return nil // IPC disabled.
	}
	listener, handler, err := rpc.StartIPCEndpoint(n.ipcEndpoint, apis)
	if err != nil {
		return err
	}
	n.ipcListener = listener
	n.ipcHandler = handler
	n.logger.Info("IPC endpoint opened", "url", n.ipcEndpoint)
	return nil
}

// stopIPC terminates the IPC RPC endpoint.
func (n *Node) stopIPC() {
	if n.ipcListener != nil {
		n.ipcListener.Close()
		n.ipcListener = nil

		n.logger.Info("IPC endpoint closed", "endpoint", n.ipcEndpoint)
	}
	if n.ipcHandler != nil {
		n.ipcHandler.Stop()
		n.ipcHandler = nil
	}
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Terminate the API, services and the p2p server.
	n.stopIPC()
	n.rpcAPIs = nil

	// unblock n.Wait
	close(n.stop)

	return nil
}

// Wait blocks the thread until the node is stopped. If the node is not running
// at the time of invocation, the method immediately returns.
func (n *Node) Wait() {
	n.lock.RLock()
	stop := n.stop
	n.lock.RUnlock()

	<-stop
}

// Restart terminates a running node and boots up a new one in its place. If the
// node isn't running, an error is returned.
func (n *Node) Restart() error {
	if err := n.Stop(); err != nil {
		return err
	}
	if err := n.Start(); err != nil {
		return err
	}
	return nil
}

// Attach creates an RPC client attached to an in-process API handler.
func (n *Node) Attach() (*rpc.Client, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return rpc.DialInProc(n.inprocHandler), nil
}

// RPCHandler returns the in-process RPC request handler.
func (n *Node) RPCHandler() (*rpc.Server, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	if n.inprocHandler == nil {
		return nil, node.ErrNodeStopped
	}
	return n.inprocHandler, nil
}

// DataDir retrieves the current datadir used by the protocol stack.
// Deprecated: No files should be stored in this directory, use InstanceDir instead.
func (n *Node) DataDir() string {
	return n.config.DataDir
}

// IPCEndpoint retrieves the current IPC endpoint used by the protocol stack.
func (n *Node) IPCEndpoint() string {
	return n.ipcEndpoint
}

func (n *Node) appendAPIs(apis []rpc.API) {
	n.rpcAPIs = append(n.rpcAPIs, apis...)
}

func (n *Node) apis() []rpc.API {
	return n.rpcAPIs
}

// Server retrieves the currently running P2P network layer. This method is meant
// only to inspect fields of the currently running server, life cycle management
// should be left to this Node entity.
func (n *Node) Server() p2p.Server {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return n.server
}

func (n *Node) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateGuardianAdminAPI(n),
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPublicGuardianAdminAPI(n),
			Public:    true,
		},
	}
}
