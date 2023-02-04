package worker

import (
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

// aggregatesBackoff is a backoff strategy used to update aggregates.
var aggregatesBackoff = backoff.ExponentialBackOff{
	InitialInterval:     10 * time.Millisecond,
	RandomizationFactor: backoff.DefaultRandomizationFactor,
	Multiplier:          backoff.DefaultMultiplier,
	MaxInterval:         500 * time.Second,
	MaxElapsedTime:      10 * time.Second,
	Stop:                backoff.Stop,
	Clock:               backoff.SystemClock,
}

func runAggregatesProcessor(tagsChan <-chan types.UserTag, idsClient idGetter.Client, aggregates db.AggregatesClient, logger *zap.Logger) {
	for tag := range tagsChan {
		if err := updateAggregatesBackoff(tag, idsClient, aggregates, aggregatesBackoff); err != nil {
			logger.Error("error updating aggregates", zap.Error(err))
		}
	}
}

// updateAggregatesBackoff updates aggregates with the given tag and retries on error according to the given backoff strategy.
func updateAggregatesBackoff(tag types.UserTag, idsClient idGetter.Client, aggregates db.AggregatesClient, bo backoff.ExponentialBackOff) error {
	err := backoff.Retry(func() error {
		return updateAggregates(tag, idsClient, aggregates)
	}, &bo)
	if err != nil {
		return fmt.Errorf("error backoff updating aggregates, %w", err)
	}
	return nil
}

// updateAggregates updates aggregates with the given tag.
func updateAggregates(tag types.UserTag, idsClient idGetter.Client, aggregates db.AggregatesClient) error {
	categoryID, err := idsClient.GetID(idGetter.CategoryCollection, tag.ProductInfo.CategoryId)
	if err != nil {
		return fmt.Errorf("error getting category id of tag, %w", err)
	}
	brandID, err := idsClient.GetID(idGetter.BrandCollection, tag.ProductInfo.BrandId)
	if err != nil {
		return fmt.Errorf("error getting brand id of tag, %w", err)
	}
	originID, err := idsClient.GetID(idGetter.OriginCollection, tag.Origin)
	if err != nil {
		return fmt.Errorf("error getting origin id of tag, %w", err)
	}

	ag, err := aggregates.Get(tag.Time)
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
		if a.BrandId == uint8(brandID) && a.Origin == uint8(originID) && a.CategoryId == uint16(categoryID) {
			found = true
			tA.Sum[i].Data += tag.ProductInfo.Price
			tA.Count[i].Data++
		}
	}

	if !found {
		tA.Sum = append(tA.Sum, db.ActionAggregates{
			CategoryId: uint16(categoryID),
			BrandId:    uint8(brandID),
			Origin:     uint8(originID),
			Data:       tag.ProductInfo.Price,
		})

		tA.Count = append(tA.Count, db.ActionAggregates{
			CategoryId: uint16(categoryID),
			BrandId:    uint8(brandID),
			Origin:     uint8(originID),
			Data:       1,
		})
	}

	if err := aggregates.Update(tag.Time, ag.Result, ag.Generation); err != nil {
		return fmt.Errorf("error updating aggregates, %w", err)
	}
	return nil
}
