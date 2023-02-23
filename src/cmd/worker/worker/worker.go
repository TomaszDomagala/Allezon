package worker

import (
	"context"
	"fmt"
	"runtime"

	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
)

type Worker interface {
	Run(ctx context.Context) error
}

type Dependencies struct {
	Consumer     *messaging.Consumer
	AggregatesDB db.Client
	Logger       *zap.Logger
	IDGetter     idGetter.Client
}

type worker struct {
	consumer     *messaging.Consumer
	aggregatesDB db.Client
	logger       *zap.Logger
	idGetter     idGetter.Client
}

const chanSize = 1024

var numProcessors = runtime.NumCPU()

func (w worker) Run(ctx context.Context) error {
	messages := make(chan messaging.UserTagMessage, chanSize)
	defer close(messages)

	for i := 0; i < numProcessors; i++ {
		go runAggregatesProcessor(messages, w.idGetter, w.aggregatesDB.Aggregates(), w.logger)
	}

	if err := w.consumer.Consume(ctx, messages); err != nil {
		return fmt.Errorf("error consuming messages, %w", err)
	}
	return nil
}

func New(deps Dependencies) Worker {
	return worker{
		consumer:     deps.Consumer,
		aggregatesDB: deps.AggregatesDB,
		logger:       deps.Logger,
		idGetter:     deps.IDGetter,
	}
}
