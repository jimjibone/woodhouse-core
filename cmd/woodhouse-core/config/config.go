package config

import "fmt"

type CoreConfig struct {
	Changed  bool           `yaml:"-"`
	Server   ServerConfig   `yaml:"server"`
	Stores   StoresConfig   `yaml:"stores"`
	InfluxDB InfluxDBConfig `yaml:"influxdb"`
}

type ServerConfig struct {
	ApiAddr string `yaml:"api-addr"`
	WebAddr string `yaml:"web-addr"`
}

type StoresConfig struct {
	DeviceStoreEnabled bool   `yaml:"device-store-enabled"`
	DeviceStorePath    string `yaml:"device-store-path"`
}

type InfluxDBConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"`
	Token   string `yaml:"token"`
	Org     string `yaml:"org"`
	Bucket  string `yaml:"bucket"`
}

var LoadedConfig CoreConfig = defaultConfig

var defaultConfig = CoreConfig{
	Server: ServerConfig{
		ApiAddr: "localhost:4000",
		WebAddr: "localhost:4080",
	},
	Stores: StoresConfig{
		DeviceStoreEnabled: false,
		DeviceStorePath:    "woodhouse-devices.json",
	},
	InfluxDB: InfluxDBConfig{
		Enabled: false,
		Addr:    "localhost:8086",
		Token:   "",
		Org:     "",
		Bucket:  "woodhouse",
	},
}

// Returns an error if the config is not valid.
func (c CoreConfig) Verify() error {
	if err := c.Server.Verify(); err != nil {
		return err
	}
	if err := c.Stores.Verify(); err != nil {
		return err
	}
	if err := c.InfluxDB.Verify(); err != nil {
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

// Returns an error if the config is not valid.
func (c StoresConfig) Verify() error {
	if c.DeviceStorePath == "" {
		return fmt.Errorf("stores.device-store-path must be defined")
	}
	return nil
}

// Returns an error if the config is not valid.
func (c InfluxDBConfig) Verify() error {
	if c.Enabled {
		if c.Addr == "" {
			return fmt.Errorf("influxdb.addr must be defined")
		}
		if c.Token == "" {
			return fmt.Errorf("influxdb.token must be defined")
		}
		if c.Org == "" {
			return fmt.Errorf("influxdb.org must be defined")
		}
		if c.Bucket == "" {
			return fmt.Errorf("influxdb.bucket must be defined")
		}
	}
	return nil
}
