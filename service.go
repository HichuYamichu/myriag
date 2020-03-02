package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hichuyamichu/myriag/errors"
	"github.com/spf13/viper"
)

// Service performs operations on docker client
type Service struct {
	docker   *client.Client
	snowNode *snowflake.Node
}

// NewService creates new upload service
func NewService() *Service {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(ctx)

	node, _ := snowflake.NewNode(0)
	return &Service{docker: cli, snowNode: node}
}

// ListLanguages return a list of avalible languages
func (s *Service) ListLanguages() []string {
	langs := viper.GetStringSlice("languages")
	return langs
}

// ListContainers return a list of avalible containers
func (s *Service) ListContainers() ([]string, error) {
	const op errors.Op = "service.ListContainers"
	ctx := context.Background()
	containers, err := s.docker.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, errors.E(err, errors.Internal, op)
	}

	res := make([]string, 0)
	for _, cont := range containers {
		contName := cont.Names[0][1:]
		if strings.HasPrefix(contName, "myriag_") {
			res = append(res, contName)
		}
	}
	return res, nil
}

// CreateContainer creates a new container
func (s *Service) CreateContainer(lang string) error {
	const op errors.Op = "service.CreateContainer"
	ctx := context.Background()
	cresp, err := s.docker.ContainerCreate(ctx,
		&container.Config{
			Image:           fmt.Sprintf("myriag_%s:latest", lang),
			User:            "1000:1000",
			WorkingDir:      "/tmp/",
			Tty:             true,
			NetworkDisabled: true,
			Entrypoint:      []string{"/bin/sh"},
		},
		&container.HostConfig{
			AutoRemove: true,
			Resources: container.Resources{
				Memory:     1.28e+8,
				MemorySwap: 1.28e+8,
			},
		},
		nil,
		fmt.Sprintf("myriag_%s", lang),
	)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	err = s.docker.ContainerStart(ctx, cresp.ID, types.ContainerStartOptions{})
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	iresp, err := s.docker.ContainerExecCreate(
		ctx,
		fmt.Sprintf("myriag_%s", lang),
		types.ExecConfig{
			Cmd: []string{"mkdir", "eval"},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	if err := s.docker.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal, op)
	}

	iresp, err = s.docker.ContainerExecCreate(
		ctx,
		fmt.Sprintf("myriag_%s", lang),
		types.ExecConfig{
			Cmd: []string{"chmod", "711", "eval"},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	if err := s.docker.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal, op)
	}

	return nil
}

// Eval evaluates provided code
func (s *Service) Eval(lang string, code string) (string, error) {
	const op errors.Op = "service.Eval"
	languages := viper.GetStringSlice("languages")
	exists := false
	for _, supportedLanguage := range languages {
		if supportedLanguage == lang {
			exists = true
		}
	}
	if !exists {
		return "", errors.E(errors.Errorf("language not found"), errors.NotFound, op)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	sf := s.snowNode.Generate()
	evalDir := fmt.Sprintf("eval/%d", sf)

	iresp, err := s.docker.ContainerExecCreate(
		ctx,
		fmt.Sprintf("myriag_%s", lang),
		types.ExecConfig{
			Cmd: []string{"mkdir", evalDir},
		},
	)
	if err != nil {
		return "", errors.E(err, errors.Internal, op)
	}

	if err := s.docker.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return "", errors.E(err, errors.Internal, op)
	}

	iresp, err = s.docker.ContainerExecCreate(
		ctx,
		fmt.Sprintf("myriag_%s", lang),
		types.ExecConfig{
			Cmd: []string{"chmod", "777", evalDir},
		},
	)
	if err != nil {
		return "", errors.E(err, errors.Internal, op)
	}

	if err := s.docker.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return "", errors.E(err, errors.Internal, op)
	}

	iresp, err = s.docker.ContainerExecCreate(
		ctx,
		fmt.Sprintf("myriag_%s", lang),
		types.ExecConfig{
			User:         "1001:1001",
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   fmt.Sprintf("/tmp/%s", evalDir),
			Cmd:          []string{"/bin/sh", "/var/run/run.sh", code},
		},
	)
	if err != nil {
		return "", errors.E(err, errors.Internal, op)
	}

	aresp, err := s.docker.ContainerExecAttach(ctx, iresp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", errors.E(err, errors.Internal, op)
	}
	defer aresp.Close()

	var outBuf, errBuf bytes.Buffer
	outputDone := make(chan error)

	go func() {
		_, err = stdcopy.StdCopy(&outBuf, &errBuf, aresp.Reader)
		outputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			return "", errors.E(err, errors.Internal, op)
		}
		break

	case <-ctx.Done():
		return "", errors.E(errors.Errorf("evaluation timeout"), errors.Timeout, op)
	}

	_, err = s.docker.ContainerExecInspect(ctx, iresp.ID)
	if err != nil {
		return "", errors.E(err, errors.Internal, op)
	}

	var evaled string
	if errBuf.Len() != 0 {
		evaled = errBuf.String()
	} else {
		evaled = outBuf.String()
	}

	iresp, err = s.docker.ContainerExecCreate(
		ctx,
		fmt.Sprintf("myriag_%s", lang),
		types.ExecConfig{
			Cmd: []string{"rm", "-rf", evalDir},
		},
	)
	if err != nil {
		return "", errors.E(err, errors.Internal, op)
	}

	if err := s.docker.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return "", errors.E(err, errors.Internal, op)
	}

	return evaled, nil
}

// Cleanup cleans up containers
func (s *Service) Cleanup() ([]string, error) {
	const op errors.Op = "service.Cleanup"
	ctx := context.Background()
	containers, err := s.docker.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, errors.E(err, errors.Internal, op)
	}

	res := make([]string, 0)
	for _, cont := range containers {
		contName := cont.Names[0][1:]
		if strings.HasPrefix(contName, "myriag_") {
			err = s.docker.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{Force: true})
			if err != nil {
				continue
			}
			res = append(res, contName)
		}
	}
	return res, nil
}
