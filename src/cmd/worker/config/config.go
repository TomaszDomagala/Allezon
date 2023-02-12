package config

import "github.com/spf13/viper"

type Config struct {
	// Server options
	Port int `mapstructure:"port"`

	// LoggerDebugLevel enables debug logging. If false, logs are written at Info level.
	LoggerDebugLevel bool `mapstructure:"logger_debug_level"`

	// Kafka options
	KafkaAddresses []string `mapstructure:"kafka_addresses"`

	// DB options
	DBAddresses []string `mapstructure:"db_addresses"`

	// ID Getter
	IDGetterAddress string `mapstructure:"id_getter_address"`
}

func field(name string, defaultValue any) {
	_ = viper.BindEnv(name)
	viper.SetDefault(name, defaultValue)
}

func New() (*Config, error) {
	field("port", 8080)

	field("logger_debug_level", true)

	field("kafka_addresses", []string{})
	field("db_addresses", []string{})
	field("id_getter_address", "")

	var c Config
	_ = viper.Unmarshal(&c)
	return &c, nil
}
