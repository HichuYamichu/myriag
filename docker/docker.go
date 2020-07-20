package docker

import (
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
	cli    *client.Client
	logger *zap.Logger
}

func New(cli *client.Client, logger *zap.Logger) *Docker {
	return &Docker{cli: cli, logger: logger}
}

func (d *Docker) Build(langs []string) error {
	const op errors.Op = "docker/Docker.Build"

	for _, lang := range langs {
		err := d.build(lang)
		if err != nil {
			return errors.E(err, op)
		}
	}

	return nil
}

func (d *Docker) BuildConcurrently(langs []string) error {
	const op errors.Op = "docker/Docker.BuildConcurrently"

	done := make(chan error)

	wg := &sync.WaitGroup{}
	for _, lang := range langs {
		wg.Add(1)
		go func(lang string) {
			err := d.build(lang)
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

	err := <-done

	return errors.E(err, op)
}

func (d *Docker) Eval(lang string, code string) (string, error) {
	const op errors.Op = "docker/Docker.Eval"

	err := isLangSupported(lang)
	if err != nil {
		return "", errors.E(err, op)
	}

	contName, err := d.fetchConntainerFor(lang)
	if err != nil {
		return "", errors.E(err, op)
	}

	timeout := getTimeoutFor(lang)

	res, err := d.eval(contName, code, timeout)
	if err != nil {
		return "", errors.E(err, op)
	}

	return res, nil
}

func (d *Docker) SetupContainers(langs []string) ([]string, error) {
	const op errors.Op = "docker/Docker.SetupContainers"

	created := make([]string, len(langs))
	for _, lang := range langs {
		contName, err := d.SetupContainer(lang)
		if err != nil {
			return nil, errors.E(err, op)
		}
		created = append(created, contName)
	}
	return created, nil
}

func (d *Docker) SetupContainer(lang string) (string, error) {
	const op errors.Op = "docker/Docker.SetupContainer"

	err := isLangSupported(lang)
	if err != nil {
		return "", errors.E(err, op)
	}

	contName, err := d.setupContainer(lang)
	if err != nil {
		return "", errors.E(err, op)
	}
	return contName, nil
}

func (d *Docker) CleanupWithInterval(interval time.Duration) {
	const _ errors.Op = "docker/Docker.CleanupWithInterval"

	ticker := time.NewTicker(interval)

	go func() {
		for {
			<-ticker.C
			_, _ = d.Cleanup()
		}
	}()
}

func (d *Docker) Cleanup() ([]string, error) {
	const op errors.Op = "docker/Docker.Cleanup"

	cleaned, err := d.cleanup()
	if err != nil {
		return nil, errors.E(err, op)
	}

	return cleaned, nil
}

func (d *Docker) KillContainers(contIDs []string) error {
	const _ errors.Op = "docker/Docker.KillContainers"

	wg := &sync.WaitGroup{}
	for _, contID := range contIDs {
		wg.Add(1)
		go func(contID string) {
			_ = d.KillContainer(contID)
			wg.Done()
		}(contID)
	}
	wg.Wait()

	return nil
}

func (d *Docker) KillContainer(contID string) error {
	const op errors.Op = "docker/Docker.KillContainer"

	err := d.killContainer(contID)
	if err != nil {
		return errors.E(err, op)
	}
	return nil
}

func (d *Docker) ListContainers() ([]string, error) {
	const op errors.Op = "docker/Docker.ListContainers"

	containers, err := d.listContainers()
	if err != nil {
		return nil, errors.E(err, op)
	}
	return containers, nil
}

func (d *Docker) fetchConntainerFor(lang string) (string, error) {
	const op errors.Op = "docker/Docker.fetchConntainerFor"

	containers, err := d.ListContainers()
	if err != nil {
		return "", errors.E(err, op)
	}

	validContainers := make([]string, 0)
	for _, contName := range containers {
		if strings.HasPrefix(contName, fmt.Sprintf("myriag_%s", lang)) {
			validContainers = append(validContainers, contName)
		}
	}

	if len(validContainers) == 0 {
		contName, err := d.SetupContainer(lang)
		if err != nil {
			return "", errors.E(err, op)
		}
		return contName, nil
	} else {
		return validContainers[rand.Intn(len(validContainers))], nil
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

func getTimeoutFor(lang string) time.Duration {
	key := fmt.Sprintf("languages.%s.timeout", lang)
	if viper.IsSet(key) {
		return time.Second * viper.GetDuration(key)
	} else {
		return time.Second * viper.GetDuration("defaultLanguage.timeout")
	}
}
