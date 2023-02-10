package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/dto"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

type userProfilesRequest struct {
	TimeRange string `form:"time_range" binding:"required"`
	Limit     int    `form:"limit,default=200" binding:"required,gte=0,lte=200"`
}

func (s server) userProfilesHandler(c *gin.Context) {
	var req userProfilesRequest
	if err := c.BindQuery(&req); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, errors.New("request must contain a time range"))
		return
	}
	from, to, err := parseTimeRange(dto.TimeRangeMilliPrecisionLayout, req.TimeRange)
	if err != nil {
		s.logger.Error("error parsing time range", zap.Error(err))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	cookie := c.Param("cookie")
	s.logger.Debug("parsed", zap.String("cookie", cookie), zap.Time("from", from), zap.Time("to", to))

	resp, err := s.userProfiles(cookie, from, to, req.Limit)
	if err != nil {
		s.logger.Error("error handling user profiles", zap.Error(err))
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

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

func convertTags(tags []types.UserTag, from, to time.Time, limit int) []dto.UserTagDTO {
	if limit == 0 {
		return []dto.UserTagDTO{}
	}
	toMilli := to.UnixMilli()
	fromMilli := from.UnixMilli()
	selected := make([]dto.UserTagDTO, 0, limit)
	// Tags we get from DB are sorted in ascending order.
	for i := len(tags) - 1; i >= 0 && len(selected) < limit; i-- {
		milli := tags[i].Time.UnixMilli()
		if fromMilli <= milli && milli < toMilli {
			selected = append(selected, dto.IntoUserTagDTO(tags[i]))
		}
	}
	return selected
}

func (s server) userProfiles(cookie string, from, to time.Time, limit int) (dto.UserProfileDTO, error) {
	res, err := s.db.UserProfiles().Get(cookie)
	if err != nil {
		if errors.Is(err, db.KeyNotFoundError) {
			s.logger.Debug("key not found", zap.String("cookie", cookie))
			return dto.UserProfileDTO{
				Cookie: cookie,
			}, nil
		}
		return dto.UserProfileDTO{}, fmt.Errorf("error getting user profiles from db, %w", err)
	}
	s.logger.Debug("got user profiles from db", zap.Any("views", res.Views), zap.Any("buys", res.Buys))

	return dto.UserProfileDTO{
		Cookie: cookie,
		Views:  convertTags(res.Views, from, to, limit),
		Buys:   convertTags(res.Buys, from, to, limit),
	}, nil
}
