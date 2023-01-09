package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

const UserTagConsumerGroup = "user-tag-consumer-group"

type UserTagConsumer interface {
	Receive() (<-chan types.UserTag, error)
}

type Consumer struct {
	logger *zap.Logger
	client sarama.ConsumerGroup
}

func NewConsumer(logger *zap.Logger, addresses []string) (*Consumer, error) {
	consumer, err := sarama.NewConsumerGroup(addresses, UserTagConsumerGroup, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return &Consumer{logger: logger, client: consumer}, nil
}

// Consume consumes messages and pushes them to the tags channel. It blocks until the context is cancelled or an error occurs.
// Should be run in a goroutine.
func (c *Consumer) Consume(ctx context.Context, tags chan<- types.UserTag) error {
	// Following conde is heavily inspired by sarama example https://github.com/Shopify/sarama/blob/main/examples/consumergroup/main.go.

	handler := consumerGroupHandler{
		logger: c.logger,
		ready:  make(chan bool),
		tags:   tags,
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	var errConsume error
	go func() {
		defer wg.Done()

		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if errConsume = c.client.Consume(ctx, []string{types.UserTagsTopic}, &handler); errConsume != nil {
				c.logger.Error("failed to consume messages", zap.Error(errConsume))
				break
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				c.logger.Debug("consumer context cancelled", zap.Error(ctx.Err()))
				return
			}
			handler.ready = make(chan bool)
		}
	}()
	<-handler.ready // Wait till the consumer has been set up.
	c.logger.Debug("sarama consumer ready")

	wg.Wait()
	if errConsume != nil {
		return fmt.Errorf("failed to consume messages: %w", errConsume)
	}
	return nil
}

type consumerGroupHandler struct {
	logger *zap.Logger
	ready  chan bool
	tags   chan<- types.UserTag
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
func (c *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready.
	close(c.ready)
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
