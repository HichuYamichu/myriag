package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Handler handles all requests
type Handler struct {
	dockerServ *Service
}

// NewHandler creates new Handler
func NewHandler(dockerServ *Service) *Handler {
	return &Handler{dockerServ: dockerServ}
}

// Languages responds with a list of avalible languages
func (h *Handler) Languages(c echo.Context) error {
	langs, err := h.dockerServ.ListLanguages()
	if err != nil {
		echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, langs)
}

// Containers responds with a list created containers
func (h *Handler) Containers(c echo.Context) error {
	containers, err := h.dockerServ.ListContainers()
	if err != nil {
		echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, containers)
}

// CreateContainer handles creating new container
func (h *Handler) CreateContainer(c echo.Context) error {
	type createContainerPayload struct {
		Language string
	}

	p := &createContainerPayload{}
	if err := c.Bind(p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	err := h.dockerServ.CreateContainer(p.Language)
	if err != nil {
		echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusCreated, http.StatusText(http.StatusCreated))
}

// Eval handles code evaluation
func (h *Handler) Eval(c echo.Context) error {
	type evalPayload struct {
		Language string
		Code     string
	}

	p := &evalPayload{}
	if err := c.Bind(p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	res, err := h.dockerServ.Eval(p.Language, p.Code)
	if err != nil {
		echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, res)
}

// Cleanup handles cleaning up
func (h *Handler) Cleanup(c echo.Context) error {
	containers, err := h.dockerServ.Cleanup()
	if err != nil {
		echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, containers)
}
