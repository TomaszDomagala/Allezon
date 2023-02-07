package db

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
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
	m, err := NewClientFromAddresses(s.logger, hostPort)
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

func sortActionAggregates(agg []ActionAggregates) {
	sort.Slice(agg, func(i, j int) bool {
		return agg[i].Key.encode() < agg[j].Key.encode()
	})
}

type aggregates struct {
	views []ActionAggregates
	buys  []ActionAggregates
}

func (s *DBSuite) compareAggregates(expected, actual aggregates) {
	sortActionAggregates(expected.views)
	sortActionAggregates(expected.buys)

	sortActionAggregates(actual.views)
	sortActionAggregates(actual.buys)

	s.Require().Equal(expected, actual, "aggregates not equal")
}

func (s *DBSuite) getAggregates(a AggregatesClient, t time.Time) (agg aggregates) {
	var err error
	agg.views, err = a.Get(t, types.View)
	s.Require().NoErrorf(err, "error getting from the database")
	agg.buys, err = a.Get(t, types.Buy)
	s.Require().NoErrorf(err, "error getting from the database")
	return
}

func (s *DBSuite) Test_Aggregates() {
	m := s.newClient()
	a := m.Aggregates()

	k1 := AggregateKey{
		CategoryId: 1,
		BrandId:    2,
		Origin:     3,
	}
	k2 := AggregateKey{
		CategoryId: 10,
		BrandId:    20,
		Origin:     30,
	}

	t := aggregates{
		views: []ActionAggregates{
			{
				Key:   k1,
				Sum:   42,
				Count: 2,
			},
			{
				Key:   k2,
				Sum:   69,
				Count: 3,
			},
		},
		buys: []ActionAggregates{
			{
				Key:   k1,
				Sum:   69,
				Count: 3,
			},
			{
				Key:   k2,
				Sum:   42,
				Count: 2,
			},
		},
	}
	min := time.Now()

	// Add views.
	err := a.Add(k1, types.UserTag{Action: types.View, Time: min, ProductInfo: types.ProductInfo{Price: 21}})
	s.Require().NoErrorf(err, "error inserting to the database")
	err = a.Add(k1, types.UserTag{Action: types.View, Time: min, ProductInfo: types.ProductInfo{Price: 21}})
	s.Require().NoErrorf(err, "error inserting to the database")
	err = a.Add(k2, types.UserTag{Action: types.View, Time: min, ProductInfo: types.ProductInfo{Price: 23}})
	s.Require().NoErrorf(err, "error inserting to the database")
	err = a.Add(k2, types.UserTag{Action: types.View, Time: min, ProductInfo: types.ProductInfo{Price: 23}})
	s.Require().NoErrorf(err, "error inserting to the database")
	err = a.Add(k2, types.UserTag{Action: types.View, Time: min, ProductInfo: types.ProductInfo{Price: 23}})
	s.Require().NoErrorf(err, "error inserting to the database")

	// Add buys.
	err = a.Add(k2, types.UserTag{Action: types.Buy, Time: min, ProductInfo: types.ProductInfo{Price: 21}})
	s.Require().NoErrorf(err, "error inserting to the database")
	err = a.Add(k2, types.UserTag{Action: types.Buy, Time: min, ProductInfo: types.ProductInfo{Price: 21}})
	s.Require().NoErrorf(err, "error inserting to the database")
	err = a.Add(k1, types.UserTag{Action: types.Buy, Time: min, ProductInfo: types.ProductInfo{Price: 23}})
	s.Require().NoErrorf(err, "error inserting to the database")
	err = a.Add(k1, types.UserTag{Action: types.Buy, Time: min, ProductInfo: types.ProductInfo{Price: 23}})
	s.Require().NoErrorf(err, "error inserting to the database")
	err = a.Add(k1, types.UserTag{Action: types.Buy, Time: min, ProductInfo: types.ProductInfo{Price: 23}})
	s.Require().NoErrorf(err, "error inserting to the database")

	res := s.getAggregates(a, min)
	s.compareAggregates(t, res)
}

func (s *DBSuite) Test_Aggregates_ReturnsKeyNotFoundErrorOnKeyNotFound() {
	m := s.newClient()

	a := m.Aggregates()
	min := time.Now()

	_, err := a.Get(min, types.View)
	s.Require().ErrorIs(err, KeyNotFoundError, "expected KeyNotFoundError")
	_, err = a.Get(min, types.Buy)
	s.Require().ErrorIs(err, KeyNotFoundError, "expected KeyNotFoundError")
}

func (s *DBSuite) Test_Aggregates_MinuteRounding() {
	m := s.newClient()
	a := m.Aggregates()

	k1 := AggregateKey{
		CategoryId: 1,
		BrandId:    2,
		Origin:     3,
	}
	k2 := AggregateKey{
		CategoryId: 10,
		BrandId:    20,
		Origin:     30,
	}

	t := aggregates{
		views: []ActionAggregates{
			{
				Key:   k1,
				Sum:   6,
				Count: 1,
			},
		},
		buys: []ActionAggregates{
			{
				Key:   k2,
				Sum:   9,
				Count: 1,
			},
		},
	}
	min := time.Now()
	min = min.Add(-(time.Duration(min.Nanosecond()) + time.Second*time.Duration(min.Second()))) // Round to exactly a minute.

	// Add views.
	err := a.Add(k1, types.UserTag{Action: types.View, Time: min, ProductInfo: types.ProductInfo{Price: 6}})
	s.Require().NoErrorf(err, "error inserting to the database")

	// Add buys.
	err = a.Add(k2, types.UserTag{Action: types.Buy, Time: min, ProductInfo: types.ProductInfo{Price: 9}})
	s.Require().NoErrorf(err, "error inserting to the database")

	res := s.getAggregates(a, min)
	s.compareAggregates(t, res)

	res = s.getAggregates(a, min.Add(30*time.Second))
	s.compareAggregates(t, res)

	res = s.getAggregates(a, min.Add(time.Minute-time.Nanosecond))
	s.compareAggregates(t, res)

	_, err = a.Get(min.Add(-time.Nanosecond), types.View)
	s.Require().ErrorIs(err, KeyNotFoundError, "error getting from the database")
	_, err = a.Get(min.Add(-time.Nanosecond), types.Buy)
	s.Require().ErrorIs(err, KeyNotFoundError, "error getting from the database")
	_, err = a.Get(min.Add(time.Minute), types.View)
	s.Require().ErrorIs(err, KeyNotFoundError, "error getting from the database")
	_, err = a.Get(min.Add(time.Minute), types.Buy)
	s.Require().ErrorIs(err, KeyNotFoundError, "error getting from the database")
}
