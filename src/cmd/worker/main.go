package main

import (
	"github.com/TomaszDomagala/Allezon/src/cmd/worker/server"
	"github.com/TomaszDomagala/Allezon/src/cmd/worker/worker"
	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"go.uber.org/zap"
	"sync"

	"github.com/TomaszDomagala/Allezon/src/cmd/worker/config"
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

	consumer, err := messaging.NewConsumer(logger, conf.KafkaAddresses)
	if err != nil {
		logger.Fatal("Error while creating producer", zap.Error(err))
	}

	client, err := db.NewClientFromAddresses(conf.DBAddresses)
	if err != nil {
		logger.Fatal("Error while creating database client", zap.Error(err))
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		wrk := worker.New(worker.Dependencies{
			Logger:   logger,
			Consumer: consumer,
			DB:       client,
		})

		if err := wrk.Run(); err != nil {
			logger.Fatal("Error while running a worker", zap.Error(err))
		}
	}()

	go func() {
		defer wg.Done()

		srv := server.New(server.Dependencies{
			Logger: logger,
			Port:   conf.Port,
		})

		if err := srv.Run(); err != nil {
			logger.Fatal("Error while running a server", zap.Error(err))
		}
	}()

	wg.Wait()
}
