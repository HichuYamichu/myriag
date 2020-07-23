package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hichuyamichu/myriag/docker"
	"github.com/hichuyamichu/myriag/errors"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Server struct {
	router *echo.Echo
	docker *docker.Docker
}

func New(docker *docker.Docker, logger *zap.Logger) *Server {
	r := echo.New()
	r.HideBanner = true
	r.HTTPErrorHandler = newErrorHandler(logger)
	r.Validator = newValidator()
	r.Use(middleware.Recover())

	s := &Server{
		router: r,
		docker: docker,
	}

	s.router.GET("/languages", s.languages)
	s.router.GET("/containers", s.containers)
	s.router.POST("/eval", s.eval)
	s.router.POST("/cleanup", s.cleanup)

	return s
}

func (s *Server) Shutdown(ctx context.Context) {
	_ = s.router.Shutdown(ctx)
}

func (s *Server) Start(host string, port string) error {
	return s.router.Start(fmt.Sprintf("%s:%s", host, port))
}

func (s *Server) languages(c echo.Context) error {
	langs := viper.GetStringSlice("languages")
	return c.JSON(http.StatusOK, langs)
}

func (s *Server) containers(c echo.Context) error {
	const op errors.Op = "server/Server.containers"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	containers, err := s.docker.ListContainers(ctx)
	if err != nil {
		return errors.E(err, op)
	}

	return c.JSON(http.StatusOK, containers)
}

func (s *Server) eval(c echo.Context) error {
	const op errors.Op = "server/Server.eval"

	type evalPayload struct {
		Language string `json:"language" validate:"required"`
		Code     string `json:"code" validate:"required"`
	}

	p := &evalPayload{}
	if err := c.Bind(p); err != nil {
		return errors.E(err, errors.Invalid, op)
	}

	if err := c.Validate(p); err != nil {
		return errors.E(err, op)
	}

	timeout := getTimeoutFor(p.Language)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	retry := 0
	maxRetry := getRetryCountFor(p.Language)
try:
	res, err := s.docker.Eval(ctx, p.Language, p.Code)
	if err != nil {
		if !errors.Is(err, errors.EvalTimeout) && retry <= maxRetry {
			retry++
			goto try
		}
		return errors.E(err, op)
	}

	type evalResponce struct {
		Result string `json:"result"`
	}

	maxOut := getMaxOutputFor(p.Language)
	// Go rune is 4 bytes wide
	if uint(len(res)*4) > maxOut {
		res = res[:maxOut/4]
	}

	return c.JSON(http.StatusOK, &evalResponce{Result: res})
}

func (s *Server) cleanup(c echo.Context) error {
	const op errors.Op = "server/Server.cleanup"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	containers, err := s.docker.Cleanup(ctx)
	if err != nil {
		return errors.E(err, op)
	}

	return c.JSON(http.StatusOK, containers)
}

func getRetryCountFor(lang string) int {
	key := fmt.Sprintf("languages.%s.retries", lang)
	if viper.IsSet(key) {
		return viper.GetInt(key)
	} else {
		return viper.GetInt("defaultLanguage.retries")
	}
}

func getMaxOutputFor(lang string) (size uint) {
	key := fmt.Sprintf("languages.%s.outputLimit", lang)
	if viper.IsSet(key) {
		size = viper.GetSizeInBytes(key)
	} else {
		size = viper.GetSizeInBytes("defaultLanguage.outputLimit")
	}
	return size
}

func getTimeoutFor(lang string) time.Duration {
	key := fmt.Sprintf("languages.%s.timeout", lang)
	if viper.IsSet(key) {
		return time.Second * viper.GetDuration(key)
	} else {
		return time.Second * viper.GetDuration("defaultLanguage.timeout")
	}
}
