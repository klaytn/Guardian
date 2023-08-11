package blockchain

import (
	"math/big"

	"github.com/klaytn/guardian/protocol"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/networks/p2p"
)

type Blockchain struct {
	blocks map[*big.Int]*types.Block
}

func NewBlockchain() *Blockchain {
	return &Blockchain{}
}

func (h *Blockchain) Start() error {
	return nil
}

func (h *Blockchain) HandleMsg(msg p2p.Msg, errCh chan error) {
	switch msg.Code {
	case protocol.StatusMsg:
		// TODO-Guardian: implement to handle status message
	}
}

func (h *Blockchain) Stop() error {
	return nil
}
