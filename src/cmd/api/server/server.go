package server

import (
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/cmd/api/config"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"time"

	ginzap "github.com/gin-contrib/zap"
)

type Server interface {
	Run() error
}

type server struct {
	conf     *config.Config
	logger   *zap.Logger
	engine   *gin.Engine
	producer messaging.UserTagsProducer
}

func (s server) Run() error {
	s.logger.Info("Starting server", zap.Int("port", s.conf.Port))
	return s.engine.Run(fmt.Sprintf(":%d", s.conf.Port))
}

func New(logger *zap.Logger, cfg *config.Config, producer messaging.UserTagsProducer) Server {
	router := gin.New()

	router.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	s := server{engine: router, producer: producer, logger: logger, conf: cfg}

	router.GET("/health", s.health)

	router.POST("/user_tags", s.userTagsHandler)
	router.POST("/user_profiles/:cookie", s.userProfilesHandler)
	router.POST("/aggregates", s.aggregatesHandler)

	return s
}
