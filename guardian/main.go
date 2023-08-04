// Modifications Copyright 2018 The klaytn Authors
// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.
//
// This file is derived from cmd/bootnode/main.go (2018/06/04).
// Modified and improved for the klaytn development.

package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/klaytn/guardian/flags"
	"github.com/klaytn/guardian/node"
	"github.com/klaytn/klaytn/api/debug"
	"github.com/klaytn/klaytn/cmd/utils"
	"github.com/klaytn/klaytn/cmd/utils/nodecmd"
	"github.com/klaytn/klaytn/log"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

// GUARDIAN
var logger = log.NewModuleLogger(log.CMDKCN)

func guardian(ctx *cli.Context) error {
	cfg := node.NewGuardianConfig(ctx)
	if err := nodecmd.CheckCommands(ctx); err != nil {
		return err
	}

	node.SetIPC(ctx, cfg)
	node.SetAuthorizedNodes(ctx, cfg)
	node.SetP2PConfig(ctx, cfg)

	// Check exit condition
	switch cfg.CheckCMDState() {
	case node.GenerateNodeKeySpecified:
		cfg.GenerateNodeKey()
	case node.NoPrivateKeyPathSpecified:
		return errors.New("Use --nodekey or --nodekeyhex to specify a private key")
	case node.NodeKeyDuplicated:
		return errors.New("Options --nodekey and --nodekeyhex are mutually exclusive")
	case node.WriteOutAddress:
		cfg.DoWriteOutAddress()
	default:
		err := cfg.ReadNodeKey()
		if err != nil {
			return err
		}
	}

	err := cfg.ValidateNetworkParameter()
	if err != nil {
		return err
	}

	node, err := node.New(cfg)
	if err != nil {
		return err
	}

	if err := startNode(node); err != nil {
		return err
	}
	node.Wait()
	return nil
}

func startNode(node *node.Node) error {
	if err := node.Start(); err != nil {
		return err
	}
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		logger.Info("Got interrupt, shutting down...")
		go node.Stop()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				logger.Info("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}
	}()
	return nil
}

func before(ctx *cli.Context) error {
	if err := altsrc.InitInputSourceWithContext(
		flags.GuardianFlags,
		altsrc.NewYamlSourceFromFlagFunc("conf"),
	)(ctx); err != nil {
		return err
	}
	return nil
}

func main() {
	// TODO-Klaytn: remove `help` command
	app := utils.NewApp("", "the Klaytn's guardian command line interface")
	app.Name = "guardian"
	app.Copyright = "Copyright 2023 The klaytn Authors"
	app.UsageText = app.Name + " [global options] [commands]"
	app.Flags = flags.GuardianFlags
	app.Commands = []*cli.Command{
		nodecmd.VersionCommand,
		nodecmd.AttachCommand,
	}

	app.Action = guardian

	app.CommandNotFound = nodecmd.CommandNotExist
	app.OnUsageError = nodecmd.OnUsageError

	app.Before = before

	app.After = func(c *cli.Context) error {
		debug.Exit()
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
