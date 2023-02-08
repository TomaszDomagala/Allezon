package server

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/dto"

	"github.com/gin-gonic/gin"
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
	if err := s.producer.Send(userTag); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}
