package config

import "github.com/spf13/viper"

type Config struct {
	EchoMode     bool `mapstructure:"echo_mode"`
	NullProducer bool `mapstructure:"null_producer"`
}

func field(name string, defaultValue any) {
	_ = viper.BindEnv(name)
	viper.SetDefault(name, defaultValue)
}

func New() (*Config, error) {
	field("echo_mode", false)
	field("null_producer", false)

	var c Config
	_ = viper.Unmarshal(&c)
	return &c, nil
}
