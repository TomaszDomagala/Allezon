package config

import "github.com/spf13/viper"

type Config struct {
	// Server options
	Port     int  `mapstructure:"port"`
	EchoMode bool `mapstructure:"echo_mode"`

	// Kafka options
	KafkaNullProducer bool     `mapstructure:"kafka_null_producer"`
	KafkaAddresses    []string `mapstructure:"kafka_addresses"`
}

func field(name string, defaultValue any) {
	_ = viper.BindEnv(name)
	viper.SetDefault(name, defaultValue)
}

func New() (*Config, error) {
	field("port", 8080)
	field("echo_mode", false)

	field("kafka_null_producer", false)
	field("kafka_addresses", []string{})

	var c Config
	_ = viper.Unmarshal(&c)
	return &c, nil
}
