package worker

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

// aggregatesBackoff is a backoff strategy used to update aggregates.
// aggregates only fail if db is down hence larger backoff.
var aggregatesBackoff = backoff.ExponentialBackOff{
	InitialInterval:     1 * time.Second,
	RandomizationFactor: backoff.DefaultRandomizationFactor,
	Multiplier:          backoff.DefaultMultiplier,
	MaxInterval:         300 * time.Second,
	MaxElapsedTime:      30 * time.Second,
	Stop:                backoff.Stop,
	Clock:               backoff.SystemClock,
}

func runAggregatesProcessor(tagsChan <-chan types.UserTag, idsClient idGetter.Client, aggregates db.AggregatesClient, logger *zap.Logger) {
	for tag := range tagsChan {
		logger.Debug("processing tag", zap.Any("tag", tag))
		if err := updateAggregatesBackoff(tag, idsClient, aggregates, aggregatesBackoff, logger); err != nil {
			logger.Error("error updating aggregates", zap.Error(err))
		}
		logger.Debug("processed tag", zap.Any("tag", tag))
	}
}

// updateAggregatesBackoff updates aggregates with the given tag and retries on error according to the given backoff strategy.
func updateAggregatesBackoff(tag types.UserTag, idsClient idGetter.Client, aggregates db.AggregatesClient, bo backoff.ExponentialBackOff, logger *zap.Logger) error {
	err := backoff.Retry(func() error {
		if err := updateAggregates(tag, idsClient, aggregates); err != nil {
			logger.Warn("error processing tag", zap.Any("tag", tag), zap.Error(err))
			return err
		}
		return nil
	}, &bo)
	if err != nil {
		return fmt.Errorf("error backoff updating aggregates, %w", err)
	}
	return nil
}

func getId(idsClient idGetter.Client, collection string, element string) (uint16, error) {
	id, err := idGetter.GetU16ID(idsClient, collection, element)
	if err != nil {
		return 0, fmt.Errorf("error getting %s id of tag, %w", collection, err)
	}
	return id, nil
}

// updateAggregates updates aggregates with the given tag.
func updateAggregates(tag types.UserTag, idsClient idGetter.Client, aggregates db.AggregatesClient) (err error) {
	var key db.AggregateKey
	key.CategoryId, err = getId(idsClient, idGetter.CategoryCollection, tag.ProductInfo.CategoryId)
	if err != nil {
		return err
	}
	key.BrandId, err = getId(idsClient, idGetter.BrandCollection, tag.ProductInfo.BrandId)
	if err != nil {
		return err
	}
	key.Origin, err = getId(idsClient, idGetter.OriginCollection, tag.Origin)
	if err != nil {
		return err
	}

	if err := aggregates.Add(key, tag); err != nil {
		return fmt.Errorf("error updating aggregates, %w", err)
	}
	return nil
}
