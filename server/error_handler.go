package server

import (
	"net/http"

	"github.com/hichuyamichu/myriag/errors"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func newErrorHandler(logger *zap.Logger) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {
		logger.Error(err.Error())

		var code int
		var message interface{}
		switch e := err.(type) {
		case *errors.Error:
			code = e.Kind.HTTPStatus()
			message = e.Kind.String()
		case *echo.HTTPError:
			code = e.Code
			message = e.Message
		default:
			code = http.StatusInternalServerError
			message = http.StatusText(http.StatusInternalServerError)
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
}
