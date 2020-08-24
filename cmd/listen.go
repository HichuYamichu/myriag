package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hichuyamichu/myriag/config"
	"github.com/hichuyamichu/myriag/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Starts http server for remote eval requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if config.BuildConcurrently() {
			err = dockerHandler.BuildConcurrently(context.Background(), config.Languages())
		} else {
			err = dockerHandler.Build(context.Background(), config.Languages())
		}
		if err != nil {
			return err
		}

		if config.PrepareContainers() {
			err = dockerHandler.SetupContainers(context.Background(), config.Languages())
			if err != nil {
				return err
			}
		}

		dockerHandler.CleanupWithInterval(config.CleanupInterval())
		srv := server.New(dockerHandler, logger)

		go func() {
			done := make(chan os.Signal, 1)
			signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			<-done
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			srv.Shutdown(ctx)
		}()

		addr := fmt.Sprintf("%s:%s", config.Host(), config.Port())
		logger.Info("starting server", zap.String("addr", addr))
		return srv.Start(addr)
	},
}
