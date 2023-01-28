package messaging

import (
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

type null struct {
	logger *zap.Logger
}

// NewNullProducer returns a producer that does nothing but log invoked methods.
func NewNullProducer(logger *zap.Logger) UserTagsProducer {
	return &null{logger: logger}
}

func (n *null) Send(tag types.UserTag) error {
	n.logger.Debug("null producer invoked", zap.String("method", "Send"), zap.Any("tag", tag))
	return nil
}
