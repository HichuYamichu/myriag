package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hichuyamichu/myriag/docker"
	"github.com/hichuyamichu/myriag/errors"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/spf13/viper"
)

type Server struct {
	router *echo.Echo
	docker *docker.Docker
}

func New(docker *docker.Docker) *Server {
	server := &Server{
		router: echo.New(),
		docker: docker,
	}

	server.configure()
	server.setRoutes()
	return server
}

func (s *Server) Shutdown(ctx context.Context) {
	_ = s.router.Shutdown(ctx)
}

func (s *Server) Start(host string, port string) error {
	return s.router.Start(fmt.Sprintf("%s:%s", host, port))
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
	api.GET("/languages", s.languages)
	api.GET("/containers", s.containers)
	api.POST("/eval", s.eval)
	api.POST("/cleanup", s.cleanup)
}

func (s *Server) languages(c echo.Context) error {
	langs := viper.GetStringSlice("languages")
	return c.JSON(http.StatusOK, langs)
}

func (s *Server) containers(c echo.Context) error {
	containers, err := s.docker.ListContainers()
	if err != nil {
		return errors.E(err, errors.Internal)
	}

	return c.JSON(http.StatusOK, containers)
}

func (s *Server) eval(c echo.Context) error {
	const op errors.Op = "docker/handler.Eval"

	type evalPayload struct {
		Language string `json:"language" validate:"required"`
		Code     string `json:"code" validate:"required"`
	}

	p := &evalPayload{}
	if err := c.Bind(p); err != nil {
		return errors.E(err, errors.Invalid, op)
	}

	if err := c.Validate(p); err != nil {
		return errors.E(err, errors.Invalid, op)
	}

	res, err := s.docker.Eval(p.Language, p.Code)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	type evalResponce struct {
		Result string `json:"result"`
	}
	return c.JSON(http.StatusOK, &evalResponce{Result: res})
}

func (s *Server) cleanup(c echo.Context) error {
	containers, err := s.docker.Cleanup()
	if err != nil {
		return errors.E(err, errors.Internal)
	}

	return c.JSON(http.StatusOK, containers)
}
