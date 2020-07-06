package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/hichuyamichu/myriag/errors"
)

type SetupContainer struct {
	cli       *client.Client
	imageName string
	contName  string
}

func (sc *SetupContainer) Do() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	err := sc.startContainer(ctx)
	if err != nil {
		return err
	}

	err = sc.createEvalDir(ctx)
	if err != nil {
		return err
	}

	err = sc.chmodEvalDir(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (sc *SetupContainer) startContainer(ctx context.Context) error {
	cresp, err := sc.cli.ContainerCreate(ctx,
		&container.Config{
			Image:           sc.imageName,
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
		sc.contName,
	)
	if err != nil {
		return errors.E(err, errors.Internal)
	}

	err = sc.cli.ContainerStart(ctx, cresp.ID, types.ContainerStartOptions{})
	if err != nil {
		return errors.E(err, errors.Internal)
	}

	return nil
}

func (sc *SetupContainer) createEvalDir(ctx context.Context) error {
	iresp, err := sc.cli.ContainerExecCreate(
		ctx,
		sc.contName,
		types.ExecConfig{
			Cmd: []string{"mkdir", "eval"},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal)
	}

	if err := sc.cli.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal)
	}

	return nil
}

func (sc *SetupContainer) chmodEvalDir(ctx context.Context) error {
	iresp, err := sc.cli.ContainerExecCreate(
		ctx,
		sc.contName,
		types.ExecConfig{
			Cmd: []string{"chmod", "711", "eval"},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal)
	}

	if err := sc.cli.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal)
	}

	return nil
}
