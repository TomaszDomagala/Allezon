package db

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
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

// TestDBSuite is an entry point for running all tests in this package.
func TestDBSuite(t *testing.T) {
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
	s.Require().NoErrorf(err, "could not run environment")
}

func (s *DBSuite) TearDownTest() {
	err := s.env.Close()
	s.Require().NoErrorf(err, "could not close environment")
	s.env = nil
}

func (s *DBSuite) newClient() Client {
	m, err := NewClientFromAddresses([]string{hostPort})
	s.Require().NoErrorf(err, "failed to create client")
	return m
}

func (s *DBSuite) TestNewClient() {
	cl := s.newClient()
	runtime.KeepAlive(cl)
}

func (s *DBSuite) Test_UserProfiles() {
	m := s.newClient()

	up := m.UserProfiles()
	const cookie = "foobar"
	t := UserProfile{
		Views: []types.UserTag{{Cookie: "42"}},
		Buys:  []types.UserTag{{Cookie: "6"}, {Cookie: "9"}},
	}

	err := up.Update(cookie, t, 0)
	s.Require().NoErrorf(err, "failed to create record")

	got, err := up.Get(cookie)
	s.Require().NoErrorf(err, "failed to get record")
	s.Require().Equal(t, got.Result)
	s.Require().Equal(uint32(1), got.Generation)

	got.Result.Buys = got.Result.Buys[:1]

	err = up.Update(cookie, got.Result, got.Generation)
	s.Require().NoErrorf(err, "failed to update record")

	updated, err := up.Get(cookie)
	s.Require().NoErrorf(err, "failed to get record")
	s.Require().Equal(got.Result, updated.Result)
	s.Require().Equal(got.Generation+1, updated.Generation)
}

func (s *DBSuite) Test_UserProfiles_ReturnsKeyNotFoundErrorOnKeyNotFound() {
	m := s.newClient()

	up := m.UserProfiles()

	_, err := up.Get("")
	s.Require().ErrorIs(err, KeyNotFoundError, "expected KeyNotFoundError")
}

func (s *DBSuite) Test_UserProfiles_Update_ErrorOnGenerationMismatch() {
	m := s.newClient()

	up := m.UserProfiles()
	const cookie = "foobar"
	t := UserProfile{}

	err := up.Update(cookie, t, 0)
	s.Require().NoErrorf(err, "failed to create record")

	err = up.Update(cookie, t, 0)
	s.Require().ErrorIs(err, GenerationMismatch, "expected generation mismatch error")

	err = up.Update(cookie, t, 2)
	s.Require().ErrorIs(err, GenerationMismatch, "expected generation mismatch error")
}

func (s *DBSuite) Test_Aggregates() {
	m := s.newClient()

	a := m.Aggregates()
	t := Aggregates{
		Views: TypeAggregates{
			Sum: []ActionAggregates{
				{
					CategoryId: 1,
					BrandId:    2,
					Origin:     3,
					Data:       4,
				},
				{
					CategoryId: 5,
					BrandId:    4,
					Origin:     3,
					Data:       2,
				},
			},
			Count: []ActionAggregates{
				{
					CategoryId: 10,
					BrandId:    20,
					Origin:     30,
					Data:       40,
				},
				{
					CategoryId: 50,
					BrandId:    40,
					Origin:     30,
					Data:       20,
				},
			},
		},
		Buys: TypeAggregates{
			Sum: []ActionAggregates{
				{
					CategoryId: 11,
					BrandId:    12,
					Origin:     13,
					Data:       14,
				},
				{
					CategoryId: 25,
					BrandId:    24,
					Origin:     23,
					Data:       22,
				},
			},
			Count: []ActionAggregates{
				{
					CategoryId: 110,
					BrandId:    120,
					Origin:     130,
					Data:       140,
				},
				{
					CategoryId: 150,
					BrandId:    140,
					Origin:     130,
					Data:       120,
				},
			},
		},
	}
	min := time.Now()

	err := a.Update(min, t, 0)
	s.Require().NoErrorf(err, "failed to create record")

	got, err := a.Get(min)
	s.Require().NoErrorf(err, "failed to get record")
	s.Require().Equal(t, got.Result)
	s.Require().Equal(uint32(1), got.Generation)

	newT := Aggregates{
		Views: t.Buys,
		Buys:  t.Views,
	}

	err = a.Update(min, newT, got.Generation)
	s.Require().NoErrorf(err, "failed to update record")

	updated, err := a.Get(min)
	s.Require().NoErrorf(err, "failed to get record")
	s.Require().Equal(newT, updated.Result)
	s.Require().Equal(got.Generation+1, updated.Generation)
}

func (s *DBSuite) Test_Aggregates_ReturnsKeyNotFoundErrorOnKeyNotFound() {
	m := s.newClient()

	a := m.Aggregates()
	min := time.Now()

	_, err := a.Get(min)
	s.Require().ErrorIs(err, KeyNotFoundError, "expected KeyNotFoundError")
}

func (s *DBSuite) Test_Aggregates_MinuteRounding() {
	m := s.newClient()

	a := m.Aggregates()
	t := Aggregates{
		Views: TypeAggregates{
			Sum: []ActionAggregates{
				{
					CategoryId: 1,
					BrandId:    2,
					Origin:     3,
					Data:       4,
				},
				{
					CategoryId: 5,
					BrandId:    4,
					Origin:     3,
					Data:       2,
				},
			},
			Count: []ActionAggregates{
				{
					CategoryId: 10,
					BrandId:    20,
					Origin:     30,
					Data:       40,
				},
				{
					CategoryId: 50,
					BrandId:    40,
					Origin:     30,
					Data:       20,
				},
			},
		},
		Buys: TypeAggregates{
			Sum: []ActionAggregates{
				{
					CategoryId: 11,
					BrandId:    12,
					Origin:     13,
					Data:       14,
				},
				{
					CategoryId: 25,
					BrandId:    24,
					Origin:     23,
					Data:       22,
				},
			},
			Count: []ActionAggregates{
				{
					CategoryId: 110,
					BrandId:    120,
					Origin:     130,
					Data:       140,
				},
				{
					CategoryId: 150,
					BrandId:    140,
					Origin:     130,
					Data:       120,
				},
			},
		},
	}
	min := time.Now()
	min = min.Add(-(time.Duration(min.Nanosecond()) + time.Second*time.Duration(min.Second())))

	err := a.Update(min, t, 0)
	s.Require().NoErrorf(err, "failed to create record")

	got, err := a.Get(min)
	s.Require().NoErrorf(err, "failed to get record")
	s.Require().Equal(t, got.Result)
	s.Require().Equal(uint32(1), got.Generation)

	got, err = a.Get(min.Add(30 * time.Second))
	s.Require().NoErrorf(err, "failed to get record")
	s.Require().Equal(t, got.Result)
	s.Require().Equal(uint32(1), got.Generation)

	got, err = a.Get(min.Add(time.Minute - time.Nanosecond))
	s.Require().NoErrorf(err, "failed to get record")
	s.Require().Equal(t, got.Result)
	s.Require().Equal(uint32(1), got.Generation)
}

func (s *DBSuite) Test_Aggregates_Update_ErrorOnGenerationMismatch() {
	m := s.newClient()

	a := m.Aggregates()
	min := time.Now()
	t := Aggregates{}

	err := a.Update(min, t, 0)
	s.Require().NoErrorf(err, "failed to create record")

	err = a.Update(min, t, 0)
	s.Require().ErrorIs(err, GenerationMismatch, "expected generation mismatch error")

	err = a.Update(min, t, 2)
	s.Require().ErrorIs(err, GenerationMismatch, "expected generation mismatch error")
}
