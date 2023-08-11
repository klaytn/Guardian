package protocol

import "errors"

var errAlreadyRegistered = errors.New("peer is already registered")

type PeerSet interface {
	Register(peer Peer) error
	Remove(id string)
}

type peerSet struct {
	peers map[string]Peer
}

func NewPeerSet() PeerSet {
	return &peerSet{
		peers: make(map[string]Peer),
	}
}

func (ps *peerSet) Register(peer Peer) error {
	if _, ok := ps.peers[peer.GetID()]; ok {
		return errAlreadyRegistered
	}

	// TODO-Guardian: validate peer
	return nil
}

func (ps *peerSet) Remove(id string) {
	delete(ps.peers, id)
}
