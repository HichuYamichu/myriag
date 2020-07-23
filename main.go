package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"github.com/hichuyamichu/myriag/docker"
	"github.com/hichuyamichu/myriag/server"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var config_path = flag.String("config", "", "path to config file")
var languages_path = flag.String("languages", "", "path to languages directory")

func init() {
	rand.Seed(time.Now().Unix())

	flag.Parse()
	if *config_path == "" {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/myriag")
	} else {
		viper.SetConfigFile(*config_path)
	}
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("fatal error with config file: %s", err)
	}
	if *languages_path != "" {
		viper.Set("languages_path", *languages_path)
	}

	viper.SetDefault("buildConcurrently", false)
	viper.SetDefault("prepareContainers", false)
	viper.SetDefault("cleanupInterval", 30)
	viper.SetDefault("defaultLanguage.memory", "256mb")
	viper.SetDefault("defaultLanguage.cpus", 0.25)
	viper.SetDefault("defaultLanguage.timeout", 20)
	viper.SetDefault("defaultLanguage.concurrent", 5)
	viper.SetDefault("defaultLanguage.retries", 10)
	viper.SetDefault("defaultLanguage.outputLimit", "4kb")
}

func main() {
	logger, _ := zap.Config{
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

	langs := languages()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logger.Error("docker connection failed", zap.Error(err))
		os.Exit(1)
	}
	cli.NegotiateAPIVersion(context.Background())
	docker := docker.New(cli, logger, langs)

	if viper.GetBool("buildConcurrently") {
		err = docker.BuildConcurrently(context.Background(), langs)
	} else {
		err = docker.Build(context.Background(), langs)
	}
	if err != nil {
		logger.Error("container build failed", zap.Error(err))
		os.Exit(1)
	}

	if viper.GetBool("prepareContainers") {
		err = docker.SetupContainers(context.Background(), langs)
		if err != nil {
			logger.Error("container setup failed", zap.Error(err))
			os.Exit(1)
		}
	}

	duration := time.Duration(viper.GetInt("cleanupInterval"))
	docker.CleanupWithInterval(time.Minute * duration)

	srv := server.New(docker, logger)

	go func() {
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-done
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	port := viper.GetString("port")
	host := viper.GetString("host")
	fmt.Println(srv.Start(host, port))
}

func languages() []string {
	res := make([]string, 0)
	languages := viper.GetStringMap("languages")
	for language := range languages {
		res = append(res, language)
	}

	return res
}
