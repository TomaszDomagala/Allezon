package messaging

import (
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"testing"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
)

var (
	// hostPort is a host:port string that is used to connect to the service.
	hostPort = "localhost:9092"

	redpandaService = &container.Service{
		Name: "redpanda",
		Options: &dockertest.RunOptions{
			Repository:   "vectorized/redpanda",
			Tag:          "latest",
			Hostname:     "redpanda",
			PortBindings: map[docker.Port][]docker.PortBinding{"9092/tcp": {{HostIP: "localhost", HostPort: "9092"}}},
		},
		AfterRun: func(env *container.Environment, resource *dockertest.Resource) error {
			// Wait for the service to be ready.
			env.Logger.Info("waiting for redpanda to start")
			err := env.Pool.Retry(func() error {
				env.Logger.Debug("checking if redpanda is ready")
				client, err := sarama.NewClient([]string{hostPort}, nil)
				if err != nil {
					return fmt.Errorf("failed to create client: %w", err)
				}
				_, err = client.Controller()
				if err != nil {
					return fmt.Errorf("failed to get controller: %w", err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to wait for redpanda: %w", err)
			}
			env.Logger.Info("redpanda started")
			return nil
		},
	}
)

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
	s.env = container.NewEnvironment(s.T().Name(), s.logger, []*container.Service{redpandaService})
	err := s.env.Run()
	if err != nil {
		errClose := s.env.Close()
		s.Assert().NoErrorf(errClose, "could not close environment after error")
		s.Require().NoErrorf(err, "could not run environment")
	}
}

func (s *MessagingSuite) TearDownTest() {
	err := s.env.Close()
	s.Require().NoErrorf(err, "could not close environment")
	s.env = nil
}

func (s *MessagingSuite) TestNewProducer() {
	_, err := NewProducer(s.logger, []string{hostPort})
	s.Require().NoErrorf(err, "failed to create producer")
}

func (s *MessagingSuite) TestProducer_Send() {
	producer, err := NewProducer(s.logger, []string{hostPort})
	s.Require().NoErrorf(err, "failed to create producer")

	tag := types.UserTag{
		Device: types.Pc,
		Action: types.View,
	}

	err = producer.Send(tag)
	s.Assert().NoErrorf(err, "failed to send message")

	client, err := sarama.NewClient([]string{hostPort}, nil)
	offset, err := client.GetOffset(UserTagsTopic, 0, sarama.OffsetNewest)
	s.Require().NoErrorf(err, "failed to get offset")
	s.Require().Equal(int64(1), offset)
}
