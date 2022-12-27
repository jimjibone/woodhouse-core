package config

type CoreConfig struct {
	Changed  bool           `yaml:"-"`
	Server   ServerConfig   `yaml:"server"`
	InfluxDB InfluxDBConfig `yaml:"influxdb"`
}

type ServerConfig struct {
	ApiAddr string `yaml:"api-addr"`
	WebAddr string `yaml:"web-addr"`
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
	InfluxDB: InfluxDBConfig{
		Enabled: false,
		Addr:    "localhost:8086",
		Token:   "",
		Org:     "",
		Bucket:  "woodhouse",
	},
}
