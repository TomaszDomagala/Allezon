package config

import "github.com/spf13/viper"

type Config struct {
	// Server options
	Port int `mapstructure:"port"`

	// LogLevel controls the log level of the application.
	LogLevel string `mapstructure:"log_level"`

	// Kafka options
	KafkaAddresses []string `mapstructure:"kafka_addresses"`

	// DB options
	DBAggregatesAddresses []string `mapstructure:"db_aggregates_addresses"`

	// ID Getter
	IDGetterAddress string `mapstructure:"id_getter_address"`
}

func field(name string, defaultValue any) {
	_ = viper.BindEnv(name)
	viper.SetDefault(name, defaultValue)
}

func New() (*Config, error) {
	field("port", 8080)

	field("log_level", "debug")

	field("kafka_addresses", []string{})
	field("db_aggregates_addresses", []string{})
	field("id_getter_address", "")

	var c Config
	_ = viper.Unmarshal(&c)
	return &c, nil
}
