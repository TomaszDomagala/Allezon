package main

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"go.elastic.co/ecszap"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/cmd/worker/server"
	"github.com/TomaszDomagala/Allezon/src/cmd/worker/worker"
	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"

	"github.com/TomaszDomagala/Allezon/src/cmd/worker/config"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
)

func main() {
	conf, err := config.New()
	if err != nil {
		panic(err)
	}
	logger := newLogger(conf)

	consumer, err := messaging.NewConsumer(logger, conf.KafkaAddresses)
	if err != nil {
		logger.Fatal("Error while creating producer", zap.Error(err))
	}

	client, err := db.NewClientFromAddresses(logger, conf.DBAddresses...)
	if err != nil {
		logger.Fatal("Error while creating database client", zap.Error(err))
	}
	getter := idGetter.NewClient(http.Client{Timeout: time.Second}, conf.IDGetterAddress)

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

// newLogger returns a logger based on the application configuration.
func newLogger(conf *config.Config) *zap.Logger {
	encoderConfig := ecszap.NewDefaultEncoderConfig()

	level := zap.DebugLevel

	core := ecszap.NewCore(encoderConfig, os.Stdout, level)
	logger := zap.New(core, zap.AddCaller())
	logger = logger.With(zap.String("app", "worker"))

	return logger
}
