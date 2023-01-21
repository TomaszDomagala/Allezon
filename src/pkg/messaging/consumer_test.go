package messaging

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
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

	var tagsToSend []types.UserTag
	for i := 0; i < 10; i++ {
		tagsToSend = append(tagsToSend, types.UserTag{Cookie: fmt.Sprintf("cookie-%d", i)})
	}
	recTags := make(chan types.UserTag)

	g, ctx := errgroup.WithContext(ctx)

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

	var tagsRec []types.UserTag
	for i := 0; i < len(tagsToSend); i++ {
		select {
		case tag := <-recTags:
			tagsRec = append(tagsRec, tag)
		case <-time.After(timeout):
			s.FailNow("timed out waiting for tags")
		}
	}
	cancel()
	_ = g.Wait()

	var tagsToSendSet = make(map[string]struct{})
	var tagsRecSet = make(map[string]struct{})

	// Order is not guaranteed between partitions, so we need to check if the sets are equal.
	for _, tag := range tagsToSend {
		tagsToSendSet[tag.Cookie] = struct{}{}
	}
	for _, tag := range tagsRec {
		tagsRecSet[tag.Cookie] = struct{}{}
	}
	s.Assert().Equalf(tagsToSendSet, tagsRecSet, "received tags do not match sent tags")
}

func (s *MessagingSuite) TestConsumer_Consume_multiple_consumers() {
	const consumersNum = testTopicPartitionsNumber
	const tagsToSendNum = 1000

	producer, err := NewProducer(s.logger, []string{hostPort})
	s.Require().NoErrorf(err, "failed to create producer")

	var consumers []*Consumer
	for i := 0; i < consumersNum; i++ {
		consumer, err := NewConsumer(s.logger, []string{hostPort})
		s.Require().NoErrorf(err, "failed to create consumer")
		consumers = append(consumers, consumer)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var tagsToSend []types.UserTag
	for i := 0; i < tagsToSendNum; i++ {
		tagsToSend = append(tagsToSend, types.UserTag{Cookie: fmt.Sprintf("cookie-%d", i)})
	}

	recTags := make(chan types.UserTag)

	var consumersUsed sync.Map

	var wg sync.WaitGroup
	wg.Add(1 + consumersNum)

	go func() {
		// Set up producer
		defer wg.Done()
		s.logger.Debug("sending tags")
		for _, tag := range tagsToSend {
			err := producer.Send(tag)
			s.Assert().NoErrorf(err, "failed to send tag %v", tag.Cookie)
		}
		s.logger.Debug("finished sending tags")
	}()

	for id, consumer := range consumers {
		go func(id int, consumer *Consumer) {
			defer wg.Done()

			// Using a private channel for each consumer to
			// register which consumer received which tag.
			privateRecTags := make(chan types.UserTag)
			defer close(privateRecTags)

			go func() {
				for tag := range privateRecTags {
					consumersUsed.Store(id, true)
					recTags <- tag
				}
			}()

			s.logger.Debug("consuming tags")

			err := consumer.Consume(ctx, privateRecTags)
			s.Assert().NoErrorf(err, "failed to consume tags")
			s.logger.Debug("finished consuming tags")
		}(id, consumer)
	}

	var tagsRec []types.UserTag
	for i := 0; i < len(tagsToSend); i++ {
		select {
		case tag := <-recTags:
			tagsRec = append(tagsRec, tag)
		case <-time.After(timeout):
			s.FailNow("timed out waiting for tags")
		}
	}
	cancel()
	wg.Wait()

	var tagsToSendSet = make(map[string]struct{})
	var tagsRecSet = make(map[string]struct{})

	// Order is not guaranteed between partitions, so we need to check if the sets are equal.
	for _, tag := range tagsToSend {
		tagsToSendSet[tag.Cookie] = struct{}{}
	}
	for _, tag := range tagsRec {
		tagsRecSet[tag.Cookie] = struct{}{}
	}
	s.Assert().Equalf(tagsToSendSet, tagsRecSet, "received tags do not match sent tags")

	var consumersUsedList []int
	consumersUsed.Range(func(key, value interface{}) bool {
		consumersUsedList = append(consumersUsedList, key.(int))
		return true
	})

	s.Assert().Equalf(len(consumersUsedList), 4, "not all consumers were used")
}
