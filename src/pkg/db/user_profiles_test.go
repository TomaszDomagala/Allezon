package db

import (
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"testing"
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
			//Name:       "aerospike",
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
		AfterRun: func(env *container.Environment, resource *dockertest.Resource) error {
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

// TestUserProfilesSuite is an entry point for running all tests in this package.
func TestUserProfilesSuite(t *testing.T) {
	suite.Run(t, new(DBSuite))
}

func (s *DBSuite) SetupSuite() {
	var err error

	s.logger, err = zap.NewDevelopment()
	s.Require().NoErrorf(err, "could not create logger")
}

func (s *DBSuite) SetupTest() {
	s.env = container.NewEnvironment(s.T().Name(), s.logger, []*container.Service{aerospikeService})
	err := s.env.Run()
	if err != nil {
		a, e := os.Getwd()
		fmt.Println(a, e)
		errClose := s.env.Close()
		s.Assert().NoErrorf(errClose, "could not close environment after error")
		s.Require().NoErrorf(err, "could not run environment")
	}
}

func (s *DBSuite) TearDownTest() {
	err := s.env.Close()
	s.Require().NoErrorf(err, "could not close environment")
	s.env = nil
}

func (s *DBSuite) TestNewModifier() {
	_, err := NewModifierFromAddresses([]string{hostPort})
	s.Require().NoErrorf(err, "failed to create modifier")
}

func (s *DBSuite) TestNewGetter() {
	_, err := NewGetterFromAddresses([]string{hostPort})
	s.Require().NoErrorf(err, "failed to create getter")
}

func (s *DBSuite) TestModifier_UserProfiles() {
	m, err := NewModifierFromAddresses([]string{hostPort})
	s.Require().NoErrorf(err, "failed to create modifier")

	up := m.UserProfiles()
	const cookie = "foobar"
	t := UserProfile{
		Views: []types.UserTag{{Cookie: "42"}},
		Buys:  []types.UserTag{{Cookie: "6"}, {Cookie: "9"}},
	}

	err = up.Update(cookie, t, 0)
	s.Require().NoErrorf(err, "failed to create record")

	got, err := up.Get(cookie)
	s.Require().Equal(t, got.Result)
	s.Require().Equal(uint32(1), got.Generation)

	got.Result.Buys = got.Result.Buys[:1]

	err = up.Update(cookie, got.Result, got.Generation)
	s.Require().NoErrorf(err, "failed to update record")

	updated, err := up.Get(cookie)
	s.Require().Equal(got.Result, updated.Result)
	s.Require().Equal(got.Generation+1, updated.Generation)
}
