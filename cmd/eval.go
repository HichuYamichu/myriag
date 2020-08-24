package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluates provided code",
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := dockerHandler.Eval(cmd.Context(), args[0], args[1])
		if err != nil {
			return err
		}
		logger.Info("eval complete", zap.String("result", res))
		return nil
	},
}
