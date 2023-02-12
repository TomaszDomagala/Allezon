package messaging

import (
	"runtime"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/container/containerutils"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
)

const testTopicPartitionsNumber = 4

func (s *MessagingSuite) createTestTopic() {
	admin, err := sarama.NewClusterAdmin(s.kafkaAddresses(), nil)
	s.Require().NoErrorf(err, "failed to create client")
	defer func() {
		err := admin.Close()
		if err != nil {
			s.env.Logger.Error("failed to close admin", zap.Error(err))
		}
	}()
	err = admin.CreateTopic(UserTagsTopic, &sarama.TopicDetail{
		NumPartitions:     testTopicPartitionsNumber,
		ReplicationFactor: 1,
	}, false)
	s.Require().NoErrorf(err, "failed to create topic")
}

// MessagingSuite is a suite for messaging integration tests.
type MessagingSuite struct {
	suite.Suite
	logger *zap.Logger

	// env is created for each test case.
	env *container.Environment
}

// TestProducerSuite is an entry point for running all tests in this package.
func TestProducerSuite(t *testing.T) {
	suite.Run(t, new(MessagingSuite))
}

func (s *MessagingSuite) SetupSuite() {
	var err error

	s.logger, err = zap.NewDevelopment()
	s.Require().NoErrorf(err, "could not create logger")
}

func (s *MessagingSuite) SetupTest() {
	s.env = container.NewEnvironment(s.T().Name(), s.logger, []*container.Service{containerutils.RedpandaService}, nil)
	err := s.env.Run()
	s.Require().NoErrorf(err, "could not run environment")
	s.createTestTopic()
}

func (s *MessagingSuite) TearDownTest() {
	err := s.env.Close()
	s.Require().NoErrorf(err, "could not close environment")
	s.env = nil
}

func (s *MessagingSuite) kafkaAddresses() []string {
	return []string{containerutils.RedpandaAddr}
}

func (s *MessagingSuite) newProducer() *Producer {
	p, err := NewProducer(s.logger, s.kafkaAddresses())
	s.Require().NoErrorf(err, "failed to create producer")
	return p
}

func (s *MessagingSuite) TestNewProducer() {
	p := s.newProducer()
	runtime.KeepAlive(p)
}

func (s *MessagingSuite) TestProducer_Send() {
	producer := s.newProducer()

	tag := types.UserTag{
		Device: types.Pc,
		Action: types.View,
	}

	err := producer.Send(tag)
	s.Assert().NoErrorf(err, "failed to send message")

	client, err := sarama.NewClient(s.kafkaAddresses(), nil)
	s.Require().NoErrorf(err, "failed to create client")
	partitions, err := client.Partitions(UserTagsTopic)
	s.Require().NoErrorf(err, "failed to get partitions")

	var foundWrittenPartition bool
	for _, partition := range partitions {
		offset, err := client.GetOffset(UserTagsTopic, partition, sarama.OffsetNewest)
		s.Require().NoErrorf(err, "failed to get offset")
		if offset != 0 {
			foundWrittenPartition = true
			break
		}
	}
	s.Assert().Truef(foundWrittenPartition, "no partition has been written to")
}
