package main

import (
	"net/http"

	"github.com/hichuyamichu/myriag/errors"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

func newServer() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = httpErrorHandler
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

func httpErrorHandler(err error, c echo.Context) {
	var code int
	var message string
	appErr, ok := err.(*errors.Error)
	if !ok {
		code = http.StatusInternalServerError
		message = http.StatusText(http.StatusInternalServerError)
	} else {
		code = appErr.Kind.HTTPStatus()
		message = appErr.Kind.String()
	}

	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, map[string]interface{}{"message": message})
		}
		if err != nil {
			c.Logger().Error(err.Error())
		}
	}
}
