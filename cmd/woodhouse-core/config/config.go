package config

type CoreConfig struct {
	Changed bool          `yaml:"-"`
	Server  ServerConfig  `yaml:"server"`
	History HistoryConfig `yaml:"history"`
}

type ServerConfig struct {
	ApiAddr string `yaml:"api-addr"`
	WebAddr string `yaml:"web-addr"`
}

type HistoryConfig struct {
	InfluxDBAddr  string `yaml:"influxdb-addr"`
	InfluxDBToken string `yaml:"influxdb-token"`
}

var LoadedConfig CoreConfig = defaultConfig

var defaultConfig = CoreConfig{
	Server: ServerConfig{
		ApiAddr: "localhost:4000",
		WebAddr: "localhost:4080",
	},
	History: HistoryConfig{
		InfluxDBAddr:  "localhost:8086",
		InfluxDBToken: "",
	},
}
