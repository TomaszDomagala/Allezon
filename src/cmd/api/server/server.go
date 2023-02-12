package server

import (
	"fmt"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/cmd/api/config"
	"github.com/TomaszDomagala/Allezon/src/cmd/api/middleware"
	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
)

type Server interface {
	Run() error
}

type Dependencies struct {
	Logger   *zap.Logger
	Cfg      *config.Config
	Producer messaging.UserTagsProducer
	DB       db.Client
	IDGetter idGetter.Client
}

type server struct {
	conf     *config.Config
	logger   *zap.Logger
	engine   *gin.Engine
	producer messaging.UserTagsProducer
	db       db.Client
	idGetter idGetter.Client
}

func (s server) Run() error {
	s.logger.Info("Starting server", zap.Int("port", s.conf.Port))
	return s.engine.Run(fmt.Sprintf(":%d", s.conf.Port))
}

func New(deps Dependencies) Server {
	router := gin.New()

	router.Use(ginzap.Ginzap(deps.Logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(deps.Logger, true))
	router.Use(middleware.ExpectationValidator(deps.Logger))

	s := server{
		engine:   router,
		producer: deps.Producer,
		logger:   deps.Logger,
		conf:     deps.Cfg,
		db:       deps.DB,
		idGetter: deps.IDGetter,
	}

	router.GET("/health", s.health)

	router.POST("/user_tags", s.userTagsHandler)
	router.POST("/user_profiles/:cookie", s.userProfilesHandler)
	router.POST("/aggregates", s.aggregatesHandler)

	return s
}
