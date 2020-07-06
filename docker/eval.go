package docker

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hichuyamichu/myriag/errors"
)

type Eval struct {
	cli      *client.Client
	contName string
	dir      string
	code     string
	result   string
}

func (e *Eval) Do() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	err := e.createUniqueEvalDir(ctx)
	if err != nil {
		return err
	}

	err = e.chmodUniqueEvalDir(ctx)
	if err != nil {
		return err
	}

	err = e.runExec(ctx)
	if err != nil {
		return err
	}

	err = e.rmUniqueEvalDir(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (e *Eval) createUniqueEvalDir(ctx context.Context) error {
	iresp, err := e.cli.ContainerExecCreate(
		ctx,
		e.contName,
		types.ExecConfig{
			Cmd: []string{"mkdir", e.dir},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal)
	}

	if err := e.cli.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal)
	}

	return nil
}

func (e *Eval) chmodUniqueEvalDir(ctx context.Context) error {
	iresp, err := e.cli.ContainerExecCreate(
		ctx,
		e.contName,
		types.ExecConfig{
			Cmd: []string{"chmod", "777", e.dir},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal)
	}

	if err := e.cli.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal)
	}

	return nil
}

func (e *Eval) runExec(ctx context.Context) error {
	iresp, err := e.cli.ContainerExecCreate(
		ctx,
		e.contName,
		types.ExecConfig{
			User:         "1001:1001",
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   fmt.Sprintf("/tmp/%s", e.dir),
			Cmd:          []string{"/bin/sh", "/var/run/run.sh", e.code},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal)
	}

	aresp, err := e.cli.ContainerExecAttach(ctx, iresp.ID, types.ExecStartCheck{})
	if err != nil {
		return errors.E(err, errors.Internal)
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
			return errors.E(err, errors.Internal)
		}
		break

	case <-ctx.Done():
		return errors.E(errors.Errorf("evaluation timeout"), errors.Timeout)
	}

	_, err = e.cli.ContainerExecInspect(ctx, iresp.ID)
	if err != nil {
		return errors.E(err, errors.Internal)
	}

	if errBuf.Len() != 0 {
		e.result = errBuf.String()
	} else {
		e.result = outBuf.String()
	}

	return nil
}

func (e *Eval) rmUniqueEvalDir(ctx context.Context) error {
	iresp, err := e.cli.ContainerExecCreate(
		ctx,
		e.contName,
		types.ExecConfig{
			Cmd: []string{"rm", "-rf", e.dir},
		},
	)
	if err != nil {
		return errors.E(err, errors.Internal)
	}

	if err := e.cli.ContainerExecStart(ctx, iresp.ID, types.ExecStartCheck{}); err != nil {
		return errors.E(err, errors.Internal)
	}

	return nil
}
