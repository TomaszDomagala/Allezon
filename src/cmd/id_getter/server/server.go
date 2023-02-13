package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"

	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/api"
	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/config"
	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/db"

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

var ErrorNotFound = errors.New("not found")

func (s server) Run() error {
	s.logger.Info("Starting server", zap.Int("port", s.conf.Port))
	return s.engine.Run(fmt.Sprintf(":%d", s.conf.Port))
}

func (s server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s server) getIDHandler(c *gin.Context) {
	var req api.GetIDRequest

	body, err := c.GetRawData()
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		s.logger.Error("can't unmarshal request", zap.Error(err), zap.ByteString("body", body))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	id, err := s.getID(req.CollectionName, req.Element)
	if err != nil {
		s.logger.Error("can't get id", zap.Error(err), zap.String("collection", req.CollectionName), zap.String("element", req.Element))
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, api.GetIdResponse{ID: int32(id)})
}

// getID returns id of element in category. It tries to find it in cache first, then in database.
// If it's not found in db, it generates new id and caches it.
func (s server) getID(category string, element string) (int, error) {
	if id, inCache := s.checkInCache(category, element); inCache {
		return id, nil
	}

	id, err := s.getIDFromDB(category, element)
	if err != nil {
		if !errors.Is(err, ErrorNotFound) {
			return 0, fmt.Errorf("error while getting id from db: %w", err)
		}
		// element not found in db, save it and return new id.
		id, err = s.saveIDInDB(category, element)
		if err != nil {
			return 0, fmt.Errorf("error while saving id in db: %w", err)
		}
	}

	s.saveInCache(category, element, id)
	return id, nil
}

// getIDFromDB returns id of element in category.
// It fetches the whole list from db and then searches for element in it.
// Index of element in list is returned as its id.
func (s server) getIDFromDB(category string, element string) (int, error) {
	list, err := s.db.GetElements(category)
	if err != nil {
		if errors.Is(err, db.KeyNotFoundError) {
			return 0, fmt.Errorf("error while getting elements from db: %w", ErrorNotFound)
		}
		return 0, fmt.Errorf("error while getting elements from db: %w", err)
	}
	idx := slices.Index(list, element)
	if idx == -1 {
		return 0, fmt.Errorf("element not found in list, %w: (%v, %v)", ErrorNotFound, category, element)
	}
	s.logger.Debug("found id in db", zap.String("category", category), zap.String("element", element), zap.Int("id", idx))
	return idx, nil
}

// saveIDInDB saves element in category and returns its id.
func (s server) saveIDInDB(category string, element string) (int, error) {
	id, err := s.db.AppendElement(category, element)
	if err != nil {
		return 0, fmt.Errorf("error while appending record, %w", err)
	}
	s.logger.Debug("saved id in db", zap.String("category", category), zap.String("element", element), zap.Int("id", id))
	return id, nil

}

func (s server) checkInCache(category string, element string) (int, bool) {
	s.idsCacheMutex.RLock()
	defer s.idsCacheMutex.RUnlock()

	if cache, ok := s.idsCache[category]; ok {
		if id, ok := cache[element]; ok {
			s.logger.Debug("found id in cache", zap.String("category", category), zap.String("element", element), zap.Int("id", id))
			return id, true
		}
	}
	return 0, false
}

func (s server) saveInCache(category string, element string, colLen int) {
	s.idsCacheMutex.Lock()
	defer s.idsCacheMutex.Unlock()

	if _, ok := s.idsCache[category]; !ok {
		s.idsCache[category] = make(map[string]int)
	}
	s.idsCache[category][element] = colLen
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

	router.POST(api.GetIDUrl, s.getIDHandler)

	return s
}
