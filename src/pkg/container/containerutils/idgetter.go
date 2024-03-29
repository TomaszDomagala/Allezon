package containerutils

import (
	"fmt"

	"github.com/ory/dockertest/v3"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
)

var (
	idgetterImageName = dockerRepo + "/idgetter"
	idgetterImageTag  = "latest"

	IDGetterPort       = "8080"
	IDGetterDockerPort = IDGetterPort + "/tcp"

	// IDGetterService is a service with idgetter container. It depends on the aerospike service and WILL panic if it is not available.
	IDGetterService = &container.Service{
		Name: "idgetter",
		Options: &dockertest.RunOptions{
			Repository:   idgetterImageName,
			Tag:          idgetterImageTag,
			Hostname:     "idgetter",
			ExposedPorts: []string{IDGetterDockerPort},
			Env: []string{
				fmt.Sprintf("PORT=%s", IDGetterPort),
				fmt.Sprintf("DB_ADDRESSES=aerospike:%s", AerospikePort),
				"LOG_LEVEL=DEBUG",
			},
		},
		OnServicesCreated: waitForService,
	}
)
