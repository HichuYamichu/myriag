package docker

import (
	"context"
	"time"

	"github.com/docker/docker/client"
)

type kill struct {
	cli    *client.Client
	contID string
}

func (k *kill) Do() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	err := k.cli.ContainerKill(ctx, k.contID, "")
	if err != nil {
		return err
	}
	return nil
}
