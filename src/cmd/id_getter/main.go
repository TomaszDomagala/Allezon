package main

import (
	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/db"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/config"
	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/server"
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

	client, err := db.NewClientFromAddresses(conf.DBAddresses)
	if err != nil {
		logger.Fatal("Error while creating database client", zap.Error(err))
	}

	srv := server.New(server.Dependencies{
		Logger: logger,
		Cfg:    conf,
		DB:     client,
	})

	if err := srv.Run(); err != nil {
		logger.Fatal("Error while running a server", zap.Error(err))
	}
}
