package cmd

import (
	"context"

	"github.com/hichuyamichu/myriag/config"
	"github.com/spf13/cobra"
)

var prepareCmd = &cobra.Command{
	Use:   "prepare",
	Short: "Prepares docker containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		return dockerHandler.SetupContainers(context.Background(), config.Languages())
	},
}
