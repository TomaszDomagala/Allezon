package tests

import (
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return abs
}

var (
	// aerospikeHostPort is a host:port string that is used to connect to the aerospike service.
	aerospikeHostPort = "localhost:3000"

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
		OnServicesCreated: func(env *container.Environment, resource *dockertest.Resource) error {
			// Wait for the service to be ready.
			env.Logger.Info("waiting for aerospike to start")
			err := env.Pool.Retry(func() error {
				env.Logger.Debug("checking if aerospike is ready")
				hosts, err := as.NewHosts(aerospikeHostPort)
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

	idgetterPort       = "8088"
	idgetterDockerPort = idgetterPort + "/tcp"

	idgetterService = &container.Service{
		Name: "idgetter",
		Options: &dockertest.RunOptions{
			Repository:   idgetterImageName,
			Tag:          idgetterImageTag,
			Hostname:     "idgetter",
			ExposedPorts: []string{idgetterDockerPort},
			Env: []string{
				"PORT=" + idgetterPort,
				"DB_ADDRESSES=aerospike:3000",
			},
		},
		OnServicesCreated: func(environment *container.Environment, resource *dockertest.Resource) error {
			hostport := resource.GetHostPort(idgetterDockerPort)
			if hostport == "" {
				return fmt.Errorf("failed to get host port for %s", idgetterDockerPort)
			}
			healthURL := fmt.Sprintf("http://%s/health", hostport)

			// Wait for the service to be ready.
			environment.Logger.Info("waiting for idgetter to start")
			err := environment.Pool.Retry(func() error {
				environment.Logger.Debug("checking if idgetter is ready at", zap.String("url", healthURL))
				_, err := http.Get(healthURL)
				if err != nil {
					return fmt.Errorf("failed to get health: %w", err)
				}

				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to wait for idgetter: %w", err)
			}
			environment.Logger.Info("idgetter started")
			return nil
		},
	}
)

type IDGetterIntegrationTestsSuite struct {
	suite.Suite
	logger *zap.Logger

	env *container.Environment
}

func TestIDGetterIntegration(t *testing.T) {
	suite.Run(t, new(IDGetterIntegrationTestsSuite))
}

func (s *IDGetterIntegrationTestsSuite) SetupSuite() {
	var err error

	s.logger, err = zap.NewDevelopment()
	s.Require().NoErrorf(err, "could not create logger")

	out, errs, err := buildIDGetterImage()
	s.Require().NoErrorf(err, "could not get dockerfiles path: stdout:\n\n%s stderr:\n\n%s", out, errs)
}

func (s *IDGetterIntegrationTestsSuite) SetupTest() {
	s.env = container.NewEnvironment(s.T().Name(), s.logger, []*container.Service{
		aerospikeService,
		idgetterService,
	}, nil)

	err := s.env.Run()
	if err != nil {
		errClose := s.env.Close()
		s.Assert().NoErrorf(errClose, "could not close environment after error")
		s.Require().NoErrorf(err, "could not run environment")
	}
}

func (s *IDGetterIntegrationTestsSuite) TearDownTest() {
	err := s.env.Close()
	s.Require().NoErrorf(err, "could not close environment")
	s.env = nil
}

func (s *IDGetterIntegrationTestsSuite) getIDGetterURL() (string, error) {
	idgetterSrv := s.env.GetService("idgetter")
	if idgetterSrv == nil {
		return "", fmt.Errorf("could not get idgetter service")
	}

	hostport := idgetterSrv.ExposedHostPort()
	if hostport == "" {
		return "", fmt.Errorf("could not get host port for idgetter service")
	}

	return hostport, nil
}

func (s *IDGetterIntegrationTestsSuite) TestIDGetter() {
	url, err := s.getIDGetterURL()
	s.Require().NoErrorf(err, "could not get idgetter url")

	// Create a new client without caching.
	client := idGetter.NewPureClient(http.Client{Timeout: 5 * time.Second}, url)

	calls := []struct {
		category   string
		name       string
		expectedID int32
	}{
		{category: "food", name: "apple", expectedID: 1},
		{category: "food", name: "banana", expectedID: 2},
		{category: "food", name: "orange", expectedID: 3},
		{category: "food", name: "apple", expectedID: 1},
		{category: "food", name: "banana", expectedID: 2},
		{category: "food", name: "orange", expectedID: 3},
		{category: "transport", name: "car", expectedID: 1},
		{category: "transport", name: "bike", expectedID: 2},
		{category: "transport", name: "bus", expectedID: 3},
		{category: "transport", name: "car", expectedID: 1},
		{category: "transport", name: "bike", expectedID: 2},
		{category: "transport", name: "bus", expectedID: 3},
	}

	for _, call := range calls {
		id, err := client.GetID(call.category, call.name)
		s.Assert().NoErrorf(err, "could not get id for %s/%s", call.category, call.name)
		s.Assert().Equalf(call.expectedID, id, "unexpected id for %s/%s", call.category, call.name)
	}
}
