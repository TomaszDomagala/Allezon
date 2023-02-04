package containerutils

import (
	"fmt"

	"github.com/ory/dockertest/v3"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
)

var (
	workerImageName = "tomaszdomagala/allezon-worker"
	workerImageTag  = "latest"

	WorkerPort       = "8082"
	WorkerDockerPort = WorkerPort + "/tcp"

	// WorkerService is a service with worker container. It depends on the redpanda and aerospike services and WILL panic if
	// they are not available. It also depends on the idgetter service, but it will not panic if it is not available.
	// However, it will not be able to process any messages, as they require the idgetter service.
	WorkerService = &container.Service{
		Name: "worker",
		Options: &dockertest.RunOptions{
			Repository:   workerImageName,
			Tag:          workerImageTag,
			Hostname:     "worker",
			ExposedPorts: []string{WorkerDockerPort},
			Env: []string{
				fmt.Sprintf("PORT=%s", WorkerPort),
				fmt.Sprintf("KAFKA_ADDRESSES=redpanda:%s", RedpandaPort),
				fmt.Sprintf("DB_ADDRESSES=aerospike:%s", AerospikePort),
			},
		},
		OnServicesCreated: waitForService,
	}
)
