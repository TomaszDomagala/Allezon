// Package container provides a simple way to start and stop docker containers for integration tests.
// It is a wrapper around ory/dockertest that provides a simple interface to start and stop docker containers.
package container

import (
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"go.uber.org/zap"
	"math/rand"
	"strings"
	"time"
)

// OnServiceCreated is a callback that is called after the service is started.
type OnServiceCreated func(environment *Environment, service *Service) error

// Service is a docker container that will be started for the test.
type Service struct {
	Name    string
	Options *dockertest.RunOptions
	// OnServicesCreated callback is called after the container is started.
	// It can be used to wait for the container to be ready, check if the container is healthy, set up, etc.
	OnServicesCreated OnServiceCreated

	// Resource is the resource that is created by the dockertest pool.
	Resource *dockertest.Resource
}

// ExposedHostPort returns first exposed port of the service.
func (s *Service) ExposedHostPort() string {
	if len(s.Options.ExposedPorts) == 0 {
		return ""
	}
	return s.Resource.GetHostPort(s.Options.ExposedPorts[0])
}

// Environment is a test suite that starts docker containers before the test and stops them after the test.
type Environment struct {
	Name   string
	Logger *zap.Logger
	Config *Config
	// Services is a list of services that will be started for the test.
	Services []*Service

	Pool *dockertest.Pool

	resources []*dockertest.Resource
	network   *dockertest.Network
}

// NewEnvironment creates a new environment and starts the services.
func NewEnvironment(name string, logger *zap.Logger, services []*Service, config *Config) *Environment {
	if config == nil {
		config = NewConfig()
	}
	return &Environment{
		Name:     name,
		Logger:   logger,
		Services: services,
		Config:   config,
	}
}

func (env *Environment) GetService(name string) *Service {
	for _, service := range env.Services {
		if service.Name == name {
			return service
		}
	}
	return nil
}

func (env *Environment) Run() error {
	var err error

	env.Logger, err = zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("could not create logger: %w", err)
	}
	env.Pool, err = dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("could not connect to pool: %w", err)
	}

	err = env.Pool.Client.Ping()
	if err != nil {
		return fmt.Errorf("could not ping pool: %w", err)
	}

	env.network, err = env.Pool.CreateNetwork(env.dockerString("network"))
	if err != nil {
		return fmt.Errorf("could not create network: %w", err)
	}

	for _, service := range env.Services {
		err = env.addService(service)
		if err != nil {
			return fmt.Errorf("could not add service %s: %w", service.Name, err)
		}
	}

	return nil
}

// GetLogs returns stdout and stderr logs of a service.
func (env *Environment) GetLogs(serviceName string) (string, string, error) {
	service := env.GetService(serviceName)
	if service == nil {
		return "", "", fmt.Errorf("service %s not found", serviceName)
	}
	var stdout, stderr strings.Builder

	err := env.Pool.Client.Logs(docker.LogsOptions{
		Container:         service.Resource.Container.ID,
		OutputStream:      &stdout,
		ErrorStream:       &stderr,
		Stdout:            true,
		Stderr:            true,
		InactivityTimeout: time.Second * 10,
	})
	if err != nil {
		return "", "", fmt.Errorf("could not get logs: %w", err)
	}
	return stdout.String(), stderr.String(), nil
}

// addService starts a service container and runs its OnServicesCreated callback.
func (env *Environment) addService(service *Service) error {
	options := service.Options
	if env.network != nil {
		options.Networks = []*dockertest.Network{env.network}
	}

	options.Name = env.dockerString(service.Name)

	r, err := env.startService(options)
	if err != nil {
		return fmt.Errorf("could not start container %s: %w", options.Name, err)
	}
	env.resources = append(env.resources, r)
	service.Resource = r

	if service.OnServicesCreated != nil {
		err = service.OnServicesCreated(env, service)
		if err != nil {
			return fmt.Errorf("error running OnServicesCreated function: %w", err)
		}
	}
	return nil
}

// startService tries to start a service using exponential backoff.
func (env *Environment) startService(options *dockertest.RunOptions) (*dockertest.Resource, error) {
	var resource *dockertest.Resource

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = env.Config.ServiceStartMaxTime
	bo.MaxInterval = env.Config.ServiceStartMaxInterval

	err := backoff.Retry(func() error {
		var err error
		resource, err = env.doStartService(options)
		return err
	}, bo)

	if err != nil {
		return nil, fmt.Errorf("could not start container %s: %w", options.Name, err)
	}
	return resource, nil
}

func (env *Environment) doStartService(options *dockertest.RunOptions) (*dockertest.Resource, error) {
	if err := env.Pool.RemoveContainerByName(options.Name); err != nil {
		env.Logger.Error("could not remove container", zap.Error(err), zap.String("name", options.Name))
	}
	resource, err := env.Pool.RunWithOptions(options)
	if err != nil {
		return nil, fmt.Errorf("could not start container %s: %w", options.Name, err)
	}
	env.Logger.Info("started container", zap.String("container_id", resource.Container.ID))
	return resource, nil
}

func (env *Environment) Close() error {
	var errs []error

	env.Logger.Info("tearing down test", zap.String("test_name", env.Name))

	for _, resource := range env.resources {
		if resource == nil {
			continue
		}
		env.Logger.Info("stopping container", zap.String("container_id", resource.Container.ID))
		err := env.Pool.Purge(resource)
		if err != nil {
			errs = append(errs, fmt.Errorf("could not stop container %s: %w", resource.Container.Name, err))
		}
	}
	//if env.network != nil {
	//	err := env.Pool.RemoveNetwork(env.network)
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
func (env *Environment) dockerString(name string) string {
	test := strings.Replace(env.Name, "/", "-", -1)
	var suffix string
	if env.Config.ServiceNameRandomSuffix {
		suffix = fmt.Sprintf("-%s", randString(env.Config.ServiceNameRandomSuffixLength))
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
