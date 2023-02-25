package worker

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
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

func runAggregatesProcessor(messages <-chan messaging.UserTagMessage, idsClient idGetter.Client, aggregates db.AggregatesClient, logger *zap.Logger) {
	for msg := range messages {
		logger.Debug("processing tag", zap.Any("tag", msg.Data()))
		if err := updateAggregatesBackoff(msg, idsClient, aggregates, aggregatesBackoff, logger); err != nil {
			logger.Error("error updating aggregates", zap.Error(err))
		}
		logger.Debug("processed tag", zap.Any("tag", msg.Data()))
	}
}

// updateAggregatesBackoff updates aggregates with the given tag and retries on error according to the given backoff strategy.
func updateAggregatesBackoff(msg messaging.UserTagMessage, idsClient idGetter.Client, aggregates db.AggregatesClient, bo backoff.ExponentialBackOff, logger *zap.Logger) error {
	err := backoff.Retry(func() error {
		if err := updateAggregates(msg, idsClient, aggregates); err != nil {
			logger.Warn("error processing tag", zap.Any("tag", msg.Data()), zap.Error(err))
			return err
		}
		return nil
	}, &bo)
	if err != nil {
		return fmt.Errorf("error backoff updating aggregates, %w", err)
	}
	return nil
}

// updateAggregates updates aggregates with the given tag.
func updateAggregates(message messaging.UserTagMessage, idsClient idGetter.Client, aggregates db.AggregatesClient) (err error) {
	var key db.AggregateKey
	tag := message.Data()
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
	message.Mark()
	return nil
}

func getId(idsClient idGetter.Client, collection string, element string) (uint16, error) {
	id, err := idGetter.GetU16ID(idsClient, collection, element, true)
	if err != nil {
		return 0, fmt.Errorf("error getting %s id of tag, %w", collection, err)
	}
	return id, nil
}
