package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/docker/docker/api/types"
	"github.com/hichuyamichu/myriag/config"
	"github.com/hichuyamichu/myriag/errors"
)

func (d *Docker) build(ctx context.Context, lang string) error {
	const op errors.Op = "docker/Docker.build"

	imageName := fmt.Sprintf("myriag_%s", lang)
	d.logger.Debug("building image", zap.String("image", imageName))

	langDir := config.PathToLanguages()
	source := fmt.Sprintf("%s/%s", langDir, lang)

	buffer := new(bytes.Buffer)
	tarfileWriter := tar.NewWriter(buffer)
	defer tarfileWriter.Close()

	err := filepath.Walk(source, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}

		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		if err := tarfileWriter.WriteHeader(header); err != nil {
			return err
		}

		data, err := os.Open(file)
		if err != nil {
			return err
		}

		if _, err = io.Copy(tarfileWriter, data); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return errors.E(err, op)
	}

	iresp, err := d.cli.ImageBuild(ctx, buffer, types.ImageBuildOptions{
		Tags:       []string{imageName},
		Remove:     true,
		PullParent: true,
	})
	if err != nil {
		return errors.E(err, op)
	}

	// ImageBuild returns immediately so we block until the stream is over
	b := make([]byte, 4)
	for {
		_, err = iresp.Body.Read(b)
		if err == io.EOF {
			break
		}
	}

	err = iresp.Body.Close()
	if err != nil {
		return errors.E(err, op)
	}

	_, _, err = d.cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return errors.E(err, op)
	}

	d.logger.Debug("build complete", zap.String("image", imageName))
	return nil
}
