package server

import (
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/cmd/api/config"
	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"time"

	ginzap "github.com/gin-contrib/zap"
)

type Server interface {
	Run() error
}

type Dependencies struct {
	Logger   *zap.Logger
	Cfg      *config.Config
	Producer messaging.UserTagsProducer
	DBGetter db.Client
}

type server struct {
	conf     *config.Config
	logger   *zap.Logger
	engine   *gin.Engine
	producer messaging.UserTagsProducer
	dbGetter db.Client
}

func (s server) Run() error {
	s.logger.Info("Starting server", zap.Int("port", s.conf.Port))
	return s.engine.Run(fmt.Sprintf(":%d", s.conf.Port))
}

func New(deps Dependencies) Server {
	router := gin.New()

	router.Use(ginzap.Ginzap(deps.Logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(deps.Logger, true))

	s := server{engine: router, producer: deps.Producer, logger: deps.Logger, conf: deps.Cfg, dbGetter: deps.DBGetter}

	router.GET("/health", s.health)

	router.POST("/user_tags", s.userTagsHandler)
	router.POST("/user_profiles/:cookie", s.userProfilesHandler)
	router.POST("/aggregates", s.aggregatesHandler)

	return s
}
