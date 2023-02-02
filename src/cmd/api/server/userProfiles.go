package server

import (
	"errors"
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/pkg/dto"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const timeRangeMilliPrecisionLayout = "2006-01-02T15:04:05.999"

func parseTimeRange(layout, str string) (time.Time, time.Time, error) {
	split := strings.Split(str, "_")
	if len(split) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("expected two parts of a time range, got %d, on %s", len(split), str)
	}
	from, err := time.Parse(layout, split[0])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing from part '%s' of time range '%s', %w", split[0], str, err)
	}
	to, err := time.Parse(layout, split[1])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing to part '%s' of time range '%s', %w", split[1], str, err)
	}
	return from, to, nil
}

func (s server) userProfilesHandler(c *gin.Context) {
	trStr, ok := c.GetQuery("time_range")
	if !ok {
		s.logger.Error("request without time range")
		_ = c.AbortWithError(http.StatusBadRequest, errors.New("request must contain a time range"))
		return
	}
	from, to, err := parseTimeRange(timeRangeMilliPrecisionLayout, trStr)
	if err != nil {
		s.logger.Error("error parsing time range", zap.Error(err))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	limitStr := c.DefaultQuery("limit", "200")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		s.logger.Error("can't convert limit to int request: %s", zap.Error(err), zap.String("limit", limitStr))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	cookie := c.Param("cookie")

	resp, err := s.userProfiles(cookie, from, to, limit)
	if err != nil {
		s.logger.Error("error handling user profiles", zap.Error(err))
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func convertTags(tags []types.UserTag, from, to time.Time, limit int) []dto.UserTagDTO {
	toMilli := to.UnixMilli()
	fromMilli := from.UnixMilli()
	// Tags we get from DB are sorted in ascending order.
	var selected []dto.UserTagDTO
	for _, tag := range tags {
		milli := tag.Time.UnixMilli()
		if fromMilli <= milli && milli < toMilli {
			selected = append(selected, dto.IntoUserTagDTO(tag))
		}
	}
	return selected
}

func (s server) userProfiles(cookie string, from, to time.Time, limit int) (dto.UserProfileDTO, error) {
	res, err := s.db.UserProfiles().Get(cookie)
	if err != nil {
		if errors.Is(err, db.KeyNotFoundError) {
			return dto.UserProfileDTO{
				Cookie: cookie,
			}, nil
		}
		return dto.UserProfileDTO{}, fmt.Errorf("error getting user profiles from db, %w", err)
	}

	return dto.UserProfileDTO{
		Cookie: cookie,
		Views:  convertTags(res.Result.Views, from, to, limit),
		Buys:   convertTags(res.Result.Buys, from, to, limit),
	}, nil
}
