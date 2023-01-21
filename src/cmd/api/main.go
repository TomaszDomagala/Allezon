package main

import (
	"github.com/TomaszDomagala/Allezon/src/pkg/db"
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

	var producer messaging.UserTagsProducer
	if conf.KafkaNullProducer {
		producer = messaging.NewNullProducer(logger)
	} else {
		producer, err = messaging.NewProducer(logger, conf.KafkaAddresses)
		if err != nil {
			logger.Fatal("Error while creating producer", zap.Error(err))
		}
	}

	getter, err := db.NewClientFromAddresses(conf.DBAddresses)
	if err != nil {
		logger.Fatal("Error while creating database client", zap.Error(err))
	}

	srv := server.New(server.Dependencies{
		Logger:   logger,
		Cfg:      conf,
		Producer: producer,
		DBGetter: getter,
	})

	if err := srv.Run(); err != nil {
		logger.Fatal("Error while running a server", zap.Error(err))
	}
}
