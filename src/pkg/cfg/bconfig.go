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

type BConfig struct {
	remoteChan     remote.Config `yaml:"-"`
	ConfigFileName string
	Type           Type
	viper          *viper.Viper `yaml:"-"`
	Node           string
	User           string
	Pass           int

	//
	EtcdUrl    string `yaml:"-"`
	RemoteType string `yaml:"-"`
}

func (c *BConfig) Parse() error {
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

func (c *BConfig) WatchConfig(ctx context.Context) error {
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

func (c *BConfig) watchRemoteConfigOnChannel(ctx context.Context) error {
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

func (c *BConfig) Reload() error {
	err := c.load()
	if err != nil {
		return err
	}
	tmp := &BConfig{}
	err = c.viper.Unmarshal(tmp)
	if err != nil {
		return err
	}
	c.Node = tmp.Node
	c.User = tmp.User
	c.Pass = tmp.Pass
	return nil
}

func (c *BConfig) addProvider() error {
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

func (c *BConfig) load() error {
	if c.Type == FILE {
		return c.viper.ReadInConfig()
	}
	return c.viper.ReadRemoteConfig()
}

func (c *BConfig) Path() string {
	return c.ConfigFileName
}

func (c *BConfig) Endpoint() string {
	return c.EtcdUrl
}

func (c *BConfig) Provider() string {
	return c.RemoteType
}

func (c *BConfig) SecretKeyring() string {
	return ""
}
