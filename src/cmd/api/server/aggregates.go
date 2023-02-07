package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/dto"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"github.com/gin-gonic/gin"
)

type aggregatesRequest struct {
	TimeRange  string   `form:"time_range" binding:"required"`
	Action     string   `form:"action" binding:"required"`
	Aggregates []string `form:"aggregates" binding:"required"`
	Origin     *string  `form:"origin" binding:"-"`
	BrandId    *string  `form:"brand_id" binding:"-"`
	CategoryId *string  `form:"category_id" binding:"-"`
}

type aggregatesResponse struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

func (s server) aggregatesHandler(c *gin.Context) {
	var req aggregatesRequest
	if err := c.BindQuery(&req); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	action, err := dto.ToAction(req.Action)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	aggregates, err := s.convertAggregates(req.Aggregates)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	from, to, err := parseTimeRange(dto.TimeRangeSecPrecisionLayout, req.TimeRange)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	resp, err := s.aggregates(
		aggregates,
		fetchParams{
			from:       from,
			to:         to,
			action:     action,
			origin:     req.Origin,
			brandId:    req.BrandId,
			categoryId: req.CategoryId,
		},
	)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (s server) convertAggregates(req []string) ([]aggregate, error) {
	var err error
	aggregates := make([]aggregate, len(req))
	for i, a := range req {
		aggregates[i], err = toAggregate(a)
		if err != nil {
			return nil, err
		}
	}

	agg := make(map[aggregate]struct{}, len(aggregates))
	for _, a := range aggregates {
		agg[a] = struct{}{}
	}
	if len(aggregates) != len(agg) {
		return nil, fmt.Errorf("aggregates list contains duplicates")
	}

	return aggregates, nil
}

func (s server) aggregates(aggregates []aggregate, params fetchParams) (aggregatesResponse, error) {
	res := newAggregatesResponse(aggregates, params)
	for _, a := range aggregates {
		actionAggregates, err := s.actionAggregates(a, params)
		if err != nil {
			return aggregatesResponse{}, fmt.Errorf("error parsing aggregates of type %s, %w", a, err)
		}
		res.appendActionAggregates(a, actionAggregates)
	}
	return res, nil
}

func newAggregatesResponse(aggregates []aggregate, params fetchParams) (res aggregatesResponse) {
	res.Columns = []string{"1m_bucket"}
}

func (r aggregatesResponse) appendActionAggregates(a aggregate, aggregates interface{}) {

}

func (s server) actionAggregates(a aggregate, params fetchParams) (interface{}, interface{}) {

}

type fetchParams struct {
	from   time.Time
	to     time.Time
	action types.Action

	origin     *string
	brandId    *string
	categoryId *string
}

type aggregate uint8

const (
	sum aggregate = iota
	count
)

func (a aggregate) String() string {
	switch a {
	case count:
		return "count"
	case sum:
		return "sum_price"
	default:
		return "unknown"
	}
}

func toAggregate(s string) (aggregate, error) {
	switch s {
	case "count":
		return count, nil
	case "sum_price":
		return sum, nil
	default:
		return 0, fmt.Errorf("can't convert to aggregate: %s", s)
	}
}
