package containerutils

import (
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
)

var (
	RedpandaPort           = "29092"
	RedpandaDockerPort     = docker.Port(RedpandaPort + "/tcp")
	RedpandaHostPort       = "9092"
	RedpandaDockerHostPort = docker.Port(RedpandaHostPort + "/tcp")

	// RedpandaService is a container.Service that runs redpanda. Note that the service does not have any
	// topic created by default. Also, to connect to the service from the host machine, you need to use
	// the PortBindings option instead of ExposedPorts (because of how kafka protocol works).
	// https://stackoverflow.com/q/51630260 - maybe there is a way to make it work with ExposedPorts? IDK for the time being.
	RedpandaService = &container.Service{
		Name: "redpanda",
		Options: &dockertest.RunOptions{
			Repository:   "vectorized/redpanda",
			Tag:          "latest",
			Hostname:     "redpanda",
			ExposedPorts: []string{string(RedpandaDockerHostPort)},
			PortBindings: map[docker.Port][]docker.PortBinding{
				RedpandaDockerPort:     {{HostIP: "localhost", HostPort: RedpandaPort}},
				RedpandaDockerHostPort: {{HostIP: "localhost", HostPort: RedpandaHostPort}},
			},
			Cmd: []string{
				"redpanda", "start",
				"--kafka-addr", fmt.Sprintf("PLAINTEXT://0.0.0.0:%s,OUTSIDE://0.0.0.0:%s", RedpandaPort, RedpandaHostPort),
				"--advertise-kafka-addr", fmt.Sprintf("PLAINTEXT://redpanda:%s,OUTSIDE://localhost:%s", RedpandaPort, RedpandaHostPort),
			},
		},
		OnServicesCreated: func(env *container.Environment, service *container.Service) error {
			hostport := env.GetService("redpanda").ExposedHostPort()

			// Wait for the service to be ready.
			env.Logger.Info("waiting for redpanda to start")
			err := env.Pool.Retry(func() error {
				return redpandaHearthCheck(env, hostport)
			})
			if err != nil {
				return fmt.Errorf("failed to wait for redpanda: %w", err)
			}
			env.Logger.Info("redpanda started")

			return nil
		},
	}
)

// redpandaHearthCheck checks if redpanda is ready to accept connections.
func redpandaHearthCheck(env *container.Environment, address string) error {
	env.Logger.Debug("checking if redpanda is ready")
	client, err := sarama.NewClient([]string{address}, nil)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer func() {
		err := client.Close()
		if err != nil {
			env.Logger.Error("failed to close client", zap.Error(err))
		}
	}()
	err = client.RefreshMetadata()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	return nil
}
