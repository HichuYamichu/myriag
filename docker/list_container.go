package docker

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/hichuyamichu/myriag/errors"
)

type listContainer struct {
	cli *client.Client
}

func (lc *listContainer) Do() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	containers, err := lc.cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, errors.E(err, errors.Internal)
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
