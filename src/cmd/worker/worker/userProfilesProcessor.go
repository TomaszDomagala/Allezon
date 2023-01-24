package worker

import (
	"github.com/TomaszDomagala/Allezon/src/pkg/db"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"go.uber.org/zap"
)

type userProfilesProcessor struct {
	db     db.Client
	logger *zap.Logger
}

func (p userProfilesProcessor) run(tagsChan <-chan types.UserTag) {
	for tag := range tagsChan {
		if err := p.processTag(tag); err != nil {
			p.logger.Error("error processing user tag", zap.Error(err))
		}
	}
}

func (p userProfilesProcessor) processTag(tag types.UserTag) error {
	// TODO implement me
	panic("not implemented")
}
