package docker

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
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
	cli       *client.Client
	logger    *zap.Logger
	languages []string

	// evalQueue stores semaphores (buffered channels) used to limit concurrent evals
	evalQueue sync.Map
}

func New(cli *client.Client, logger *zap.Logger, langs []string) *Docker {
	return &Docker{cli: cli, logger: logger, languages: langs}
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

	d.logger.Info("finished building images")
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

	err := d.isLangSupported(lang)
	if err != nil {
		return "", errors.E(err, op)
	}

	contName, err := d.fetchConntainerFor(ctx, lang)
	if err != nil {
		return "", errors.E(err, op)
	}

	max := getMaxConcurrentEvlasFor(lang)
	entry, _ := d.evalQueue.LoadOrStore(contName, make(chan struct{}, max))
	sem := entry.(chan struct{})
	sem <- struct{}{}
	res, err := d.eval(ctx, contName, code)
	<-sem
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

	err := d.isLangSupported(lang)
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

func (d *Docker) CleanupWithInterval(interval time.Duration) {
	const _ errors.Op = "docker/Docker.CleanupWithInterval"
	d.logger.Info("periodic cleanup is set", zap.Duration("interval", interval))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
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

	desiredConts := make([]string, 0)
	for _, contName := range containers {
		if strings.HasPrefix(contName, fmt.Sprintf("myriag_%s", lang)) {
			desiredConts = append(desiredConts, contName)
		}
	}

	if len(desiredConts) == 0 {
		contName, err := d.SetupContainer(ctx, lang)
		if err != nil {
			return "", errors.E(err, op)
		}
		return contName, nil
	} else {
		return desiredConts[rand.Intn(len(desiredConts))], nil
	}
}

func (d *Docker) isLangSupported(lang string) error {
	const op errors.Op = "docker/isLangSupported"

	exists := false
	for _, supportedLanguage := range d.languages {
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

func getMaxConcurrentEvlasFor(lang string) int {
	key := fmt.Sprintf("languages.%s.concurrent", lang)
	if viper.IsSet(key) {
		return viper.GetInt(key)
	} else {
		return viper.GetInt("defaultLanguage.concurrent")
	}
}
