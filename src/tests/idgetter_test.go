package tests

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	"github.com/TomaszDomagala/Allezon/src/pkg/container/containerutils"
	"github.com/TomaszDomagala/Allezon/src/pkg/idGetter"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type IDGetterIntegrationTestsSuite struct {
	suite.Suite
	logger *zap.Logger

	env *container.Environment
}

func TestIDGetterIntegration(t *testing.T) {
	suite.Run(t, new(IDGetterIntegrationTestsSuite))
}

func (s *IDGetterIntegrationTestsSuite) SetupSuite() {
	var err error

	s.logger, err = zap.NewDevelopment()
	s.Require().NoErrorf(err, "could not create logger")

	err = containerutils.BuildIDGetterImage()
	s.Require().NoErrorf(err, "could not build idgetter image")
}

func (s *IDGetterIntegrationTestsSuite) SetupTest() {
	s.env = container.NewEnvironment(s.T().Name(), s.logger, []*container.Service{
		containerutils.AerospikeService,
		containerutils.IDGetterService,
	}, nil)

	err := s.env.Run()
	if err != nil {
		s.Require().NoErrorf(err, "could not run environment")
	}
}

func (s *IDGetterIntegrationTestsSuite) TearDownTest() {
	err := s.env.Close()
	s.Require().NoErrorf(err, "could not close environment")
	s.env = nil
}

func (s *IDGetterIntegrationTestsSuite) getIDGetterURL() (string, error) {
	idgetterSrv := s.env.GetService("idgetter")
	if idgetterSrv == nil {
		return "", fmt.Errorf("could not get idgetter service")
	}

	hostport := idgetterSrv.ExposedHostPort()
	if hostport == "" {
		return "", fmt.Errorf("could not get host port for idgetter service")
	}

	return hostport, nil
}

func (s *IDGetterIntegrationTestsSuite) TestIDGetter() {
	url, err := s.getIDGetterURL()
	s.Require().NoErrorf(err, "could not get idgetter url")

	// Create a new client without caching.
	client := idGetter.NewPureClient(http.Client{Timeout: 5 * time.Second}, url)

	calls := []struct {
		category   string
		name       string
		expectedID int32
	}{
		{category: "food", name: "apple", expectedID: 1},
		{category: "food", name: "banana", expectedID: 2},
		{category: "food", name: "orange", expectedID: 3},
		{category: "food", name: "apple", expectedID: 1},
		{category: "food", name: "banana", expectedID: 2},
		{category: "food", name: "orange", expectedID: 3},
		{category: "transport", name: "car", expectedID: 1},
		{category: "transport", name: "bike", expectedID: 2},
		{category: "transport", name: "bus", expectedID: 3},
		{category: "transport", name: "car", expectedID: 1},
		{category: "transport", name: "bike", expectedID: 2},
		{category: "transport", name: "bus", expectedID: 3},
	}

	for _, call := range calls {
		id, err := client.GetID(call.category, call.name)
		s.Assert().NoErrorf(err, "could not get id for %s/%s", call.category, call.name)
		s.Assert().Equalf(call.expectedID, id, "unexpected id for %s/%s", call.category, call.name)
	}
}
