package protocol

import (
	"math/big"

	"github.com/klaytn/guardian/protocol/service"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/networks/p2p"
)

type ProtocolManager struct {
	networkId uint64
	chainId   *big.Int

	peers     PeerSet
	protocols []p2p.Protocol
	services  []service.Service
}

func NewProtocolManager(services ...service.Service) *ProtocolManager {
	pm := &ProtocolManager{
		peers: NewPeerSet(),
	}

	// register services
	for _, service := range services {
		pm.register(service)
	}

	pm.protocols = append(pm.protocols, p2p.Protocol{
		Name:    "guardian",
		Version: 1,
		Length:  23,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			newPeer := NewPeer(p, rw)
			return pm.handle(newPeer)
		},
		RunWithRWs: nil, // TODO-Guardian: implement RunWithRWs
		NodeInfo:   nil,
		PeerInfo:   nil,
	})
	return pm
}

func (pm *ProtocolManager) register(service service.Service) {
	pm.services = append(pm.services, service)
}

func (pm *ProtocolManager) handle(p Peer) error {
	// TODO-Guardian: implement to get genesis hash somehow
	if err := p.Handshake(pm.networkId, pm.getChainId(), nil, common.Hash{}, common.Hash{}); err != nil {
		return err
	}

	if err := pm.peers.Register(p); err != nil {
		return err
	}
	defer pm.removePeer(p.GetID())

	var (
		msgCh = make(chan p2p.Msg)
		errCh = make(chan error)
	)

	go pm.distributeMsgToServices(msgCh, errCh)
	for {
		msg, err := p.ReadMsg()
		if err != nil {
			return err
		}
		select {
		case msgCh <- msg:
		case err := <-errCh:
			return err
		}
	}
}

func (pm *ProtocolManager) distributeMsgToServices(msgCh chan p2p.Msg, errCh chan error) {
	for msg := range msgCh {
		for _, service := range pm.services {
			go service.HandleMsg(msg, errCh)
		}
	}
}

func (pm *ProtocolManager) removePeer(id string) {
	pm.peers.Remove(id)
}

func (pm *ProtocolManager) getChainId() *big.Int {
	return pm.chainId
}

func (pm *ProtocolManager) Protocols() []p2p.Protocol {
	return pm.protocols
}

func (pm *ProtocolManager) Start() error {
	for _, service := range pm.services {
		if err := service.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (pm *ProtocolManager) Stop() error {
	for _, service := range pm.services {
		if err := service.Stop(); err != nil {
			// TODO-Guardian: handle error / logging error
		}
	}
	return nil
}
