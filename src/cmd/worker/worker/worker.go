package worker

import (
	"context"
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/messaging"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"go.uber.org/zap"
	"runtime"
)

type Worker interface {
	Run(ctx context.Context) error
}

type Dependencies struct {
	Consumer *messaging.Consumer
	DB       db.Client
	Logger   *zap.Logger
}

type worker struct {
	consumer *messaging.Consumer
	db       db.Client
	logger   *zap.Logger
}

const chanSize = 1024

var numProcessors = runtime.NumCPU()

func (w worker) Run(ctx context.Context) error {
	tagsChan := make(chan types.UserTag, chanSize)
	defer close(tagsChan)

	for i := 0; i < numProcessors; i++ {
		p := processor{logger: w.logger}
		go p.Run(tagsChan)
	}

	if err := w.consumer.Consume(ctx, tagsChan); err != nil {
		return fmt.Errorf("error consuming messages, %w", err)
	}
	return nil
}

func (p processor) Run(tagsChan <-chan types.UserTag) {
	for tag := range tagsChan {
		if err := p.processTag(tag); err != nil {
			p.logger.Error("error processing user tag", zap.Error(err))
		}
	}
}

func (p processor) processTag(tag types.UserTag) error {
	// TODO implement me
	panic("not implemented")
}

type processor struct {
	logger *zap.Logger
}

func New(deps Dependencies) Worker {
	return worker{
		consumer: deps.Consumer,
		db:       deps.DB,
		logger:   deps.Logger,
	}
}
