package config

import "fmt"

type CoreConfig struct {
	Changed bool         `yaml:"-"`
	Server  ServerConfig `yaml:"server"`
}

type ServerConfig struct {
	ApiAddr string `yaml:"api-addr"`
	WebAddr string `yaml:"web-addr"`
}

var LoadedConfig CoreConfig = defaultConfig

var defaultConfig = CoreConfig{
	Server: ServerConfig{
		ApiAddr: "localhost:4000",
		WebAddr: "localhost:4080",
	},
}

// Returns an error if the config is not valid.
func (c CoreConfig) Verify() error {
	if err := c.Server.Verify(); err != nil {
		return err
	}
	return nil
}

// Returns an error if the config is not valid.
func (c ServerConfig) Verify() error {
	if c.ApiAddr == "" {
		return fmt.Errorf("server.api-addr must be defined")
	}
	if c.WebAddr == "" {
		return fmt.Errorf("server.web-addr must be defined")
	}
	return nil
}
