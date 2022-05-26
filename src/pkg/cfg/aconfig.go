package cfg

import (
	"context"
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/linyuanbin/viper-reload/src/pkg/cfg/remote"
)

type Type int64

const (
	FILE   Type = 0
	REMOTE Type = 1
)

type AConfig struct {
	remoteChan     remote.Config `yaml:"-"`
	ConfigFileName string
	viper          *viper.Viper `yaml:"-"`
	Type           Type

	Speed int
	Time  int64

	//
	EtcdUrl    string `yaml:"-"`
	RemoteType string `yaml:"-"`
}

func (c *AConfig) Parse() error {
	c.viper.SetConfigType("yaml")
	if err := c.addProvider(); err != nil {
		return err
	}
	err := c.load()
	if err != nil {
		return err
	}
	return c.viper.Unmarshal(c)
}

func (c *AConfig) load() error {
	if c.Type == FILE {
		return c.viper.ReadInConfig()
	}
	return c.viper.ReadRemoteConfig()
}

func (c *AConfig) WatchConfig(ctx context.Context) error {
	if c.Type == REMOTE {
		return c.watchRemoteConfigOnChannel(ctx)
	} else {
		c.viper.WatchConfig()
		c.viper.OnConfigChange(func(in fsnotify.Event) {
			if err := c.Reload(); err != nil {
				log.Println(err.Error())
			}
			fmt.Println(c)
		})
	}
	return nil
}

func (c *AConfig) watchRemoteConfigOnChannel(ctx context.Context) error {
	err := c.viper.WatchRemoteConfigOnChannel()
	if err != nil {
		return err
	}
	watchChan := c.remoteChan.GetChangeChannel(c)
	for {
		select {
		case <-watchChan:
			err = c.Reload()
			if err != nil {
				fmt.Println("WatchRemote Reload", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *AConfig) Reload() error {
	err := c.load()
	if err != nil {
		return err
	}
	tmp := &AConfig{}
	err = c.viper.Unmarshal(tmp)
	if err != nil {
		return err
	}
	c.Speed = tmp.Speed
	c.Time = tmp.Time
	return nil
}

func (c *AConfig) addProvider() error {
	if c.Type == FILE {
		if c.ConfigFileName == "" {
			return errors.Errorf("configFileName is empty")
		}
		c.viper.SetConfigFile(c.ConfigFileName)
	} else {
		err := c.viper.AddRemoteProvider(c.Provider(), c.Endpoint(), c.Path())
		if err != nil {
			return errors.Errorf("add remote provider failed：%v", err)
		}
		c.remoteChan = remote.New()
		viper.RemoteConfig = c.remoteChan
	}
	return nil
}

func (c *AConfig) Path() string {
	return c.ConfigFileName
}

func (c *AConfig) Endpoint() string {
	return c.EtcdUrl
}

func (c *AConfig) Provider() string {
	return c.RemoteType
}

func (c *AConfig) SecretKeyring() string {
	return ""
}
