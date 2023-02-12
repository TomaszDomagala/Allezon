package containerutils

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/ory/dockertest/v3"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
)

var (
	AerospikePort       = "3000"
	AerospikeDockerPort = "3000/tcp"

	// AerospikeService is a container.Service that can be used to start Aerospike.
	// It forwards the Aerospike port to the host on a random port.
	AerospikeService = &container.Service{
		Name: "aerospike",
		Options: &dockertest.RunOptions{
			Repository:   "aerospike/aerospike-server",
			Tag:          "5.5.0.7",
			Hostname:     "aerospike",
			Mounts:       []string{pathToConfDir() + ":/assets"},
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

//go:embed assets/aerospike.conf
var aerospikeConf []byte

var aerospikeConfDir string
var aerospikeConfOnce sync.Once

func pathToConfDir() string {
	aerospikeConfOnce.Do(func() {
		var err error
		aerospikeConfDir, err = os.MkdirTemp(os.TempDir(), "assets-*")
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(aerospikeConfDir, "aerospike.conf"), aerospikeConf, 0444); err != nil {
			panic(err)
		}
	})
	return aerospikeConfDir
}
