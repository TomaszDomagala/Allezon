package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return abs
}

var (
	// hostPort is a host:port string that is used to connect to the service.
	hostPort = "localhost:3000"

	aerospikeService = &container.Service{
		Name: "aerospike",
		Options: &dockertest.RunOptions{
			Repository: "aerospike",
			Tag:        "ce-6.2.0.2",
			Hostname:   "aerospike",
			Mounts:     []string{absPath("./assets") + ":/assets"},
			PortBindings: map[docker.Port][]docker.PortBinding{
				"3000/tcp": {{HostIP: "localhost", HostPort: "3000"}},
				"3001/tcp": {{HostIP: "localhost", HostPort: "3001"}},
				"3002/tcp": {{HostIP: "localhost", HostPort: "3002"}},
			},
			Cmd: []string{"--config-file", "/assets/aerospike.conf"},
		},
		OnServicesCreated: func(env *container.Environment, _ *container.Service) error {
			// Wait for the service to be ready.
			env.Logger.Info("waiting for aerospike to start")
			err := env.Pool.Retry(func() error {
				env.Logger.Debug("checking if aerospike is ready")
				hosts, err := as.NewHosts(hostPort)
				if err != nil {
					return fmt.Errorf("failed to get hosts, %w", err)
				}
				_, err = as.NewClientWithPolicyAndHost(nil, hosts...)
				if err != nil {
					return fmt.Errorf("failed to create client: %w", err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to wait for aerospike: %w", err)
			}
			env.Logger.Info("aerospike started")

			return nil
		},
	}
)

// DBSuite is a suite for db integration tests.
type DBSuite struct {
	suite.Suite
	logger *zap.Logger

	// env is created for each test case.
	env *container.Environment
}

// TestIDGetterDBSuite is an entry point for running all tests in this package.
func TestIDGetterDBSuite(t *testing.T) {
	suite.Run(t, new(DBSuite))
}

func (s *DBSuite) SetupSuite() {
	var err error

	s.logger, err = zap.NewDevelopment()
	s.Require().NoErrorf(err, "could not create logger")
}

func (s *DBSuite) SetupTest() {
	s.env = container.NewEnvironment(s.T().Name(), s.logger, []*container.Service{aerospikeService}, nil)
	err := s.env.Run()
	if err != nil {
		a, e := os.Getwd()
		fmt.Println(a, e)
		s.Require().NoErrorf(err, "could not run environment")
	}
}

func (s *DBSuite) TearDownTest() {
	err := s.env.Close()
	s.Require().NoErrorf(err, "could not close environment")
	s.env = nil
}

func (s *DBSuite) TestNewModifier() {
	_, err := NewClientFromAddresses([]string{hostPort})
	s.Require().NoErrorf(err, "failed to create client")
}

func (s *DBSuite) TestNewGetter() {
	_, err := NewClientFromAddresses([]string{hostPort})
	s.Require().NoErrorf(err, "failed to create getter")
}

func (s *DBSuite) Test_Ids() {
	c, err := NewClientFromAddresses([]string{hostPort})
	s.Require().NoErrorf(err, "failed to create client")

	const name = "foobar"
	t := "foo"

	l, err := c.AppendElement(name, t)
	s.Require().NoErrorf(err, "failed to create record")
	s.Require().Equal(1, l, "list length mismatch")

	got, err := c.GetElements(name)
	s.Require().Equal([]string{t}, got)

	t2 := "bar"

	l2, err := c.AppendElement(name, t2)
	s.Require().NoErrorf(err, "failed to update record")
	s.Require().Equal(2, l2, "list length mismatch")

	updated, err := c.GetElements(name)
	s.Require().Equal([]string{t, t2}, updated)
}

func (s *DBSuite) Test_Ids_ErrorOnDuplicate() {
	c, err := NewClientFromAddresses([]string{hostPort})
	s.Require().NoErrorf(err, "failed to create client")

	const name = "foobar"
	t := "foo"

	_, err = c.AppendElement(name, t)
	s.Require().NoErrorf(err, "failed to create record")

	got, err := c.GetElements(name)
	s.Require().Equal([]string{t}, got)

	t2 := "foo"

	_, err = c.AppendElement(name, t2)
	s.Require().ErrorIs(err, ElementExists, "no error on element exists")
}
