package cfg

import (
	"context"

	"github.com/spf13/viper"
)

type Config struct {
	AConfig *AConfig
	BConfig *BConfig
}

func NewConfig() *Config {
	return &Config{
		AConfig: &AConfig{
			Type:           FILE,
			ConfigFileName: "./src/configs/aconfig.yaml",
			viper:          viper.New(),
			EtcdUrl:        "127.0.0.1:2379",
			RemoteType:     "etcd"},
		BConfig: &BConfig{
			Type:           FILE,
			ConfigFileName: "./src/configs/bconfig.yaml",
			viper:          viper.New(),
			EtcdUrl:        "127.0.0.1:2379",
			RemoteType:     "etcd"},
	}
}

func (c *Config) Parse() error {
	err := c.AConfig.Parse()
	if err != nil {
		return err
	}
	if err = c.BConfig.Parse(); err != nil {
		return err
	}
	return nil
}

func (c *Config) WatchConfig(ctx context.Context) {
	go c.AConfig.WatchConfig(ctx)
	go c.BConfig.WatchConfig(ctx)
}
