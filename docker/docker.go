package docker

import (
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/docker/docker/client"
	"github.com/hichuyamichu/myriag/errors"
	"github.com/spf13/viper"
)

var snowflakes, _ = snowflake.NewNode(1)

type Docker struct {
	cli      *client.Client
	contPool containerPool
}

func New(cli *client.Client) *Docker {
	return &Docker{cli: cli}
}

func (d *Docker) Eval(lang string, code string) (string, error) {
	err := isLangSupported(lang)
	if err != nil {
		return "", err
	}

	d.contPool.RLock()
	contName, ok := d.contPool.containers[lang]
	d.contPool.RUnlock()
	// TODO: ensure only one container gets created
	if !ok {
		err = d.SetupContainer(lang)
		if err != nil {
			return "", err
		}
	}

	e := &Eval{
		cli:      d.cli,
		contName: contName,
		dir:      nameEvalDir(lang),
		code:     code,
	}
	err = e.Do()
	if err != nil {
		return "", err
	}

	return e.result, nil
}

func (d *Docker) SetupContainers(langs []string) error {
	for _, lang := range langs {
		err := d.SetupContainer(lang)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Docker) SetupContainer(lang string) error {
	err := isLangSupported(lang)
	if err != nil {
		return err
	}

	contName := nameContainer(lang)
	sc := &SetupContainer{
		cli:       d.cli,
		contName:  contName,
		imageName: nameImage(lang),
	}
	err = sc.Do()
	if err != nil {
		return err
	}

	d.contPool.Lock()
	d.contPool.containers[lang] = contName
	d.contPool.Unlock()
	return nil
}

func (d *Docker) CleanupWithInterval(interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for {
			<-ticker.C
			d.Cleanup()
		}
	}()
}

func (d *Docker) Cleanup() ([]string, error) {
	c := &Cleanup{cli: d.cli}
	cleaned, err := c.Do()
	if err != nil {
		return nil, err
	}

	return cleaned, err
}

func (d *Docker) KillContainers(contIDs []string) error {
	for _, contID := range contIDs {
		err := d.KillContainer(contID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Docker) ListContainers() ([]string, error) {
	lc := listContainer{}
	containers, err := lc.Do()
	if err != nil {
		return nil, err
	}
	return containers, nil
}

func (d *Docker) KillContainer(contID string) error {
	k := &kill{cli: d.cli, contID: contID}
	err := k.Do()
	if err != nil {
		return err
	}
	return nil
}

type containerPool struct {
	sync.RWMutex
	containers map[string]string
}

func isLangSupported(lang string) error {
	languages := viper.GetStringSlice("languages")
	exists := false
	for _, supportedLanguage := range languages {
		if supportedLanguage == lang {
			exists = true
			break
		}
	}
	if !exists {
		return errors.E(errors.Errorf("language not found"), errors.NotFound)
	}
	return nil
}

func nameContainer(lang string) string {
	sf := snowflakes.Generate()
	return fmt.Sprintf("myriag_%s_%d", lang, sf)
}

func nameEvalDir(lang string) string {
	sf := snowflakes.Generate()
	return fmt.Sprintf("eval/%d", sf)
}

func nameImage(lang string) string {
	return fmt.Sprintf("myriag_%s", lang)
}
