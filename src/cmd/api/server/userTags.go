package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
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

	errGrp.Go(func() (err error) {
		start := time.Now()
		const timeout = 70 * time.Millisecond
		const sleep = 10 * time.Millisecond
		for time.Since(start) <= timeout {
			if updateErr := updateUserProfile(userTag, s.db.UserProfiles()); updateErr == nil {
				return nil
			} else {
				err = multierror.Append(err, updateErr)
				time.Sleep(sleep)
			}
		}
		return fmt.Errorf("adding user profiles timeout, %w", err)
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

const maxLen = 200

func updateUserProfile(tag types.UserTag, userProfiles db.UserProfileClient) error {
	up, err := userProfiles.Get(tag.Cookie)
	if err != nil && !errors.Is(err, db.KeyNotFoundError) {
		return fmt.Errorf("error getting tag, %w", err)
	}
	var arrPtr *[]types.UserTag
	switch tag.Action {
	case types.Buy:
		arrPtr = &up.Result.Buys
	case types.View:
		arrPtr = &up.Result.Views
	default:
		return fmt.Errorf("unknown action, %d", tag.Action)
	}
	var newArr []types.UserTag
	for i, t := range *arrPtr {
		if t.Time.Before(tag.Time) {
			newArr = slices.Insert(*arrPtr, i, tag)
			break
		}
	}
	if newArr == nil {
		newArr = append(*arrPtr, tag)
	}
	if len(newArr) > maxLen {
		newArr = newArr[:maxLen]
	}
	*arrPtr = newArr

	if err := userProfiles.Update(tag.Cookie, up.Result, up.Generation); err != nil {
		return fmt.Errorf("error updating tag, %w", err)
	}
	return nil
}
