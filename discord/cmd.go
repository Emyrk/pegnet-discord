package discord

import "github.com/spf13/cobra"

func (a *PegnetDiscordBot) RootCmd() *cobra.Command {
	return &cobra.Command{
		Use: "!pegnet <command>",
	}
}
