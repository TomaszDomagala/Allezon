package main

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/logutils"

	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/config"
	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/server"
)

func main() {
	conf, err := config.New()
	if err != nil {
		panic(err)
	}
	logger, err := logutils.NewLogger("idgetter", conf.LogLevel)
	if err != nil {
		panic(fmt.Errorf("failed to create logger: %w", err))
	}

	logger.Info("Config loaded: ", zap.Any("config", conf))

	client, err := db.NewClientFromAddresses(conf.DBAddresses...)
	if err != nil {
		logger.Fatal("Error while creating database client", zap.Error(err))
	}
	logger.Info("Database client created")

	srv := server.New(server.Dependencies{
		Logger: logger,
		Cfg:    conf,
		DB:     client,
	})

	if err := srv.Run(); err != nil {
		logger.Fatal("Error while running a server", zap.Error(err))
	}
}
