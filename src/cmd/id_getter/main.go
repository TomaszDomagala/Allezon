package main

import (
	"os"

	"go.elastic.co/ecszap"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/db"

	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/config"
	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/server"
)

func main() {
	conf, err := config.New()
	if err != nil {
		panic(err)
	}
	logger := newLogger(conf)

	logger.Info("Config loaded: ", zap.Any("config", conf))

	client, err := db.NewClientFromAddresses(conf.DBAddresses)
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

// newLogger returns a logger based on the application configuration.
func newLogger(conf *config.Config) *zap.Logger {
	encoderConfig := ecszap.NewDefaultEncoderConfig()

	level := zap.InfoLevel
	if conf.LoggerDebugLevel {
		level = zap.DebugLevel
	}

	core := ecszap.NewCore(encoderConfig, os.Stdout, level)
	logger := zap.New(core, zap.AddCaller())
	logger = logger.With(zap.String("app", "idgetter"))

	return logger
}
