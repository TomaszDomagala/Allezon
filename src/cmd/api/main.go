package main

import (
	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"go.uber.org/zap"
	"net/http"
	"time"

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
		logger.Info("Using null producer")
		producer = messaging.NewNullProducer(logger)
	} else {
		logger.Info("Using kafka producer")
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
		logger.Info("Using aerospike database client")
		dbClient, err = db.NewClientFromAddresses(conf.DBAddresses)
		if err != nil {
			logger.Fatal("Error while creating database client", zap.Error(err))
		}
	}

	var getter idGetter.Client
	if conf.IDGetterNullClient {
		logger.Info("Using null id getter client")
		getter = idGetter.NewNullClient(logger)
	} else {
		logger.Info("Using id getter client")
		getter = idGetter.NewClient(&http.Client{Timeout: time.Second}, conf.IDGetterAddress)
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
