package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleans up (kills) active docker containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cleaned, err := dockerHandler.Cleanup(cmd.Context())
		if err != nil {
			return err
		}
		logger.Info("cleaned", zap.Strings("cleaned", cleaned))
		return nil
	},
}
