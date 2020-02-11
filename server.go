package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

func newServer() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Logger.SetLevel(log.INFO)

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	dockerService := NewService()
	dockerHandler := NewHandler(dockerService)

	api := e.Group("/api")
	api.GET("/languages", dockerHandler.Languages)
	api.GET("/containers", dockerHandler.Containers)
	api.POST("/create_container", dockerHandler.CreateContainer)
	api.POST("/eval", dockerHandler.Eval)
	api.POST("/cleanup", dockerHandler.Cleanup)

	return e
}
