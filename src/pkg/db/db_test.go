package db

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/nsf/jsondiff"
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
	now := time.Now()

	const cookieFoo = "foo"
	const cookieBar = "bar"
	profiles := map[string]UserProfile{
		cookieFoo: {
			Views: []types.UserTag{{Time: now, Action: types.View, Cookie: cookieFoo}},
			Buys:  []types.UserTag{{Time: now.Add(time.Second), Action: types.Buy, Cookie: cookieFoo}, {Time: now.Add(time.Minute), Action: types.Buy, Cookie: cookieFoo}},
		},
		cookieBar: {
			Views: []types.UserTag{{Time: now.Add(time.Second), Action: types.View, Cookie: cookieBar}, {Time: now.Add(time.Minute), Action: types.View, Cookie: cookieBar}},
			Buys:  []types.UserTag{{Time: now, Action: types.Buy, Cookie: cookieBar}},
		},
	}

	// Insert
	for _, profile := range profiles {
		for i, view := range profile.Views {
			newLen, err := up.Add(&view)
			s.Require().NoErrorf(err, "failed to create record")
			s.Require().Equal(i+1, newLen, "length mismatch")
		}
		for i, buy := range profile.Buys {
			newLen, err := up.Add(&buy)
			s.Require().NoErrorf(err, "failed to create record")
			s.Require().Equal(i+1, newLen, "length mismatch")
		}
	}
	// Check
	for cookie, profile := range profiles {
		res, err := up.Get(cookie)
		s.Require().NoErrorf(err, "failed to get record")
		resSer, err := json.Marshal(res)
		s.Require().NoErrorf(err, "failed to serialize profile")
		profileSer, err := json.Marshal(profile)
		s.Require().NoErrorf(err, "failed to serialize profile")
		opts := jsondiff.DefaultConsoleOptions()
		status, diff := jsondiff.Compare(profileSer, resSer, &opts)
		s.Assert().Equal(jsondiff.FullMatch, status, diff)
	}
}

func (s *DBSuite) Test_UserProfiles_RemoveOverLimit() {
	m := s.newClient()

	up := m.UserProfiles()
	now := time.Now()

	const cookieFoo = "foo"
	profile := UserProfile{
		Buys: []types.UserTag{{Time: now.Add(time.Second), Action: types.Buy, Cookie: cookieFoo}, {Time: now.Add(time.Minute), Action: types.Buy, Cookie: cookieFoo}},
	}

	// Insert
	for i, view := range profile.Views {
		newLen, err := up.Add(&view)
		s.Require().NoErrorf(err, "failed to create record")
		s.Require().Equal(i+1, newLen, "length mismatch")
	}
	for i, buy := range profile.Buys {
		newLen, err := up.Add(&buy)
		s.Require().NoErrorf(err, "failed to create record")
		s.Require().Equal(i+1, newLen, "length mismatch")
	}

	// Check insertion
	res, err := up.Get(cookieFoo)
	s.Require().NoErrorf(err, "failed to get record")
	resSer, err := json.Marshal(res)
	s.Require().NoErrorf(err, "failed to serialize profile")
	profileSer, err := json.Marshal(profile)
	s.Require().NoErrorf(err, "failed to serialize profile")
	opts := jsondiff.DefaultConsoleOptions()
	status, diff := jsondiff.Compare(profileSer, resSer, &opts)
	s.Assert().Equal(jsondiff.FullMatch, status, diff)

	// Remove
	err = up.RemoveOverLimit(cookieFoo, types.Buy, 1)
	s.Require().NoErrorf(err, "failed to remove over limit")

	// Check removal
	profile.Buys = profile.Buys[1:]

	res, err = up.Get(cookieFoo)
	s.Require().NoErrorf(err, "failed to get record")
	resSer, err = json.Marshal(res)
	s.Require().NoErrorf(err, "failed to serialize profile")
	profileSer, err = json.Marshal(profile)
	s.Require().NoErrorf(err, "failed to serialize profile")
	opts = jsondiff.DefaultConsoleOptions()
	status, diff = jsondiff.Compare(profileSer, resSer, &opts)
	s.Assert().Equal(jsondiff.FullMatch, status, diff)
}

func (s *DBSuite) Test_UserProfiles_RemoveOverLimit_Errors() {
	m := s.newClient()

	up := m.UserProfiles()

	const cookieFoo = "foo"

	// Non-existing key removal.
	err := up.RemoveOverLimit(cookieFoo, types.View, 10)
	s.Require().NoErrorf(err, "error removing")

	l, err := up.Add(&types.UserTag{Action: types.Buy, Cookie: cookieFoo, Time: time.Now()})
	s.Require().NoErrorf(err, "error adding key")
	s.Require().Equal(1, l, "unexpected length")

	// Non-existing action removal.
	err = up.RemoveOverLimit(cookieFoo, types.View, 10)
	s.Require().NoErrorf(err, "error removing")
}

func (s *DBSuite) Test_UserProfiles_ReturnsKeyNotFoundErrorOnKeyNotFound() {
	m := s.newClient()

	up := m.UserProfiles()

	_, err := up.Get("")
	s.Require().ErrorIs(err, KeyNotFoundError, "expected KeyNotFoundError")
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
