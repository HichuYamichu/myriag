package docker

import (
	"net/http"

	"github.com/hichuyamichu/myriag/errors"
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
	langs := h.dockerServ.ListLanguages()
	return c.JSON(http.StatusOK, langs)
}

// Containers responds with a list created containers
func (h *Handler) Containers(c echo.Context) error {
	const op errors.Op = "docker/handler.Containers"

	containers, err := h.dockerServ.ListContainers()
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	return c.JSON(http.StatusOK, containers)
}

// CreateContainer handles creating new container
func (h *Handler) CreateContainer(c echo.Context) error {
	const op errors.Op = "docker/handler.CreateContainer"

	type createContainerPayload struct {
		Language string `json:"language" validate:"required"`
	}

	p := &createContainerPayload{}
	if err := c.Bind(p); err != nil {
		return errors.E(err, errors.Invalid, op)
	}

	if err := c.Validate(p); err != nil {
		return errors.E(err, errors.Invalid, op)
	}

	err := h.dockerServ.CreateContainer(p.Language)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	return c.NoContent(http.StatusCreated)
}

// Eval handles code evaluation
func (h *Handler) Eval(c echo.Context) error {
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

	res, err := h.dockerServ.Eval(p.Language, p.Code)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	type evalResponce struct {
		Result string `json:"result"`
	}
	return c.JSON(http.StatusOK, &evalResponce{Result: res})
}

// Cleanup handles cleaning up
func (h *Handler) Cleanup(c echo.Context) error {
	const op errors.Op = "docker/handler.Cleanup"

	containers, err := h.dockerServ.Cleanup()
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	return c.JSON(http.StatusOK, containers)
}
