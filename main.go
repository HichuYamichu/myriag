package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/docker/docker/client"
	"github.com/hichuyamichu/myriag/docker"
	"github.com/hichuyamichu/myriag/router"
	"github.com/labstack/echo/v4"
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
	app := bootstrap()

	go func() {
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-done
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		app.Shutdown(ctx)
	}()

	port := viper.GetString("port")
	host := viper.GetString("host")
	app.Logger.Fatal(app.Start(fmt.Sprintf("%s:%s", host, port)))
}

func bootstrap() *echo.Echo {
	dockerCLI, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("fatal error docker CLI: %s", err)
	}
	dockerCLI.NegotiateAPIVersion(context.Background())
	node, _ := snowflake.NewNode(0)

	dockerService := docker.NewService(dockerCLI, node)
	dockerHandler := docker.NewHandler(dockerService)

	r := router.New()

	api := r.Group("/api")
	api.GET("/languages", dockerHandler.Languages)
	api.GET("/containers", dockerHandler.Containers)
	api.POST("/create_container", dockerHandler.CreateContainer)
	api.POST("/eval", dockerHandler.Eval)
	api.POST("/cleanup", dockerHandler.Cleanup)

	return r
}
