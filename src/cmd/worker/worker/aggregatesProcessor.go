package worker

import (
	"fmt"
	"math"
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
	if categoryID > math.MaxUint16 {

	}

	brandID, err := idsClient.GetID(idGetter.BrandCollection, tag.ProductInfo.BrandId)
	if err != nil {
		return fmt.Errorf("error getting brand id of tag, %w", err)
	}
	originID, err := idsClient.GetID(idGetter.OriginCollection, tag.Origin)
	if err != nil {
		return fmt.Errorf("error getting origin id of tag, %w", err)
	}

	key := db.AggregateKey{
		CategoryId: uint16(categoryID),
		BrandId:    uint16(brandID),
		Origin:     uint16(originID),
	}

	if err := aggregates.Add(key, tag); err != nil {
		return fmt.Errorf("error updating aggregates, %w", err)
	}
	return nil
}
