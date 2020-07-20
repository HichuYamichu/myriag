package docker

import (
	"context"
	"time"

	"github.com/hichuyamichu/myriag/errors"
)

func (d *Docker) killContainer(contID string) error {
	const op errors.Op = "docker/Docker.killContainer"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	err := d.cli.ContainerKill(ctx, contID, "")
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}
	return nil
}
