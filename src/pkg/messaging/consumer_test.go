package messaging

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
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

	var tagsToSend []types.UserTag
	for i := 0; i < 10; i++ {
		tagsToSend = append(tagsToSend, types.UserTag{Cookie: fmt.Sprintf("cookie-%d", i)})
	}
	recTags := make(chan types.UserTag)

	g := new(errgroup.Group)

	g.Go(func() error {
		s.logger.Debug("sending tags")
		for _, tag := range tagsToSend {
			err := producer.Send(tag)
			s.Assert().NoErrorf(err, "failed to send tag %v", tag.Cookie)
		}
		s.logger.Debug("finished sending tags")
		return nil
	})
	g.Go(func() error {
		s.logger.Debug("consuming tags")

		err := consumer.Consume(ctx, recTags)
		s.Assert().NoErrorf(err, "failed to consume tags")
		s.logger.Debug("finished consuming tags")
		return nil
	})

	var tags []types.UserTag
	for i := 0; i < len(tagsToSend); i++ {
		select {
		case tag := <-recTags:
			tags = append(tags, tag)
		case <-time.After(timeout):
			s.FailNow("timed out waiting for tags")
		}
	}
	cancel()
	_ = g.Wait()

	s.Require().Equalf(tagsToSend, tags, "received tags do not match sent tags")
}

// FIXME: this test fails as our implementation uses only single topic partition, so we can't consume messages in parallel
//func (s *MessagingSuite) TestConsumer_Consume_multiple_consumers() {
//	const consumersNum = 10
//	const tagsToSendNum = 100000
//
//	producer, err := NewProducer(s.logger, []string{hostPort})
//	s.Require().NoErrorf(err, "failed to create producer")
//
//	var consumers []*Consumer
//	for i := 0; i < consumersNum; i++ {
//		consumer, err := NewConsumer(s.logger, []string{hostPort})
//		s.Require().NoErrorf(err, "failed to create consumer")
//		consumers = append(consumers, consumer)
//	}
//
//	ctx, cancel := context.WithCancel(context.Background())
//
//	var tagsToSend []types.UserTag
//	for i := 0; i < tagsToSendNum; i++ {
//		tagsToSend = append(tagsToSend, types.UserTag{Cookie: fmt.Sprintf("cookie-%d", i)})
//	}
//
//	recTags := make(chan types.UserTag)
//
//	var consumersUsed sync.Map
//
//	var wg sync.WaitGroup
//	wg.Add(1 + consumersNum)
//
//	go func() {
//		// Set up producer
//		defer wg.Done()
//		s.logger.Debug("sending tags")
//		for _, tag := range tagsToSend {
//			err := producer.Send(tag)
//			s.Assert().NoErrorf(err, "failed to send tag %v", tag.Cookie)
//		}
//		s.logger.Debug("finished sending tags")
//	}()
//
//	for id, consumer := range consumers {
//		go func(id int, consumer *Consumer) {
//			defer wg.Done()
//
//			// Using a private channel for each consumer to
//			// register which consumer received which tag.
//			privateRecTags := make(chan types.UserTag)
//			defer close(privateRecTags)
//
//			go func() {
//				for tag := range privateRecTags {
//					consumersUsed.Store(id, true)
//					recTags <- tag
//				}
//			}()
//
//			s.logger.Debug("consuming tags")
//
//			err := consumer.Consume(ctx, privateRecTags)
//			s.Assert().NoErrorf(err, "failed to consume tags")
//			s.logger.Debug("finished consuming tags")
//		}(id, consumer)
//	}
//
//	var tags []types.UserTag
//	for i := 0; i < len(tagsToSend); i++ {
//		select {
//		case tag := <-recTags:
//			tags = append(tags, tag)
//		case <-time.After(timeout):
//			s.FailNow("timed out waiting for tags")
//		}
//	}
//	cancel()
//	wg.Wait()
//	s.Require().Equalf(tagsToSend, tags, "received tags do not match sent tags")
//
//	var consumersUsedList []int
//	consumersUsed.Range(func(key, value interface{}) bool {
//		consumersUsedList = append(consumersUsedList, key.(int))
//		return true
//	})
//
//	s.Require().Greaterf(len(consumersUsedList), 1, "only one consumer was used %v", consumersUsedList)
//}
