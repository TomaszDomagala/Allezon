package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

const (
	UserTagsTopic         = "user-tags"
	UserTagsConsumerGroup = "user-tags-consumer-group"
)

type UserTagsConsumer interface {
	Receive() (<-chan types.UserTag, error)
}

type Consumer struct {
	logger *zap.Logger
	client sarama.ConsumerGroup
}

func NewConsumer(logger *zap.Logger, addresses []string) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumer, err := sarama.NewConsumerGroup(addresses, UserTagsConsumerGroup, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return &Consumer{logger: logger, client: consumer}, nil
}

// Consume consumes messages and pushes them to the tags channel. It blocks until the context is cancelled or an error occurs.
// Should be run in a goroutine.
func (c *Consumer) Consume(ctx context.Context, tags chan<- types.UserTag) error {
	// Following code is heavily inspired by sarama example https://github.com/Shopify/sarama/blob/main/examples/consumergroup/main.go.

	handler := consumerGroupHandler{
		logger: c.logger,
		tags:   tags,
	}
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := c.client.Consume(ctx, []string{UserTagsTopic}, &handler); err != nil {
				c.logger.Error("failed to consume messages", zap.Error(err))
				return err
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				c.logger.Debug("consumer context cancelled", zap.Error(ctx.Err()))
				return nil
			}
		}
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to consume messages: %w", err)
	}
	return nil
}

type consumerGroupHandler struct {
	logger *zap.Logger
	tags   chan<- types.UserTag
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
func (c *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready.
	c.logger.Debug("consumer group handler ready")
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg := <-claim.Messages():
			c.logger.Debug("received message", zap.ByteString("key", msg.Key), zap.ByteString("value", msg.Value), zap.String("topic", msg.Topic), zap.Int32("partition", msg.Partition), zap.Int64("offset", msg.Offset))
			var tag types.UserTag
			if err := json.Unmarshal(msg.Value, &tag); err != nil {
				c.logger.Error("failed to unmarshal message", zap.Error(err))
				continue
			}
			c.tags <- tag
			session.MarkMessage(msg, "")
		case <-session.Context().Done():
			return nil
		}
	}
}
