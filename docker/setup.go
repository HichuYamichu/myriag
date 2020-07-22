package docker

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/hichuyamichu/myriag/errors"
)

func (d *Docker) setupContainer(ctx context.Context, lang string) (string, error) {
	const op errors.Op = "docker/Docker.setupContainer"

	imageName := fmt.Sprintf("myriag_%s", lang)
	sf := snowflakes.Generate()
	contName := fmt.Sprintf("myriag_%s_%d", lang, sf)

	d.logger.Debug("starting container", zap.String("lang", lang), zap.String("container", contName))
	err := d.startContainer(ctx, imageName, contName, lang)
	if err != nil {
		return "", errors.E(err, op)
	}
	d.logger.Debug("started container", zap.String("lang", lang), zap.String("container", contName))

	d.logger.Debug("creating eval dir", zap.String("container", contName))
	err = d.createEvalDir(ctx, contName)
	if err != nil {
		return "", errors.E(err, op)
	}
	d.logger.Debug("created eval dir", zap.String("container", contName))

	d.logger.Debug("chmoding eval dir", zap.String("container", contName))
	err = d.chmodEvalDir(ctx, contName)
	if err != nil {
		return "", errors.E(err, op)
	}
	d.logger.Debug("chmoded eval dir", zap.String("container", contName))

	return contName, nil
}

func (d *Docker) startContainer(ctx context.Context, imageName, contName, lang string) error {
	const op errors.Op = "docker/Docker.startContainer"

	cresp, err := d.cli.ContainerCreate(ctx,
		&container.Config{
			Image:           imageName,
			User:            "1000:1000",
			WorkingDir:      "/tmp/",
			Tty:             true,
			NetworkDisabled: true,
			Entrypoint:      []string{"/bin/sh"},
		},
		&container.HostConfig{
			AutoRemove: true,
			Resources: container.Resources{
				NanoCPUs:   getNanoCPUFor(lang),
				Memory:     getMemoryFor(lang),
				MemorySwap: getMemoryFor(lang),
			},
		},
		nil,
		contName,
	)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	err = d.cli.ContainerStart(ctx, cresp.ID, types.ContainerStartOptions{})
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	return nil
}

func (d *Docker) createEvalDir(ctx context.Context, contName string) error {
	const op errors.Op = "docker/Docker.createEvalDir"

	iresp, err := d.cli.ContainerExecCreate(
		ctx,
		contName,
		types.ExecConfig{
			Cmd: []string{"mkdir", "eval"},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	if err := d.cli.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal, op)
	}

	return nil
}

func (d *Docker) chmodEvalDir(ctx context.Context, contName string) error {
	const op errors.Op = "docker/Docker.chmodEvalDir"

	iresp, err := d.cli.ContainerExecCreate(
		ctx,
		contName,
		types.ExecConfig{
			Cmd: []string{"chmod", "711", "eval"},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	if err := d.cli.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal, op)
	}

	return nil
}

func getMemoryFor(lang string) int64 {
	key := fmt.Sprintf("languages.%s.memory", lang)
	if viper.IsSet(key) {
		return int64(viper.GetSizeInBytes(key))
	} else {
		return int64(viper.GetSizeInBytes("defaultLanguage.memory"))
	}
}

func getNanoCPUFor(lang string) int64 {
	key := fmt.Sprintf("languages.%s.cpus", lang)
	if viper.IsSet(key) {
		return int64(10e9 * viper.GetFloat64(key))
	} else {
		return int64(10e9 * viper.GetFloat64("defaultLanguage.cpus"))
	}
}
