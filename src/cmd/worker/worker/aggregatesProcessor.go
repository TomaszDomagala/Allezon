package worker

import (
	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"go.uber.org/zap"
)

type aggregatesProcessor struct {
	db       db.Client
	logger   *zap.Logger
	idGetter idGetter.Client
}

func (p aggregatesProcessor) run(tagsChan <-chan types.UserTag) {
	for tag := range tagsChan {
		if err := p.processTag(tag); err != nil {
			p.logger.Error("error processing user tag", zap.Error(err))
		}
	}
}

func (p aggregatesProcessor) processTag(tag types.UserTag) error {
	// TODO implement me
	panic("not implemented")
}
