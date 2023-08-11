package txpool

import (
	"math/big"

	"github.com/klaytn/guardian/protocol"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/networks/p2p"
)

type txpool struct {
	blocks map[*big.Int]*types.Block
}

func NewTxPool() *txpool {
	return &txpool{}
}

func (h *txpool) Start() error {
	return nil
}

func (h *txpool) HandleMsg(msg p2p.Msg, errCh chan error) {
	switch msg.Code {
	case protocol.StatusMsg:
		// TODO-Guardian: implement to handle status message
	}
}

func (h *txpool) Stop() error {
	return nil
}
