package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"

	"github.com/TomaszDomagala/Allezon/src/cmd/api/config"
	"github.com/TomaszDomagala/Allezon/src/cmd/api/server"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
)

func main() {
	conf, err := config.New()
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	logger := newLogger(conf)

	logger.Info("Initializing messaging", zap.Strings("addresses", conf.KafkaAddresses))
	err = messaging.Initialize(logger, conf.KafkaAddresses, &sarama.TopicDetail{
		NumPartitions:     conf.KafkaNumPartitions,
		ReplicationFactor: conf.KafkaReplicationFactor,
	})
	if err != nil {
		logger.Fatal("failed to initialize messaging", zap.Error(err), zap.Strings("addresses", conf.KafkaAddresses))
	}

	var producer messaging.UserTagsProducer
	if conf.KafkaNullProducer {
		logger.Info("Using null producer")
		producer = messaging.NewNullProducer(logger)
	} else {
		logger.Info("Using kafka producer", zap.Strings("addresses", conf.KafkaAddresses))
		producer, err = messaging.NewProducer(logger, conf.KafkaAddresses)
		if err != nil {
			logger.Fatal("Error while creating producer", zap.Error(err))
		}
	}

	var dbClient db.Client
	if conf.DBNullClient {
		logger.Info("Using null database client")
		dbClient = db.NewNullClient(logger)
	} else {
		logger.Info("Using aerospike database client, addresses: ", zap.Strings("addresses", conf.DBAddresses))
		dbClient, err = db.NewClientFromAddresses(logger, conf.DBAddresses...)
		if err != nil {
			logger.Fatal("Error while creating database client", zap.Error(err))
		}
	}

	var getter idGetter.Client
	if conf.IDGetterNullClient {
		logger.Info("Using null id getter client")
		getter = idGetter.NewNullClient(logger)
	} else {
		logger.Info("Using id getter client", zap.String("address", conf.IDGetterAddress))
		getter = idGetter.NewClient(http.Client{Timeout: time.Second}, conf.IDGetterAddress)
	}

	srv := server.New(server.Dependencies{
		Logger:   logger,
		Cfg:      conf,
		Producer: producer,
		DB:       dbClient,
		IDGetter: getter,
	})

	if err := srv.Run(); err != nil {
		logger.Fatal("Error while running a server", zap.Error(err))
	}
}

// newLogger returns a logger based on the application configuration.
func newLogger(conf *config.Config) *zap.Logger {
	encoderConfig := ecszap.NewDefaultEncoderConfig()

	level := zap.InfoLevel
	if conf.LoggerDebugLevel {
		level = zap.DebugLevel
	}

	core := ecszap.NewCore(encoderConfig, os.Stdout, level)
	logger := zap.New(core, zap.AddCaller())

	return logger
}
