package messaging

import (
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

type UserTagsProducer interface {
	Send(tag types.UserTag) error
}

type Producer struct {
	logger   *zap.Logger
	producer sarama.SyncProducer
}

func NewProducer(logger *zap.Logger, addresses []string) (*Producer, error) {
	producer, err := sarama.NewSyncProducer(addresses, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &Producer{logger: logger, producer: producer}, nil
}

func (p *Producer) Send(tag types.UserTag) error {
	start := time.Now()

	tagBytes, err := types.MarshalUserTag(&tag)
	if err != nil {
		return fmt.Errorf("failed to marshal user tag: %w", err)
	}
	partition, offset, err := p.producer.SendMessage(&sarama.ProducerMessage{
		Topic:     UserTagsTopic,
		Value:     sarama.ByteEncoder(tagBytes),
		Partition: 0,
	})

	logOpts := []zap.Field{
		zap.String("topic", UserTagsTopic),
		zap.ByteString("value", tagBytes),
		zap.Duration("duration", time.Since(start)),
	}
	if err != nil {
		p.logger.Error("failed to send kafka message", append(logOpts, zap.Error(err))...)
		return fmt.Errorf("failed to send kafka message: %w", err)
	}
	p.logger.Debug("kafka message sent", append(logOpts, zap.Int32("partition", partition), zap.Int64("offset", offset))...)
	return nil
}
