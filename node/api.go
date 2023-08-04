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
	"context"
	"fmt"

	"github.com/klaytn/klaytn/networks/p2p"
	"github.com/klaytn/klaytn/networks/p2p/discover"
	"github.com/klaytn/klaytn/networks/rpc"
)

// PublicAdminAPI is the collection of administrative API methods exposed over
// both secure and unsecure RPC channels.
type PublicGuardianAdminAPI struct {
	node *Node // Node interfaced by this API
}

// NewPublicAdminAPI creates a new API definition for the public admin methods
// of the node itself.
func NewPublicGuardianAdminAPI(node *Node) *PublicGuardianAdminAPI {
	return &PublicGuardianAdminAPI{node: node}
}

// Peers retrieves all the information we know about each individual peer at the
// protocol granularity.
func (api *PublicGuardianAdminAPI) Peers() ([]*p2p.PeerInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}
	return server.PeersInfo(), nil
}

// NodeInfo retrieves all the information we know about the host node at the
// protocol granularity.
func (api *PublicGuardianAdminAPI) NodeInfo() (*p2p.NodeInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}
	return server.NodeInfo(), nil
}

// Datadir retrieves the current data directory the node is using.
func (api *PublicGuardianAdminAPI) Datadir() string {
	return api.node.DataDir()
}

// PrivateAdminAPI is the collection of administrative API methods exposed only
// over a secure RPC channel.
type PrivateGuardianAdminAPI struct {
	node *Node // Node interfaced by this API
}

// NewPrivateAdminAPI creates a new API definition for the private admin methods
// of the node itself.
func NewPrivateGuardianAdminAPI(node *Node) *PrivateGuardianAdminAPI {
	return &PrivateGuardianAdminAPI{node: node}
}

// addPeerInternal does common part for AddPeer.
func addPeerInternal(server p2p.Server, url string, onParentChain bool) (*discover.Node, error) {
	// Try to add the url as a static peer and return
	node, err := discover.ParseNode(url)
	if err != nil {
		return nil, fmt.Errorf("invalid kni: %v", err)
	}
	server.AddPeer(node)
	return node, nil
}

// AddPeer requests connecting to a remote node, and also maintaining the new
// connection at all times, even reconnecting if it is lost.
func (api *PrivateGuardianAdminAPI) AddPeer(url string) (bool, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return false, ErrNodeStopped
	}
	// TODO-Klaytn Refactoring this to check whether the url is valid or not by dialing and return it.
	if _, err := addPeerInternal(server, url, false); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

// RemovePeer disconnects from a a remote node if the connection exists
func (api *PrivateGuardianAdminAPI) RemovePeer(url string) (bool, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return false, ErrNodeStopped
	}
	// Try to remove the url as a static peer and return
	node, err := discover.ParseNode(url)
	if err != nil {
		return false, fmt.Errorf("invalid kni: %v", err)
	}
	server.RemovePeer(node)
	return true, nil
}

// PeerEvents creates an RPC subscription which receives peer events from the
// node's p2p.Server
func (api *PrivateGuardianAdminAPI) PeerEvents(ctx context.Context) (*rpc.Subscription, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}

	// Create the subscription
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	go func() {
		events := make(chan *p2p.PeerEvent)
		sub := server.SubscribeEvents(events)
		defer sub.Unsubscribe()

		for {
			select {
			case event := <-events:
				notifier.Notify(rpcSub.ID, event)
			case <-sub.Err():
				return
			case <-rpcSub.Err():
				return
			case <-notifier.Closed():
				return
			}
		}
	}()

	return rpcSub, nil
}
