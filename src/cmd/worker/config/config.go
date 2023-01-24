package config

import "github.com/spf13/viper"

type Config struct {
	// Server options
	Port int `mapstructure:"port"`

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
	field("kafka_addresses", []string{})
	field("db_addresses", []string{})
	field("port", 8080)

	var c Config
	_ = viper.Unmarshal(&c)
	return &c, nil
}
