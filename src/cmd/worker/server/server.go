package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"time"

	ginzap "github.com/gin-contrib/zap"
)

type Server interface {
	Run() error
}

type Dependencies struct {
	Logger *zap.Logger
	Port   int
}

type server struct {
	logger *zap.Logger
	engine *gin.Engine
	port   int
}

func (s server) Run() error {
	s.logger.Info("Starting server", zap.Int("port", s.port))
	return s.engine.Run(fmt.Sprintf(":%d", s.port))
}

func (s server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func New(deps Dependencies) Server {
	router := gin.New()

	router.Use(ginzap.Ginzap(deps.Logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(deps.Logger, true))

	s := server{engine: router, logger: deps.Logger}

	router.GET("/health", s.health)

	return s
}
