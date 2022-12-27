package server

import (
	"github.com/Shopify/sarama"
	"github.com/gin-gonic/gin"
)

type Server interface {
	Run() error
}

type server struct {
	engine        *gin.Engine
	kafkaProducer sarama.SyncProducer
}

func (s server) Run() error {
	return s.engine.Run()
}

func New(producer sarama.SyncProducer) Server {
	router := gin.Default()
	s := server{engine: router, kafkaProducer: producer}

	router.POST("/user_tags", s.userTagsHandler)
	router.POST("/user_profiles/:cookie", s.userProfilesHandler)
	router.POST("/aggregates", s.aggregatesHandler)

	return s
}
