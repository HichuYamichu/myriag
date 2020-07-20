package docker

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hichuyamichu/myriag/errors"
)

func (d *Docker) eval(contName string, code string, timeout time.Duration) (string, error) {
	const op errors.Op = "docker/Docker.eval"

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sf := snowflakes.Generate()
	dir := fmt.Sprintf("eval/%d", sf)

	err := d.createUniqueEvalDir(ctx, contName, dir)
	if err != nil {
		return "", errors.E(err, op)
	}
	err = d.chmodUniqueEvalDir(ctx, contName, dir)
	if err != nil {
		return "", errors.E(err, op)
	}
	res, err := d.runExec(ctx, contName, dir, code)
	if err != nil {
		return "", errors.E(err, op)
	}
	_ = d.rmUniqueEvalDir(ctx, contName, dir)

	return res, nil
}

func (d *Docker) createUniqueEvalDir(ctx context.Context, contName, dir string) error {
	const op errors.Op = "docker/Docker.createUniqueEvalDir"

	iresp, err := d.cli.ContainerExecCreate(
		ctx,
		contName,
		types.ExecConfig{
			Cmd: []string{"mkdir", dir},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	if err := d.cli.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal, op)
	}

	return nil
}

func (d *Docker) chmodUniqueEvalDir(ctx context.Context, contName, dir string) error {
	const op errors.Op = "docker/Docker.chmodUniqueEvalDir"

	iresp, err := d.cli.ContainerExecCreate(
		ctx,
		contName,
		types.ExecConfig{
			Cmd: []string{"chmod", "777", dir},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	if err := d.cli.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal, op)
	}

	return nil
}

func (d *Docker) runExec(ctx context.Context, contName, dir, code string) (string, error) {
	const op errors.Op = "docker/Docker.runExec"

	iresp, err := d.cli.ContainerExecCreate(
		ctx,
		contName,
		types.ExecConfig{
			User:         "1001:1001",
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   fmt.Sprintf("/tmp/%s", dir),
			Cmd:          []string{"/bin/sh", "/var/run/run.sh", code},
		},
	)
	if err != nil {
		return "", errors.E(err, errors.Internal, op)
	}

	aresp, err := d.cli.ContainerExecAttach(ctx, iresp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", errors.E(err, errors.Internal, op)
	}
	defer aresp.Close()

	var outBuf, errBuf bytes.Buffer
	outputDone := make(chan error)

	go func() {
		_, err = stdcopy.StdCopy(&outBuf, &errBuf, aresp.Reader)
		outputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			return "", errors.E(err, errors.Internal, op)
		}
		break

	case <-ctx.Done():
		return "", errors.E(errors.Errorf("evaluation timeout"), errors.EvalTimeout, op)
	}

	_, err = d.cli.ContainerExecInspect(ctx, iresp.ID)
	if err != nil {
		return "", errors.E(err, errors.Internal, op)
	}

	if errBuf.Len() != 0 {
		return errBuf.String(), nil
	} else {
		return outBuf.String(), nil
	}
}

func (d *Docker) rmUniqueEvalDir(ctx context.Context, contName, dir string) error {
	const op errors.Op = "docker/Docker.rmUniqueEvalDir"

	iresp, err := d.cli.ContainerExecCreate(
		ctx,
		contName,
		types.ExecConfig{
			Cmd: []string{"rm", "-rf", dir},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal, op)
	}

	if err := d.cli.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal, op)
	}

	return nil
}
