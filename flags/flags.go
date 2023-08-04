package flags

import (
	"github.com/klaytn/klaytn/cmd/utils"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

var (
	GuardianFlags = Merge(
		nodeFlags,
		p2pFlags,
		rpcFlags,
	)

	nodeFlags = []cli.Flag{
		utils.ConfFlag,
		altsrc.NewStringFlag(utils.SrvTypeFlag),
		altsrc.NewPathFlag(utils.DataDirFlag),
		altsrc.NewStringFlag(utils.GenKeyFlag),
		altsrc.NewBoolFlag(utils.WriteAddressFlag),
		altsrc.NewStringFlag(utils.BNAddrFlag),
		altsrc.NewStringFlag(utils.NATFlag),
		altsrc.NewStringFlag(utils.NetrestrictFlag),
		altsrc.NewBoolFlag(utils.MetricsEnabledFlag),
		altsrc.NewBoolFlag(utils.PrometheusExporterFlag),
		altsrc.NewIntFlag(utils.PrometheusExporterPortFlag),
		altsrc.NewStringFlag(utils.AuthorizedNodesFlag),
		altsrc.NewUint64Flag(utils.NetworkIdFlag),
	}

	p2pFlags = []cli.Flag{
		altsrc.NewIntFlag(utils.ListenPortFlag),
		altsrc.NewIntFlag(utils.SubListenPortFlag),
		altsrc.NewBoolFlag(utils.MultiChannelUseFlag),
		altsrc.NewIntFlag(utils.MaxConnectionsFlag),
		altsrc.NewIntFlag(utils.MaxRequestContentLengthFlag),
		altsrc.NewIntFlag(utils.MaxPendingPeersFlag),
		altsrc.NewUint64Flag(utils.TargetGasLimitFlag),
		altsrc.NewBoolFlag(utils.NoDiscoverFlag),
		altsrc.NewDurationFlag(utils.RWTimerWaitTimeFlag),
		altsrc.NewUint64Flag(utils.RWTimerIntervalFlag),
		altsrc.NewStringFlag(utils.NodeKeyFileFlag),
		altsrc.NewStringFlag(utils.NodeKeyHexFlag),
	}

	rpcFlags = []cli.Flag{
		altsrc.NewBoolFlag(utils.RPCEnabledFlag),
		altsrc.NewStringFlag(utils.RPCListenAddrFlag),
		altsrc.NewIntFlag(utils.RPCPortFlag),
		altsrc.NewStringFlag(utils.RPCApiFlag),
	}
)

// Merge merges the given flag slices.
func Merge(groups ...[]cli.Flag) []cli.Flag {
	var ret []cli.Flag
	for _, group := range groups {
		ret = append(ret, group...)
	}
	return ret
}
