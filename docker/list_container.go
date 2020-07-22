package docker

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/hichuyamichu/myriag/errors"
)

func (d *Docker) listContainers(ctx context.Context) ([]types.Container, error) {
	const op errors.Op = "docker/Docker.listContainers"

	containers, err := d.cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, errors.E(err, errors.Internal, op)
	}

	res := make([]types.Container, 0)
	for _, cont := range containers {
		contName := cont.Names[0][1:]
		if strings.HasPrefix(contName, "myriag_") {
			res = append(res, cont)
		}
	}
	return res, nil
}
