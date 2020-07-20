package docker

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/hichuyamichu/myriag/errors"
)

func (d *Docker) listContainers() ([]string, error) {
	const op errors.Op = "docker/Docker.listContainers"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	containers, err := d.cli.ContainerList(ctx, types.ContainerListOptions{})
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
