package docker

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/docker/docker/client"
	"github.com/hichuyamichu/myriag/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var snowflakes, _ = snowflake.NewNode(1)

type Docker struct {
	cli    *client.Client
	logger *zap.Logger
}

func New(cli *client.Client, logger *zap.Logger) *Docker {
	return &Docker{cli: cli, logger: logger}
}

func (d *Docker) Build(ctx context.Context, langs []string) error {
	const op errors.Op = "docker/Docker.Build"

	d.logger.Info("building images", zap.Strings("languages", langs))
	for _, lang := range langs {
		err := d.build(ctx, lang)
		if err != nil {
			return errors.E(err, op)
		}
	}
	d.logger.Info("finished building images", zap.Strings("languages", langs))

	return nil
}

func (d *Docker) BuildConcurrently(ctx context.Context, langs []string) error {
	const op errors.Op = "docker/Docker.BuildConcurrently"
	d.logger.Info("building images concurrently", zap.Strings("languages", langs))

	done := make(chan error)
	wg := &sync.WaitGroup{}
	for _, lang := range langs {
		wg.Add(1)
		go func(lang string) {
			err := d.build(ctx, lang)
			if err != nil {
				done <- err
			}
			wg.Done()
		}(lang)
	}

	go func() {
		wg.Wait()
		done <- nil
	}()

	if err := <-done; err != nil {
		return errors.E(err, op)
	}

	d.logger.Info("finished building images")
	return nil
}

func (d *Docker) Eval(ctx context.Context, lang string, code string) (string, error) {
	const op errors.Op = "docker/Docker.Eval"
	d.logger.Info("starting eval", zap.String("language", lang), zap.String("code", code))

	err := isLangSupported(lang)
	if err != nil {
		return "", errors.E(err, op)
	}

	contName, err := d.fetchConntainerFor(ctx, lang)
	if err != nil {
		return "", errors.E(err, op)
	}

	res, err := d.eval(ctx, contName, code)
	if err != nil {
		return "", errors.E(err, op)
	}

	d.logger.Info("finished eval", zap.String("container", contName))
	return res, nil
}

func (d *Docker) SetupContainers(ctx context.Context, langs []string) error {
	const op errors.Op = "docker/Docker.SetupContainers"
	d.logger.Info("setting up containers")

	done := make(chan error)
	wg := &sync.WaitGroup{}
	for _, lang := range langs {
		wg.Add(1)
		go func(lang string) {
			_, err := d.SetupContainer(ctx, lang)
			if err != nil {
				done <- err
			}
			wg.Done()
		}(lang)
	}

	go func() {
		wg.Wait()
		done <- nil
	}()

	if err := <-done; err != nil {
		return errors.E(err, op)
	}

	d.logger.Info("finished setting up containers")
	return nil
}

func (d *Docker) SetupContainer(ctx context.Context, lang string) (string, error) {
	const op errors.Op = "docker/Docker.SetupContainer"
	d.logger.Info("setting up container", zap.String("lang", lang))

	err := isLangSupported(lang)
	if err != nil {
		return "", errors.E(err, op)
	}

	contName, err := d.setupContainer(ctx, lang)
	if err != nil {
		return "", errors.E(err, op)
	}

	d.logger.Info("finished setting up container", zap.String("lang", lang), zap.String("container", contName))
	return contName, nil
}

func (d *Docker) CleanupWithInterval(interval time.Duration, timeout time.Duration) {
	const _ errors.Op = "docker/Docker.CleanupWithInterval"
	d.logger.Info("periodic cleanup is set", zap.Duration("interval", interval))

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(interval)
	go func() {
		for {
			<-ticker.C
			cleaned, err := d.Cleanup(ctx)
			if err != nil {
				d.logger.Error("failed to cleanup containers", zap.Error(err))
			}
			d.logger.Info("finished cleaning up containers", zap.Strings("cleaned", cleaned))
		}
	}()
}

func (d *Docker) Cleanup(ctx context.Context) ([]string, error) {
	const op errors.Op = "docker/Docker.Cleanup"
	d.logger.Info("starting cleanup")

	containers, err := d.listContainers(ctx)
	if err != nil {
		return nil, errors.E(err, op)
	}

	res := make([]string, 0)
	cleaned := make(chan string)
	wg := &sync.WaitGroup{}
	for _, cont := range containers {
		contName := cont.Names[0][1:]
		wg.Add(1)
		go func(contID string) {
			err = d.killContainer(ctx, contID)
			if err != nil {
				d.logger.Error("failed to kill container", zap.String("container", contName))
			} else {
				cleaned <- contName
			}
			wg.Done()
		}(cont.ID)

		go func() {
			wg.Wait()
			close(cleaned)
		}()

		for cont := range cleaned {
			res = append(res, cont)
		}
	}

	return res, nil
}

func (d *Docker) ListContainers(ctx context.Context) ([]string, error) {
	const op errors.Op = "docker/Docker.ListContainers"

	containers, err := d.listContainers(ctx)
	if err != nil {
		return nil, errors.E(err, op)
	}

	res := make([]string, 0)
	for _, cont := range containers {
		contName := cont.Names[0][1:]
		res = append(res, contName)
	}

	return res, nil
}

func (d *Docker) fetchConntainerFor(ctx context.Context, lang string) (string, error) {
	const op errors.Op = "docker/Docker.fetchConntainerFor"

	containers, err := d.ListContainers(ctx)
	if err != nil {
		return "", errors.E(err, op)
	}

	if len(containers) == 0 {
		contName, err := d.SetupContainer(ctx, lang)
		if err != nil {
			return "", errors.E(err, op)
		}
		return contName, nil
	} else {
		return containers[rand.Intn(len(containers))], nil
	}
}

func isLangSupported(lang string) error {
	const op errors.Op = "docker/isLangSupported"

	languages := viper.GetStringMap("languages")
	exists := false
	for supportedLanguage := range languages {
		if supportedLanguage == lang {
			exists = true
			break
		}
	}
	if !exists {
		return errors.E(errors.Errorf("language `%s` not found", lang), errors.LanguageNotFound, op)
	}
	return nil
}
