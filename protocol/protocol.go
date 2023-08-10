package protocol

import (
	"fmt"
	"github.com/klaytn/klaytn/common"
	"math/big"
)

// Constants to match up protocol versions and messages
const (
	klay62 = 62
	klay63 = 63
	klay64 = 64
	klay65 = 65
)

// ProtocolName is the official short name of the protocol used during capability negotiation.
var ProtocolName = "klay"

// ProtocolVersions are the upported versions of the klay protocol (first is primary).
var ProtocolVersions = []uint{klay65, klay64, klay63, klay62}

// ProtocolLengths are the number of implemented message corresponding to different protocol versions.
var ProtocolLengths = []uint64{21, 19, 17, 8}

const ProtocolMaxMsgSize = 12 * 1024 * 1024 // Maximum cap on the size of a protocol message

const (
	StatusMsg = 0x00
)

type errCode int

const (
	ErrMsgTooLarge = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrProtocolVersionMismatch
	ErrNetworkIdMismatch
	ErrGenesisBlockMismatch
	ErrChainIDMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSuspendedPeer
	ErrUnexpectedTxType
	ErrFailedToGetStateDB
	ErrUnsupportedEnginePolicy
)

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

// statusData is the network packet for the status message.
type statusData struct {
	ProtocolVersion uint32
	NetworkId       uint64
	TD              *big.Int    // TODO-Guardian: TD is not used in Klaytn.
	CurrentBlock    common.Hash // TODO-Guardian: CurrentBlock is not used in Klaytn.
	GenesisBlock    common.Hash
	ChainID         *big.Int // ChainID to sign a transaction.
}
