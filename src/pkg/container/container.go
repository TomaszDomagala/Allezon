// Package container provides a simple way to start and stop docker containers for integration tests.
// It is a wrapper around ory/dockertest that provides a simple interface to start and stop docker containers.
package container

import (
	"fmt"
	"strings"

	"github.com/ory/dockertest/v3"
	"go.uber.org/zap"
)

// Service is a docker container that will be started for the test.
type Service struct {
	Name    string
	Options *dockertest.RunOptions
	// AfterRun callback is called after the container is started.
	// It can be used to wait for the container to be ready, check if the container is healthy, set up, etc.
	AfterRun func(env *Environment, resource *dockertest.Resource) error
}

// Environment is a test suite that starts docker containers before the test and stops them after the test.
type Environment struct {
	Name     string
	Logger   *zap.Logger
	services []*Service

	Pool      *dockertest.Pool
	resources []*dockertest.Resource
	network   *dockertest.Network
}

func NewEnvironment(name string, logger *zap.Logger, services []*Service) *Environment {
	return &Environment{
		Name:     name,
		Logger:   logger,
		services: services,
	}
}

func (s *Environment) Run() error {
	var err error

	s.Logger, err = zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("could not create logger: %w", err)
	}
	s.Pool, err = dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("could not connect to pool: %w", err)
	}

	err = s.Pool.Client.Ping()
	if err != nil {
		return fmt.Errorf("could not ping pool: %w", err)
	}

	s.network, err = s.Pool.CreateNetwork(prefixTestName(s.Name, "network"))
	if err != nil {
		return fmt.Errorf("could not create network: %w", err)
	}

	for _, service := range s.services {
		s.Logger.Info("starting container", zap.String("name", service.Name))

		options := service.Options
		options.Name = prefixTestName(s.Name, service.Name)

		if err = s.Pool.RemoveContainerByName(options.Name); err != nil {
			s.Logger.Error("could not remove container", zap.Error(err))
		}
		r, err := s.Pool.RunWithOptions(options)
		if err != nil {
			return fmt.Errorf("could not start container: %w", err)
		}
		if service.AfterRun != nil {
			err = service.AfterRun(s, r)
			if err != nil {
				return fmt.Errorf("error running AfterRun function: %w", err)
			}
		}

		s.resources = append(s.resources, r)
		s.Logger.Info("started container", zap.String("container_id", r.Container.ID))
	}

	return nil
}

func (s *Environment) Close() error {
	var errs []error

	s.Logger.Info("tearing down test", zap.String("test_name", s.Name))

	for _, resource := range s.resources {
		s.Logger.Info("stopping container", zap.String("container_id", resource.Container.ID))
		err := s.Pool.Purge(resource)
		if err != nil {
			errs = append(errs, fmt.Errorf("could not stop container: %w", err))
		}
	}
	if s.network != nil {
		err := s.Pool.RemoveNetwork(s.network)
		if err != nil {
			errs = append(errs, fmt.Errorf("could not remove network: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("could not tear down environment: %v", errs)
	}
	return nil
}

// prefixTestName returns a name prefixed with the test name, in a docker friendly way.
func prefixTestName(test, name string) string {
	test = strings.Replace(test, "/", "-", -1)
	return fmt.Sprintf("%s-%s", test, name)
}
