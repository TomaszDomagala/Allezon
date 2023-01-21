package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/api"
	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/config"
	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/db"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"net/http"
	"sync"
	"time"

	ginzap "github.com/gin-contrib/zap"
)

type Server interface {
	Run() error
}

type Dependencies struct {
	Logger *zap.Logger
	Cfg    *config.Config
	DB     db.Client
}

type server struct {
	conf          *config.Config
	logger        *zap.Logger
	engine        *gin.Engine
	db            db.Client
	idsCache      map[string]map[string]int
	idsCacheMutex *sync.RWMutex
}

func (s server) Run() error {
	s.logger.Info("Starting server", zap.Int("port", s.conf.Port))
	return s.engine.Run(fmt.Sprintf(":%d", s.conf.Port))
}

func (s server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s server) getId(c *gin.Context) {
	var req api.GetIdRequest

	body, err := c.GetRawData()
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		s.logger.Error("can't unmarshal request: %s", zap.Error(err), zap.ByteString("body", body))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	id, err := s.getIdHelper(req.CollectionName, req.Element)
	if err != nil {
		s.logger.Error("couldn't get id, %s", zap.Error(err))
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, api.GetIdResponse{Id: int32(id)})
	return
}

func (s server) getIdHelper(name string, element string) (int, error) {
	if id, inCache := s.checkInCache(name, element); inCache {
		return id, nil
	}
	colLen, err := s.db.Append(name, element)
	if err != nil {
		if errors.Is(err, db.ElementExists) {
			return s.getIdFromDb(name, element)
		}
		return 0, fmt.Errorf("error while appending record %w", err)
	}

	s.saveInCache(name, element, colLen)
	return colLen, nil
}

func (s server) checkInCache(name string, element string) (int, bool) {
	s.idsCacheMutex.RLock()
	defer s.idsCacheMutex.RUnlock()

	if cache, ok := s.idsCache[name]; ok {
		if id, ok := cache[element]; ok {
			return id, true
		}
	}
	return 0, false
}

func (s server) getIdFromDb(name string, element string) (int, error) {
	list, err := s.db.Get(name)
	if err != nil {
		return 0, fmt.Errorf("couldn't get list from db, %w", err)
	}
	idx := slices.Index(list, element)
	if idx == -1 {
		return 0, fmt.Errorf("element not found in list, %w", err)
	}
	s.saveListInCache(name, list)

	return idx, nil
}

func (s server) saveInCache(name string, element string, colLen int) {
	s.idsCacheMutex.Lock()
	defer s.idsCacheMutex.Unlock()

	if cache, ok := s.idsCache[name]; ok {
		cache[element] = colLen
	} else {
		s.idsCache[name] = map[string]int{
			element: colLen,
		}
	}
}

func (s server) saveListInCache(name string, list []string) {
	s.idsCacheMutex.Lock()
	defer s.idsCacheMutex.Unlock()

	cache, ok := s.idsCache[name]
	if !ok {
		cache = make(map[string]int, len(list))
	}
	for i, el := range list {
		cache[el] = i
	}
	s.idsCache[name] = cache
}

func New(deps Dependencies) Server {
	router := gin.New()

	router.Use(ginzap.Ginzap(deps.Logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(deps.Logger, true))

	s := server{
		engine:        router,
		logger:        deps.Logger,
		conf:          deps.Cfg,
		db:            deps.DB,
		idsCache:      make(map[string]map[string]int),
		idsCacheMutex: &sync.RWMutex{},
	}

	router.GET("/health", s.health)

	router.POST("/get_id", s.getId)

	return s
}
