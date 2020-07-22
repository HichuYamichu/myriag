package docker

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/docker/docker/api/types"
	"github.com/hichuyamichu/myriag/errors"
)

func (d *Docker) build(ctx context.Context, lang string) error {
	const op errors.Op = "docker/Docker.build"

	sf := snowflakes.Generate()
	imageName := fmt.Sprintf("myriag_%s_%d", lang, sf)
	d.logger.Debug("building image", zap.String("image", imageName))
	// buildContext := strings.NewReader("/home/hy/source/go/myriag/languages/bash/Dockerfile")
	// ibres, err := b.cli.ImageBuild(ctx, buildContext, types.ImageBuildOptions{Dockerfile: "Dockerfile"})
	ibres, err := d.cli.ImageBuild(ctx, nil, types.ImageBuildOptions{Dockerfile: "./dockerfile"})
	if err != nil {
		return errors.E(err, op)
	}
	defer ibres.Body.Close()
	fmt.Println(ibres)

	d.logger.Debug("build complete", zap.String("image", imageName))
	return nil
}
