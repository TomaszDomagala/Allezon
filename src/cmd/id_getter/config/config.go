package config

import "github.com/spf13/viper"

type Config struct {
	// Server options
	Port     int  `mapstructure:"port"`
	EchoMode bool `mapstructure:"echo_mode"`

	// LogLevel controls the log level of the application.
	LogLevel string `mapstructure:"log_level"`

	// DB options
	DBNullClient bool     `mapstructure:"db_null_client"`
	DBAddresses  []string `mapstructure:"db_addresses"`
}

func field(name string, defaultValue any) {
	_ = viper.BindEnv(name)
	viper.SetDefault(name, defaultValue)
}

func New() (*Config, error) {
	field("port", 8080)
	field("echo_mode", false)

	field("log_level", "debug")

	field("db_null_client", false)
	field("db_addresses", []string{})

	var c Config
	_ = viper.Unmarshal(&c)
	return &c, nil
}
