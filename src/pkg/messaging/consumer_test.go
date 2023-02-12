package messaging

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

var timeout = time.Second * 20

func (s *MessagingSuite) newConsumer() *Consumer {
	c, err := NewConsumer(s.logger, s.kafkaAddresses())
	s.Require().NoErrorf(err, "failed to create consumer")
	return c
}

func (s *MessagingSuite) TestNewConsumer() {
	c := s.newConsumer()
	runtime.KeepAlive(c)
}

func (s *MessagingSuite) TestConsumer_Consume() {
	producer := s.newProducer()

	consumer := s.newConsumer()

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

	// Order is not guaranteed, so we need to sort the slices.
	sort.Slice(tagsToSend, func(i, j int) bool {
		return tagsToSend[i].Cookie < tagsToSend[j].Cookie
	})
	sort.Slice(tagsRec, func(i, j int) bool {
		return tagsRec[i].Cookie < tagsRec[j].Cookie
	})

	s.Assert().Equalf(tagsToSend, tagsRec, "received tags do not match sent tags")
}

func (s *MessagingSuite) TestConsumer_Consume_multiple_consumers() {
	const consumersNum = testTopicPartitionsNumber
	const tagsToSendNum = 1000

	producer := s.newProducer()

	var consumers []*Consumer
	for i := 0; i < consumersNum; i++ {
		consumers = append(consumers, s.newConsumer())
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

	// Order is not guaranteed, so we need to sort the slices.
	sort.Slice(tagsToSend, func(i, j int) bool {
		return tagsToSend[i].Cookie < tagsToSend[j].Cookie
	})
	sort.Slice(tagsRec, func(i, j int) bool {
		return tagsRec[i].Cookie < tagsRec[j].Cookie
	})
	s.Assert().Equalf(tagsToSend, tagsRec, "received tags do not match sent tags")

	var consumersUsedList []int
	consumersUsed.Range(func(key, value interface{}) bool {
		consumersUsedList = append(consumersUsedList, key.(int))
		return true
	})

	s.Assert().Equalf(len(consumersUsedList), consumersNum, "not all consumers were used")
}
