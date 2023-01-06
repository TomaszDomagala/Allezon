package server

import (
	"github.com/Shopify/sarama"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"time"

	ginzap "github.com/gin-contrib/zap"
)

type Server interface {
	Run() error
}

type server struct {
	logger        *zap.Logger
	engine        *gin.Engine
	kafkaProducer sarama.SyncProducer
}

func (s server) Run() error {
	return s.engine.Run()
}

func New(logger *zap.Logger, producer sarama.SyncProducer) Server {
	router := gin.New()

	router.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	s := server{engine: router, kafkaProducer: producer, logger: logger}

	router.POST("/user_tags", s.userTagsHandler)
	router.POST("/user_profiles/:cookie", s.userProfilesHandler)
	router.POST("/aggregates", s.aggregatesHandler)

	return s
}
