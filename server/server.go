package server

import (
	"context"
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/docker/docker/client"
	"github.com/hichuyamichu/myriag/docker"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

// Server application struct
type Server struct {
	router *echo.Echo

	dockerHandler *docker.Handler
}

// New creates new Server
func New(dockerClient *client.Client) *Server {
	node, _ := snowflake.NewNode(0)

	dockerService := docker.NewService(dockerClient, node)
	dockerHandler := docker.NewHandler(dockerService)

	server := &Server{
		router:        echo.New(),
		dockerHandler: dockerHandler,
	}

	server.configure()
	server.setRoutes()
	return server
}

func (s *Server) configure() {
	s.router.HideBanner = true
	s.router.HTTPErrorHandler = httpErrorHandler
	s.router.Validator = NewValidator()
	s.router.Logger.SetLevel(log.INFO)

	s.router.Use(middleware.Logger())
	s.router.Use(middleware.Recover())
}

func (s *Server) setRoutes() {
	api := s.router.Group("/api")
	api.GET("/languages", s.dockerHandler.Languages)
	api.GET("/containers", s.dockerHandler.Containers)
	api.POST("/create_container", s.dockerHandler.CreateContainer)
	api.POST("/eval", s.dockerHandler.Eval)
	api.POST("/cleanup", s.dockerHandler.Cleanup)
}

// Shutdown shuts down the server
func (s *Server) Shutdown(ctx context.Context) {
	s.router.Shutdown(ctx)
}

// Start starts the server
func (s *Server) Start(host string, port string) error {
	return s.router.Start(fmt.Sprintf("%s:%s", host, port))
}
