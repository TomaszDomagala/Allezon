package worker

import (
	"context"
	"fmt"
	"runtime"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"go.uber.org/zap"
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
	userTagsChan := make(chan types.UserTag, chanSize)
	aggregatesChan := make(chan types.UserTag, chanSize)
	defer close(tagsChan)
	defer close(userTagsChan)
	defer close(aggregatesChan)

	for i := 0; i < numProcessors; i++ {
		go runAggregatesProcessor(aggregatesChan, w.idGetter, w.db.Aggregates(), w.logger)
		go runUpdateUserProfileProcessor(userTagsChan, w.db.UserProfiles(), w.logger)
	}

	go func() {
		for tag := range tagsChan {
			userTagsChan <- tag
			aggregatesChan <- tag
		}
	}()

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
