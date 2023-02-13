package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/cmd/worker/server"
	"github.com/TomaszDomagala/Allezon/src/cmd/worker/worker"
	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/TomaszDomagala/Allezon/src/pkg/logutils"

	"github.com/TomaszDomagala/Allezon/src/cmd/worker/config"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
)

func main() {
	conf, err := config.New()
	if err != nil {
		panic(err)
	}
	logger, err := logutils.NewLogger("worker", conf.LogLevel)
	if err != nil {
		panic(fmt.Errorf("failed to create logger: %w", err))
	}
	consumer, err := messaging.NewConsumer(logger, conf.KafkaAddresses)
	if err != nil {
		logger.Fatal("Error while creating producer", zap.Error(err))
	}

	client, err := db.NewClientFromAddresses(logger, conf.DBAddresses...)
	if err != nil {
		logger.Fatal("Error while creating database client", zap.Error(err))
	}
	getter := idGetter.NewClient(http.Client{Timeout: 5 * time.Second}, conf.IDGetterAddress, logger)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		wrk := worker.New(worker.Dependencies{
			Logger:   logger,
			Consumer: consumer,
			DB:       client,
			IDGetter: getter,
		})

		if err := wrk.Run(context.Background()); err != nil {
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
