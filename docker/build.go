package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/hichuyamichu/myriag/errors"
)

func (d *Docker) build(lang string) error {
	const op errors.Op = "docker/Docker.build"

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	sf := snowflakes.Generate()
	_ = fmt.Sprintf("myriag_%s_%d", lang, sf)
	// buildContext := strings.NewReader("/home/hy/source/go/myriag/languages/bash/Dockerfile")
	// ibres, err := b.cli.ImageBuild(ctx, buildContext, types.ImageBuildOptions{Dockerfile: "Dockerfile"})
	ibres, err := d.cli.ImageBuild(ctx, nil, types.ImageBuildOptions{Dockerfile: "./dockerfile"})
	if err != nil {
		return errors.E(err, op)
	}
	defer ibres.Body.Close()
	fmt.Println(ibres)
	return nil
}
