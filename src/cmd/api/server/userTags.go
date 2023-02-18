package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/TomaszDomagala/Allezon/src/pkg/dto"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
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
		return s.addUserTag(&userTag)
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

const userTagLimit = 200

const userTagGCThreshold = int(userTagLimit * float32(1.1))

func (s server) addUserTag(tag *types.UserTag) error {
	newLen, err := s.profilesDB.UserProfiles().Add(tag)
	if err != nil {
		return fmt.Errorf("error updating userTags, %w", err)
	}
	if newLen > userTagGCThreshold {
		go s.removeOldUserTags(tag.Cookie, tag.Action)
	}
	return nil
}

func (s server) removeOldUserTags(cookie string, action types.Action) {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 500 * time.Millisecond
	bo.Multiplier = 3
	bo.MaxElapsedTime = time.Minute
	bo.MaxInterval = 10 * time.Second

	err := backoff.Retry(func() error {
		if err := s.profilesDB.UserProfiles().RemoveOverLimit(cookie, action, userTagLimit); err != nil {
			s.logger.Debug("error cleaning user profiles", zap.String("cookie", cookie), zap.Stringer("action", action), zap.Error(err))
			return err
		}
		return nil
	}, bo)
	if err != nil {
		s.logger.Error("timeout cleaning user profiles", zap.Any("cookie", cookie), zap.Stringer("action", action), zap.Error(err))
	}
}
