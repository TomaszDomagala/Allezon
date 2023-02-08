package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/dto"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"github.com/gin-gonic/gin"
)

type aggregatesRequest struct {
	TimeRange  string   `form:"time_range" binding:"required"`
	Action     string   `form:"action" binding:"required,oneof=BUY VIEW"`
	Aggregates []string `form:"aggregates" binding:"required"`
	Origin     *string  `form:"origin" binding:"-"`
	BrandId    *string  `form:"brand_id" binding:"-"`
	CategoryId *string  `form:"category_id" binding:"-"`
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
	if err := validateAggregatesTimeRange(from, to); err != nil {
		err = fmt.Errorf("error validating time range %s-%s, %w", from, to, err)
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

func validateAggregatesTimeRange(from, to time.Time) error {
	if from.After(to) {
		return fmt.Errorf("from is before to")
	}
	if to.Sub(from) > 10*time.Minute {
		return fmt.Errorf("timerange is larger than 10 minutes")
	}
	if from.Second() > 0 {
		return fmt.Errorf("from is not second aligned")
	}
	if to.Second() > 0 {
		return fmt.Errorf("to is not second aligned")
	}
	return nil
}

func (s server) convertAggregates(req []string) ([]types.Aggregate, error) {
	var err error
	aggregates := make([]types.Aggregate, len(req))
	for i, a := range req {
		aggregates[i], err = dto.ToAggregate(a)
		if err != nil {
			return nil, err
		}
	}

	agg := make(map[types.Aggregate]struct{}, len(aggregates))
	for _, a := range aggregates {
		agg[a] = struct{}{}
	}
	if len(aggregates) != len(agg) {
		return nil, fmt.Errorf("aggregates list contains duplicates")
	}

	return aggregates, nil
}

func (s server) aggregates(aggregates []types.Aggregate, params fetchParams) (dto.AggregatesDTO, error) {
	f, err := s.newFilters(params.origin, params.brandId, params.categoryId)
	if err != nil {
		return dto.AggregatesDTO{}, fmt.Errorf("error creating filters, %w", err)
	}
	res := newAggregatesResponseBuilder(aggregates, params)
	for t := params.from; t.Before(params.to); t = t.Add(time.Minute) {
		aggs, err := s.db.Aggregates().Get(t, params.action)
		var sum, count uint64
		if err != nil {
			if !errors.Is(err, db.KeyNotFoundError) {
				return dto.AggregatesDTO{}, fmt.Errorf("error getting aggregates for time %s, %w", t, err)
			}
		} else {
			sum, count = s.filterAggregates(aggs, f)
		}
		res.appendAggregates(t, sum, count)
	}

	return res.toResponse(), nil
}

func (s server) filterAggregates(aggs []db.ActionAggregates, f filters) (sum uint64, count uint64) {
	for _, agg := range aggs {
		if f.match(agg.Key) {
			sum += agg.Sum
			count += uint64(agg.Count)
		}
	}
	return
}

type aggregatesResponseBuilder struct {
	columns []string
	rows    [][]string

	aggs   []types.Aggregate
	params fetchParams
}

func (b *aggregatesResponseBuilder) toResponse() dto.AggregatesDTO {
	return dto.AggregatesDTO{
		Columns: b.columns,
		Rows:    b.rows,
	}
}

func (b *aggregatesResponseBuilder) appendAggregates(t time.Time, sum uint64, count uint64) {
	row := make([]string, 0, len(b.columns))
	row = append(row, t.Format(dto.TimeRangeSecPrecisionLayout), b.params.action.String())
	if b.params.origin != nil {
		row = append(row, *b.params.origin)
	}
	if b.params.brandId != nil {
		row = append(row, *b.params.origin)
	}
	if b.params.categoryId != nil {
		row = append(row, *b.params.origin)
	}
	for _, a := range b.aggs {
		switch a {
		case types.Count:
			row = append(row, fmt.Sprint(count))
		case types.Sum:
			row = append(row, fmt.Sprint(sum))
		}
	}
	b.rows = append(b.rows, row)
}

func newAggregatesResponseBuilder(aggregates []types.Aggregate, params fetchParams) (res aggregatesResponseBuilder) {
	res.columns = []string{"1m_bucket", "action"}
	if params.origin != nil {
		res.columns = append(res.columns, "origin")
	}
	if params.brandId != nil {
		res.columns = append(res.columns, "brand_id")
	}
	if params.categoryId != nil {
		res.columns = append(res.columns, "category_id")
	}
	for _, a := range aggregates {
		res.columns = append(res.columns, string(a))
	}

	res.aggs = aggregates
	res.params = params

	return res
}

func (s server) newFilters(origin, brandId, categoryId *string) (f filters, err error) {
	f.originId, err = s.getId(idGetter.OriginCollection, origin)
	if err != nil {
		return filters{}, err
	}
	f.categoryId, err = s.getId(idGetter.CategoryCollection, categoryId)
	if err != nil {
		return filters{}, err
	}
	f.brandId, err = s.getId(idGetter.BrandCollection, brandId)
	if err != nil {
		return filters{}, err
	}
	return f, nil
}

func (s server) getId(collection string, elementPtr *string) (*uint16, error) {
	if elementPtr == nil {
		return nil, nil
	}
	id, err := idGetter.GetU16ID(s.idGetter, collection, *elementPtr)
	if err != nil {
		return nil, fmt.Errorf("error getting %s id of filter, %w", collection, err)
	}
	return &id, nil
}

type filters struct {
	originId   *uint16
	brandId    *uint16
	categoryId *uint16
}

func (f filters) match(key db.AggregateKey) bool {
	return checkFilter(f.originId, key.Origin) &&
		checkFilter(f.brandId, key.BrandId) &&
		checkFilter(f.categoryId, key.CategoryId)
}

func checkFilter(f *uint16, k uint16) bool {
	if f == nil {
		return true
	}
	return *f == k
}

type fetchParams struct {
	from   time.Time
	to     time.Time
	action types.Action

	origin     *string
	brandId    *string
	categoryId *string
}
