package service

import (
	"github.com/klaytn/klaytn/networks/p2p"
)

type Service interface {
	// Start starts the service.
	Start() error

	// Stop stops the service.
	Stop() error

	// HandleMsg handles the message.
	HandleMsg(p2p.Msg, chan error)
}
