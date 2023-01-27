package worker

import (
	"errors"
	"fmt"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"go.uber.org/zap"
)

type aggregatesProcessor struct {
	aggregates db.AggregatesClient
	idGetter   idGetter.Client
	base       baseProcessor
}

func newAggregatesProcessor(aggregates db.AggregatesClient, idGetter idGetter.Client, logger *zap.Logger) aggregatesProcessor {
	a := aggregatesProcessor{
		aggregates: aggregates,
		idGetter:   idGetter,
	}
	a.base = newBaseProcessor(a.processTagOnce, logger)
	return a
}

func (p aggregatesProcessor) run(tagsChan <-chan types.UserTag) {
	p.base.run(tagsChan)
}

func (p aggregatesProcessor) processTagOnce(tag types.UserTag) error {
	categoryId, err := p.idGetter.GetId(idGetter.CategoryCollection, tag.ProductInfo.CategoryId)
	if err != nil {
		return fmt.Errorf("error getting category id of tag, %w", err)
	}
	brandId, err := p.idGetter.GetId(idGetter.BrandCollection, tag.ProductInfo.BrandId)
	if err != nil {
		return fmt.Errorf("error getting brand id of tag, %w", err)
	}
	originId, err := p.idGetter.GetId(idGetter.OriginCollection, tag.Origin)
	if err != nil {
		return fmt.Errorf("error getting origin id of tag, %w", err)
	}

	ag, err := p.aggregates.Get(tag.Time)
	if err != nil && !errors.Is(err, db.KeyNotFoundError) {
		return fmt.Errorf("error getting aggregates, %w", err)
	}

	var tA *db.TypeAggregates
	switch tag.Action {
	case types.Buy:
		tA = &ag.Result.Buys
	case types.View:
		tA = &ag.Result.Views
	default:
		return fmt.Errorf("unknown action, %d", tag.Action)
	}

	found := false
	for i, a := range tA.Sum {
		if a.BrandId == uint8(brandId) && a.Origin == uint8(originId) && a.CategoryId == uint16(categoryId) {
			found = true
			tA.Sum[i].Data += tag.ProductInfo.Price
			tA.Count[i].Data++
		}
	}

	if !found {
		tA.Sum = append(tA.Sum, db.ActionAggregates{
			CategoryId: uint16(categoryId),
			BrandId:    uint8(brandId),
			Origin:     uint8(originId),
			Data:       tag.ProductInfo.Price,
		})

		tA.Count = append(tA.Count, db.ActionAggregates{
			CategoryId: uint16(categoryId),
			BrandId:    uint8(brandId),
			Origin:     uint8(originId),
			Data:       1,
		})
	}

	if err := p.aggregates.Update(tag.Time, ag.Result, ag.Generation); err != nil {
		return fmt.Errorf("error updating aggregates, %w", err)
	}
	return nil
}
