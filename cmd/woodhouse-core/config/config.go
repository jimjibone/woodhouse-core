package config

import (
	"fmt"
	"os"
)

type CoreConfig struct {
	Changed      bool         `yaml:"-"`
	InstanceName string       `yaml:"instance-name"`
	Server       ServerConfig `yaml:"server"`
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
func (c *CoreConfig) Verify() error {
	// Default the instance name to the system hostname if not set.
	if c.InstanceName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("instance-name not set and failed to get hostname: %w", err)
		}
		c.InstanceName = hostname
		c.Changed = true
	}
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
