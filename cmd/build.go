package cmd

import (
	"context"

	"github.com/hichuyamichu/myriag/config"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Builds required docker containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		if config.BuildConcurrently() {
			return dockerHandler.BuildConcurrently(context.Background(), config.Languages())
		} else {
			return dockerHandler.Build(context.Background(), config.Languages())
		}
	},
}
