package worker

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/cmd/worker/worker/userProfilesGc"
	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

type Worker interface {
	Run(ctx context.Context) error
}

type Dependencies struct {
	Consumer *messaging.Consumer
	DB       db.Client
	Logger   *zap.Logger
	IDGetter idGetter.Client
}

type worker struct {
	consumer *messaging.Consumer
	db       db.Client
	logger   *zap.Logger
	idGetter idGetter.Client
}

const chanSize = 1024

var numProcessors = runtime.NumCPU()

func (w worker) Run(ctx context.Context) error {
	tagsChan := make(chan types.UserTag, chanSize)
	defer close(tagsChan)
	aggregatesChan := make(chan types.UserTag, chanSize)
	defer close(aggregatesChan)
	gcEventsChan := make(chan userProfilesGc.Event, chanSize)
	defer close(gcEventsChan)

	go func() {
		bo := backoff.NewExponentialBackOff()
		bo.InitialInterval = 500 * time.Millisecond
		bo.Multiplier = 3
		bo.MaxElapsedTime = time.Minute
		bo.MaxInterval = 10 * time.Second

		deps := userProfilesGc.Dependencies{
			Logger:              w.logger,
			EventChan:           gcEventsChan,
			GcKeyInterval:       time.Minute,
			Backoff:             bo,
			UserProfilesCleaner: w.db.UserProfiles().RemoveOverLimit,
			Limit:               200,
		}
		gc := userProfilesGc.New(deps)
		gc.Run() // Lifetime bound to gcEventsChan.
	}()

	go func() {
		for tag := range tagsChan {
			aggregatesChan <- tag
			gcEventsChan <- userProfilesGc.Event{
				Cookie: tag.Cookie,
				Action: tag.Action,
			}
		}
	}()

	for i := 0; i < numProcessors; i++ {
		go runAggregatesProcessor(aggregatesChan, w.idGetter, w.db.Aggregates(), w.logger)
	}

	if err := w.consumer.Consume(ctx, tagsChan); err != nil {
		return fmt.Errorf("error consuming messages, %w", err)
	}
	return nil
}

func New(deps Dependencies) Worker {
	return worker{
		consumer: deps.Consumer,
		db:       deps.DB,
		logger:   deps.Logger,
		idGetter: deps.IDGetter,
	}
}
