package main

import (
	"github.com/Shopify/sarama"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/cmd/api/config"
	"github.com/TomaszDomagala/Allezon/src/cmd/api/server"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	conf, err := config.New()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	var producer sarama.SyncProducer
	if conf.NullProducer {
		producer = messaging.NewNullProducer(logger)
	} else {
		producer, err = newProducer()
		if err != nil {
			logger.Fatal("Error while creating producer", zap.Error(err))
		}
	}

	srv := server.New(logger, producer)

	if err := srv.Run(); err != nil {
		//log.Fatalf("Error while running a server, %s", err)
		logger.Fatal("Error while running a server", zap.Error(err))
	}
}
