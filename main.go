package main

import (
	"context"

	"github.com/Emyrk/pegnet-discord/discord"
	"github.com/pegnet/pegnet-node/node"
	"github.com/pegnet/pegnet/balances"
	pcmd "github.com/pegnet/pegnet/cmd"
	"github.com/pegnet/pegnet/common"
	"github.com/spf13/cobra"
)

// This is a bit of a hack atm.
// Because the pegnet node uses sqlite, it adds a dependency for cgo.
// To isolate the dependency to just users that need to use the `node`,
// the `pegnet node` command was brought into this package. Because of the
// 'persistent' use of flags, there are many flags that do not affect the node that will
// be listed. I think for now, this is the simplest change to remove the dependency.

func init() {
	// Reset commands, so `pegnetnode` is the only h
	pcmd.RootCmd.ResetCommands()
	pcmd.RootCmd.Flags().Bool("mock", false, "Do not actually connect to discord")
	pcmd.RootCmd.Run = func(cmd *cobra.Command, args []string) {
		discordNode.Run(cmd, args)
	}
}

// main
func main() {
	pcmd.Execute()
}

var discordNode = &cobra.Command{
	Use:   "node",
	Short: "Runs a pegnet node",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		common.GlobalExitHandler.AddCancel(cancel)
		pcmd.ValidateConfig(pcmd.Config) // Will fatal log if it fails
		b := balances.NewBalanceTracker()

		// Services
		monitor := pcmd.LaunchFactomMonitor(pcmd.Config)
		grader := pcmd.LaunchGrader(pcmd.Config, monitor, b, ctx, false)

		pegnetnode, err := node.NewPegnetNode(pcmd.Config, monitor, grader)
		if err != nil {
			pcmd.CmdError(cmd, err)
		}
		common.GlobalExitHandler.AddExit(pegnetnode.Close)

		go pegnetnode.Run(ctx)

		var _ = cancel
		apiserver := pcmd.LaunchAPI(pcmd.Config, nil, grader, b, false)
		apiserver.Mux.Handle("/node/v1", pegnetnode.APIMux())
		// Let's add the pegnet node timeseries to the handle
		apiport, err := pcmd.Config.Int(common.ConfigAPIPort)
		if err != nil {
			pcmd.CmdError(cmd, err)
		}
		var _ = apiport
		//go apiserver.Listen(apiport)

		mock, _ := cmd.Flags().GetBool("mock")
		var bot *discord.PegnetDiscordBot
		// Launch the discord bot
		if mock {
			bot, err = discord.NewMockPegnetDiscordBot(pcmd.Config)
		} else {
			bot, err = discord.NewPegnetDiscordBot(args[0], pcmd.Config)
		}
		if err != nil {
			pcmd.CmdError(cmd, err)
		}
		bot.Node = pegnetnode
		bot.API = apiserver

		common.GlobalExitHandler.AddCancel(cancel)

		bot.Run(ctx)

	},
}
