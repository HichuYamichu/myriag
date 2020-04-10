package server

import (
	"net/http"

	"github.com/hichuyamichu/myriag/errors"
	"github.com/labstack/echo/v4"
)

func httpErrorHandler(err error, c echo.Context) {
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
