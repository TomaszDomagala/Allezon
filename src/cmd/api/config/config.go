package config

import "github.com/spf13/viper"

type Config struct {
	// Server options
	Port     int  `mapstructure:"port"`
	EchoMode bool `mapstructure:"echo_mode"`

	// LogLevel controls the log level of the application.
	LogLevel string `mapstructure:"log_level"`

	// Kafka options
	KafkaNullProducer      bool     `mapstructure:"kafka_null_producer"`
	KafkaAddresses         []string `mapstructure:"kafka_addresses"`
	KafkaNumPartitions     int32    `mapstructure:"kafka_num_partitions"`
	KafkaReplicationFactor int16    `mapstructure:"kafka_replication_factor"`

	// DB options
	DBAddresses  []string `mapstructure:"db_addresses"`
	DBNullClient bool     `mapstructure:"db_null_client"`

	// ID Getter
	IDGetterAddress    string `mapstructure:"id_getter_address"`
	IDGetterNullClient bool   `mapstructure:"id_getter_null_client"`
}

func field(name string, defaultValue any) {
	_ = viper.BindEnv(name)
	viper.SetDefault(name, defaultValue)
}

func New() (*Config, error) {
	field("port", 8080)
	field("echo_mode", false)

	field("log_level", "info")

	field("kafka_null_producer", false)
	field("kafka_addresses", []string{})
	field("kafka_num_partitions", 1)
	field("kafka_replication_factor", 1)

	field("db_addresses", []string{})
	field("db_null_client", false)

	field("id_getter_address", "")
	field("id_getter_null_client", false)

	var c Config
	_ = viper.Unmarshal(&c)
	return &c, nil
}
