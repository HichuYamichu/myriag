package docker

import (
	"context"

	"github.com/hichuyamichu/myriag/errors"
	"go.uber.org/zap"
)

func (d *Docker) killContainer(ctx context.Context, contID string) error {
	const op errors.Op = "docker/Docker.killContainer"
	d.logger.Debug("starting container kill", zap.String("id", contID))

	err := d.cli.ContainerKill(ctx, contID, "")
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	d.logger.Debug("container killed", zap.String("id", contID))
	return nil
}
