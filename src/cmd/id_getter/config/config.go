package config

import "github.com/spf13/viper"

type Config struct {
	// Server options
	Port     int  `mapstructure:"port"`
	EchoMode bool `mapstructure:"echo_mode"`

	// DB options
	DBAddresses []string `mapstructure:"db_addresses"`
}

func field(name string, defaultValue any) {
	_ = viper.BindEnv(name)
	viper.SetDefault(name, defaultValue)
}

func New() (*Config, error) {
	field("port", 8080)
	field("echo_mode", false)

	field("db_addresses", []string{})

	var c Config
	_ = viper.Unmarshal(&c)
	return &c, nil
}
