package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/TomaszDomagala/Allezon/src/pkg/dto"
)

func (s server) userTagsHandler(c *gin.Context) {
	var req dto.UserTagDTO
	if err := c.BindJSON(&req); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	s.logger.Debug("handling user tag request", zap.Any("request", req))

	userTag, err := dto.FromUserTagDTO(req)
	if err != nil {
		body, err := c.GetRawData()
		if err != nil {
			s.logger.Error("can't get request body", zap.Error(err))
		}
		s.logger.Error("can't convert request to user tag: %s", zap.Error(err), zap.ByteString("body", body))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	var errGrp errgroup.Group

	errGrp.Go(func() error {
		if _, err := s.db.UserProfiles().Add(userTag); err != nil {
			return fmt.Errorf("error updating userTags, %w", err)
		}
		return nil
	})
	errGrp.Go(func() error {
		return s.producer.Send(userTag)
	})

	if err := errGrp.Wait(); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}
