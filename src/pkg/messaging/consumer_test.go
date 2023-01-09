package messaging

import (
	"context"
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"go.uber.org/zap"
	"sync"
	"time"
)

var timeout = time.Second * 5

func (s *MessagingSuite) TestNewConsumer() {
	_, err := NewConsumer(s.logger, []string{hostPort})
	s.Require().NoErrorf(err, "failed to create consumer")
}

func (s *MessagingSuite) TestConsumer_Consume() {
	s.T().Skip() // TODO: fix this test
	
	producer, err := NewProducer(s.logger, []string{hostPort})
	s.Require().NoErrorf(err, "failed to create producer")

	consumer, err := NewConsumer(s.logger, []string{hostPort})
	s.Require().NoErrorf(err, "failed to create consumer")

	tags := make(chan types.UserTag)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	var sendTags []types.UserTag
	for i := 0; i < 10; i++ {
		sendTags = append(sendTags, types.UserTag{Cookie: fmt.Sprintf("cookie-%d", i)})
	}

	recTags := make(chan types.UserTag)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s.logger.Debug("sending tags")
		for tag := range tags {
			err := producer.Send(tag)
			s.logger.Debug("sent tag", zap.Any("tag", tag))
			s.Assert().NoErrorf(err, "failed to send tag %v", tag.Cookie)
		}
	}()
	go func() {
		defer wg.Done()
		s.logger.Debug("consuming tags")

		err := consumer.Consume(ctx, recTags)
		s.Assert().NoErrorf(err, "failed to consume tags")
	}()

	wg.Wait()
	cancel()

	for _, tag := range sendTags {
		select {
		case recTag := <-recTags:
			s.Assert().Equalf(tag, recTag, "received tag does not match sent tag")
		case <-ctx.Done():
			s.Fail("context timed out")
		}
	}

}
