package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"github.com/hichuyamichu/myriag/docker"
	"github.com/hichuyamichu/myriag/server"
	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/myriag")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("fatal error config file: %s", err)
	}
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("fatal error docker CLI: %s", err)
	}
	cli.NegotiateAPIVersion(context.Background())
	docker := docker.New(cli)
	// docker.BuildImage("go")
	// docker.CreateContainer("go")
	// err = docker.KillContainer("0094efb64852")
	// _, err = docker.Cleanup()
	if err != nil {
		panic(err)
	}

	srv := server.New(docker)

	go func() {
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-done
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	// port := viper.GetString("port")
	// host := viper.GetString("host")
	// srv.Start(host, port)
}
