package docker

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/hichuyamichu/myriag/errors"
)

type Cleanup struct {
	cli *client.Client
}

func (c *Cleanup) Do() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	containers, err := c.cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, errors.E(err, errors.Internal)
	}

	res := make([]string, 0)
	for _, cont := range containers {
		contName := cont.Names[0][1:]
		if strings.HasPrefix(contName, "myriag_") {
			err = c.cli.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{Force: true})
			if err != nil {
				continue
			}
			res = append(res, contName)
		}
	}
	return res, nil
}
