package router

import (
	"net/http"

	"github.com/hichuyamichu/myriag/errors"
	"github.com/labstack/echo/v4"
)

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
