package config

import "github.com/spf13/viper"

type Config struct {
	// Server options
	Port     int  `mapstructure:"port"`
	EchoMode bool `mapstructure:"echo_mode"`

	// Kafka options
	KafkaNullProducer bool     `mapstructure:"kafka_null_producer"`
	KafkaAddresses    []string `mapstructure:"kafka_addresses"`

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
	field("echo_mode", false)

	field("kafka_null_producer", false)
	field("kafka_addresses", []string{})
	
	field("db_addresses", []string{})

	field("id_getter_address", "")

	var c Config
	_ = viper.Unmarshal(&c)
	return &c, nil
}
