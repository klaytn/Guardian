package protocol

import (
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/networks/p2p"
	"math/big"
	"time"
)

const handshakeTimeout = 5 * time.Second

type Peer interface {
	Handshake(network uint64, chainID, td *big.Int, head common.Hash, genesis common.Hash) error
	ReadMsg() (p2p.Msg, error)
	WriteMsg(msg p2p.Msg) error
	GetID() string
}

type basePeer struct {
	*p2p.Peer
	rw p2p.MsgReadWriter

	version int
}

func NewPeer(p *p2p.Peer, rw p2p.MsgReadWriter) Peer {
	return &basePeer{
		Peer: p,
		rw:   rw,
	}
}

// Handshake executes the Klaytn protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (p *basePeer) Handshake(network uint64, chainID, td *big.Int, head common.Hash, genesis common.Hash) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)
	var status statusData // safe to read after two values have been received from errc

	go func() {
		errc <- p2p.Send(p.rw, StatusMsg, &statusData{
			ProtocolVersion: uint32(p.version),
			NetworkId:       network,
			TD:              td,
			CurrentBlock:    head,
			GenesisBlock:    genesis,
			ChainID:         chainID,
		})
	}()
	go func() {
		errc <- p.readStatus(network, &status, genesis, chainID)
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return p2p.DiscReadTimeout
		}
	}
	return nil
}

func (p *basePeer) readStatus(network uint64, status *statusData, genesis common.Hash, chainID *big.Int) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != StatusMsg {
		return errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if status.GenesisBlock != genesis {
		return errResp(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock[:8], genesis[:8])
	}
	if status.NetworkId != network {
		return errResp(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, network)
	}
	if status.ChainID.Cmp(chainID) != 0 {
		return errResp(ErrChainIDMismatch, "%v (!= %v)", status.ChainID.String(), chainID.String())
	}
	if int(status.ProtocolVersion) != p.version {
		return errResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
	}
	return nil
}

func (p *basePeer) GetID() string {
	return p.ID().String()
}

func (p *basePeer) ReadMsg() (p2p.Msg, error) {
	return p.rw.ReadMsg()
}

func (p *basePeer) WriteMsg(msg p2p.Msg) error {
	return p.rw.WriteMsg(msg)
}
