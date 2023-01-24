package worker

import (
	"errors"
	"fmt"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

type baseProcessor struct {
	logger         *zap.Logger
	config         baseProcessorCfg
	processTagOnce func(tag types.UserTag) error
}

type baseProcessorCfg struct {
	backoff backoff.ExponentialBackOff
}

func newBaseProcessor(processTagOnce func(tag types.UserTag) error, logger *zap.Logger) baseProcessor {
	return baseProcessor{
		logger:         logger,
		processTagOnce: processTagOnce,
		config: baseProcessorCfg{
			backoff: backoff.ExponentialBackOff{
				InitialInterval:     10 * time.Millisecond,
				RandomizationFactor: backoff.DefaultRandomizationFactor,
				Multiplier:          backoff.DefaultMultiplier,
				MaxInterval:         500 * time.Second,
				MaxElapsedTime:      10 * time.Second,
				Stop:                backoff.Stop,
				Clock:               backoff.SystemClock,
			}},
	}
}

func (p baseProcessor) run(tagsChan <-chan types.UserTag) {
	for tag := range tagsChan {
		if err := p.processTag(tag); err != nil {
			p.logger.Error("error processing user tag", zap.Error(err))
		}
	}
}

func (p baseProcessor) processTag(tag types.UserTag) error {
	bo := p.config.backoff
	bo.Reset()

	err := backoff.Retry(func() error {
		err := p.processTagOnce(tag)
		if err != nil {
			p.logger.Error("error processing user tag", zap.Error(err))
			if !errors.Is(err, db.GenerationMismatch) {
				return fmt.Errorf("error while processing user tag with cookie %s and timestamp %s, %w", tag.Cookie, tag.Time, err)
			}
		}
		return nil
	}, &bo)
	if err != nil {
		return fmt.Errorf("error while processing with backoff user tag with cookie %s and timestamp %s, %w", tag.Cookie, tag.Time, err)
	}
	return nil
}
