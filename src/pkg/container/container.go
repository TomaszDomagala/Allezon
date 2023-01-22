// Package container provides a simple way to start and stop docker containers for integration tests.
// It is a wrapper around ory/dockertest that provides a simple interface to start and stop docker containers.
package container

import (
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"math/rand"
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
	Config   *Config
	services []*Service

	Pool      *dockertest.Pool
	resources []*dockertest.Resource
	//network   *dockertest.Network
}

func NewEnvironment(name string, logger *zap.Logger, services []*Service, config *Config) *Environment {
	if config == nil {
		config = NewConfig()
	}
	return &Environment{
		Name:     name,
		Logger:   logger,
		services: services,
		Config:   config,
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

	//s.network, err = s.Pool.CreateNetwork(prefixTestName(s.Name, "network"))
	if err != nil {
		return fmt.Errorf("could not create network: %w", err)
	}

	for _, service := range s.services {
		options := service.Options
		options.Name = s.dockerString(service.Name)

		r, err := s.startService(options)
		if err != nil {
			return fmt.Errorf("could not start container %s: %w", options.Name, err)
		}
		s.resources = append(s.resources, r)

		if service.AfterRun != nil {
			err = service.AfterRun(s, r)
			if err != nil {
				return fmt.Errorf("error running AfterRun function: %w", err)
			}
		}

	}

	return nil
}

// startService tries to start a service using exponential backoff.
func (s *Environment) startService(options *dockertest.RunOptions) (*dockertest.Resource, error) {
	var resource *dockertest.Resource

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = s.Config.ServiceStartMaxTime
	bo.MaxInterval = s.Config.ServiceStartMaxInterval

	err := backoff.Retry(func() error {
		var err error
		resource, err = s.doStartService(options)
		return err
	}, bo)

	if err != nil {
		return nil, fmt.Errorf("could not start container %s: %w", options.Name, err)
	}
	return resource, nil
}

func (s *Environment) doStartService(options *dockertest.RunOptions) (*dockertest.Resource, error) {
	if err := s.Pool.RemoveContainerByName(options.Name); err != nil {
		s.Logger.Error("could not remove container", zap.Error(err), zap.String("name", options.Name))
	}
	resource, err := s.Pool.RunWithOptions(options)
	if err != nil {
		return nil, fmt.Errorf("could not start container %s: %w", options.Name, err)
	}
	s.Logger.Info("started container", zap.String("container_id", resource.Container.ID))
	return resource, nil
}

func (s *Environment) Close() error {
	var errs []error

	s.Logger.Info("tearing down test", zap.String("test_name", s.Name))

	for _, resource := range s.resources {
		if resource == nil {
			continue
		}
		s.Logger.Info("stopping container", zap.String("container_id", resource.Container.ID))
		err := s.Pool.Purge(resource)
		if err != nil {
			errs = append(errs, fmt.Errorf("could not stop container %s: %w", resource.Container.Name, err))
		}
	}
	//if s.network != nil {
	//	err := s.Pool.RemoveNetwork(s.network)
	//	if err != nil {
	//		errs = append(errs, fmt.Errorf("could not remove network: %w", err))
	//	}
	//}

	if len(errs) > 0 {
		return fmt.Errorf("could not tear down environment: %v", errs)
	}
	return nil
}

// dockerString returns a name prefixed with the test name, in a docker friendly way.
// If ServiceNameRandomSuffix is set, a random suffix is added to the name.
func (s *Environment) dockerString(name string) string {
	test := strings.Replace(s.Name, "/", "-", -1)
	var suffix string
	if s.Config.ServiceNameRandomSuffix {
		suffix = fmt.Sprintf("-%s", randString(s.Config.ServiceNameRandomSuffixLength))
	}

	return fmt.Sprintf("%s-%s%s", test, name, suffix)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
