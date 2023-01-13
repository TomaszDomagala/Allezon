package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

var timeout = time.Second * 20

func (s *MessagingSuite) TestNewConsumer() {
	_, err := NewConsumer(s.logger, []string{hostPort})
	s.Require().NoErrorf(err, "failed to create consumer")
}

func (s *MessagingSuite) TestConsumer_Consume() {
	producer, err := NewProducer(s.logger, []string{hostPort})
	s.Require().NoErrorf(err, "failed to create producer")

	consumer, err := NewConsumer(s.logger, []string{hostPort})
	s.Require().NoErrorf(err, "failed to create consumer")

	ctx, cancel := context.WithCancel(context.Background())

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
		for _, tag := range sendTags {
			err := producer.Send(tag)
			s.Assert().NoErrorf(err, "failed to send tag %v", tag.Cookie)
		}
		s.logger.Debug("finished sending tags")
	}()
	go func() {
		defer wg.Done()
		s.logger.Debug("consuming tags")

		err := consumer.Consume(ctx, recTags)
		s.Assert().NoErrorf(err, "failed to consume tags")
		s.logger.Debug("finished consuming tags")
	}()

	var tags []types.UserTag
	for i := 0; i < len(sendTags); i++ {
		select {
		case tag := <-recTags:
			tags = append(tags, tag)
		case <-time.After(timeout):
			s.FailNow("timed out waiting for tags")
		}
	}
	cancel()

	wg.Wait()

	s.Require().Equalf(sendTags, tags, "received tags do not match sent tags")

}
