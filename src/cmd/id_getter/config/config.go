package config

import "github.com/spf13/viper"

type Config struct {
	// Server options
	Port     int  `mapstructure:"port"`
	EchoMode bool `mapstructure:"echo_mode"`

	// LoggerDebugLevel enables debug logging. If false, logs are written at Info level.
	LoggerDebugLevel bool `mapstructure:"logger_debug_level"`

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

	field("logger_debug_level", true)

	field("db_null_client", false)
	field("db_addresses", []string{})

	var c Config
	_ = viper.Unmarshal(&c)
	return &c, nil
}
