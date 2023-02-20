package containerutils

import (
	"fmt"

	"github.com/ory/dockertest/v3"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
)

var (
	apiImageName = dockerRepo + "/api"
	apiImageTag  = "latest"

	APIPort       = "8080"
	APIDockerPort = APIPort + "/tcp"

	// APIService is a service that runs the API. It depends on the redpanda and aerospike services and WILL panic if
	// they are not available.
	// API also depends on the idgetter service, but it will not panic if it is not available. However, it will not be
	// able to handle /user_profiles/:cookie and /aggregates endpoints, as they require the idgetter service.
	APIService = &container.Service{
		Name: "api",
		Options: &dockertest.RunOptions{
			Repository:   apiImageName,
			Tag:          apiImageTag,
			Hostname:     "api",
			ExposedPorts: []string{APIDockerPort},
			Env: []string{
				fmt.Sprintf("PORT=%s", APIPort),
				fmt.Sprintf("KAFKA_ADDRESSES=redpanda:%s", RedpandaPort),
				fmt.Sprintf("DB_PROFILES_ADDRESSES=aerospike:%s", AerospikePort),
				fmt.Sprintf("DB_AGGREGATES_ADDRESSES=aerospike:%s", AerospikePort),
				fmt.Sprintf("ID_GETTER_ADDRESS=idgetter:%s", IDGetterPort),
				"LOG_LEVEL=DEBUG",
			},
		},
		OnServicesCreated: waitForService,
	}
)
