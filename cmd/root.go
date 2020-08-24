package cmd

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/hichuyamichu/myriag/config"
	"github.com/hichuyamichu/myriag/docker"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"math/rand"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile        string
	dockerfilesDir string

	logger        *zap.Logger
	dockerHandler *docker.Docker

	rootCmd = &cobra.Command{
		Use:          "myriag",
		Short:        "Arbitrary code execution server using Docker",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logger, _ = zap.Config{
				Encoding:          "json",
				Level:             zap.NewAtomicLevelAt(zapcore.DebugLevel),
				OutputPaths:       []string{"stdout"},
				DisableCaller:     true,
				DisableStacktrace: true,
				EncoderConfig: zapcore.EncoderConfig{
					MessageKey:     "message",
					LevelKey:       "level",
					EncodeLevel:    zapcore.CapitalLevelEncoder,
					EncodeDuration: zapcore.StringDurationEncoder,
				},
			}.Build()
			defer logger.Sync()

			initConfig()

			cli, err := client.NewClientWithOpts(client.FromEnv)
			if err != nil {
				return err
			}
			cli.NegotiateAPIVersion(context.Background())
			dockerHandler = docker.New(cli, logger)
			return nil
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rand.Seed(time.Now().Unix())
	config.SetDefaults()
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringVarP(&dockerfilesDir, "languages", "l", "", "docker files dir path")
	rootCmd.AddCommand(listenCmd)
	rootCmd.AddCommand(evalCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(prepareCmd)
}

func initConfig() {
	if cfgFile != "" {
		config.UseConfigFile(cfgFile)
	} else {
		config.TryFindConfig()
	}

	if err := config.ReadInConfig(); err == nil {
		logger.Info("config file found", zap.String("path", config.ConfigFileUsed()))
	} else {
		logger.Error("error reading config file", zap.Error(err))
	}

	if dockerfilesDir != "" {
		viper.Set("languages_path", dockerfilesDir)
	}
}
