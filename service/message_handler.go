package service

import (
	"github.com/klaytn/guardian/protocol"
	"github.com/klaytn/klaytn/networks/p2p"
)

type MsgHandler struct {
	peer protocol.Peer
}

func NewMsgHandler() *MsgHandler {
	return &MsgHandler{}
}

func (h *MsgHandler) Start() error {
	return nil
}

func (h *MsgHandler) HandleMsg(msg p2p.Msg, errCh chan error) {
	switch msg.Code {
	case protocol.StatusMsg:
		// TODO-Guardian: implement to handle status message
	}
}

func (h *MsgHandler) Stop() error {
	return nil
}
