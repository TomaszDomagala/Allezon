package containerutils

import (
	"fmt"

	"github.com/ory/dockertest/v3"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
)

var (
	apiImageName = "tomaszdomagala/allezon-api"
	apiImageTag  = "latest"

	ApiPort       = "8080"
	ApiDockerPort = ApiPort + "/tcp"

	// ApiService is a service that runs the API. It depends on the redpanda and aerospike services and WILL panic if
	// they are not available.
	// API also depends on the idgetter service, but it will not panic if it is not available. However, it will not be
	// able to handle /user_profiles/:cookie and /aggregates endpoints, as they require the idgetter service.
	ApiService = &container.Service{
		Name: "api",
		Options: &dockertest.RunOptions{
			Repository:   apiImageName,
			Tag:          apiImageTag,
			Hostname:     "api",
			ExposedPorts: []string{ApiDockerPort},
			Env: []string{
				fmt.Sprintf("PORT=%s", ApiPort),
				fmt.Sprintf("KAFKA_ADDRESSES=redpanda:%s", RedpandaPort),
				fmt.Sprintf("DB_ADDRESSES=aerospike:%s", AerospikePort),
				fmt.Sprintf("ID_GETTER_ADDRESS=idgetter:%s", IDGetterPort),
			},
		},
		OnServicesCreated: waitForService,
	}
)
