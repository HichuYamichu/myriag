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
		return h.Error(c, err)
	}

	return c.JSON(http.StatusOK, langs)
}

// Containers responds with a list created containers
func (h *Handler) Containers(c echo.Context) error {
	containers, err := h.dockerServ.ListContainers()
	if err != nil {
		return h.Error(c, err)
	}

	return c.JSON(http.StatusOK, containers)
}

// CreateContainer handles creating new container
func (h *Handler) CreateContainer(c echo.Context) error {
	type createContainerPayload struct {
		Language string `json:"language"`
	}

	p := &createContainerPayload{}
	if err := c.Bind(p); err != nil {
		return h.Error(c, err)
	}

	err := h.dockerServ.CreateContainer(p.Language)
	if err != nil {
		return h.Error(c, err)
	}

	return c.NoContent(http.StatusCreated)
}

// Eval handles code evaluation
func (h *Handler) Eval(c echo.Context) error {
	type evalPayload struct {
		Language string `json:"language"`
		Code     string `json:"code"`
	}

	type evalResponce struct {
		Result string `json:"result"`
	}

	p := &evalPayload{}
	if err := c.Bind(p); err != nil {
		return h.Error(c, err)
	}

	res, err := h.dockerServ.Eval(p.Language, p.Code)
	if err != nil {
		return h.Error(c, err)
	}

	return c.JSON(http.StatusOK, &evalResponce{Result: res})
}

// Cleanup handles cleaning up
func (h *Handler) Cleanup(c echo.Context) error {
	containers, err := h.dockerServ.Cleanup()
	if err != nil {
		return h.Error(c, err)
	}

	return c.JSON(http.StatusOK, containers)
}

func (h *Handler) Error(c echo.Context, err error) error {
	switch err {
	case errTimeout:
		return echo.NewHTTPError(http.StatusGatewayTimeout)
	case errLanguageNotFound:
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
}
