package containerutils

import (
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/ory/dockertest/v3"
	"path/filepath"
)

var (
	AerospikePort       = "3000"
	AerospikeDockerPort = "3000/tcp"

	// AerospikeService is a container.Service that can be used to start Aerospike.
	// It forwards the Aerospike port to the host on a random port.
	AerospikeService = &container.Service{
		Name: "aerospike",
		Options: &dockertest.RunOptions{
			Repository:   "aerospike",
			Tag:          "ce-6.2.0.2",
			Hostname:     "aerospike",
			Mounts:       []string{absPath("./assets") + ":/assets"},
			ExposedPorts: []string{AerospikeDockerPort},
			Cmd:          []string{"--config-file", "/assets/aerospike.conf"},
		},
		OnServicesCreated: func(env *container.Environment, service *container.Service) error {
			address := service.ExposedHostPort()
			if address == "" {
				return fmt.Errorf("failed to get exposed host port for aerospike")
			}

			// Wait for the service to be ready.
			env.Logger.Info("waiting for aerospike to start")
			err := env.Pool.Retry(func() error {
				env.Logger.Debug("checking if aerospike is ready")
				hosts, err := as.NewHosts(address)
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

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return abs
}
