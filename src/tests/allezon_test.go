package tests

import (
	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	"github.com/TomaszDomagala/Allezon/src/pkg/container/containerutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"testing"
)

type AllezonIntegrationTestSuite struct {
	suite.Suite
	logger *zap.Logger

	env *container.Environment
}

func TestAllezonIntegration(t *testing.T) {
	suite.Run(t, new(AllezonIntegrationTestSuite))
}

func (s *AllezonIntegrationTestSuite) SetupSuite() {
	var err error

	s.logger, err = zap.NewDevelopment()
	s.Require().NoErrorf(err, "could not create logger")

	// Build images before running tests.
	s.logger.Info("building images")
	s.Require().NoErrorf(containerutils.BuildApiImage(), "could not build api image")
	s.Require().NoErrorf(containerutils.BuildWorkerImage(), "could not build aerospike image")
	s.Require().NoErrorf(containerutils.BuildIDGetterImage(), "could not build idgetter image")
	s.logger.Info("finished building images")
}

func (s *AllezonIntegrationTestSuite) SetupTest() {
	s.env = container.NewEnvironment(s.T().Name(), s.logger, []*container.Service{
		containerutils.RedpandaService,
		containerutils.AerospikeService,
		containerutils.IDGetterService,
		containerutils.WorkerService,
		containerutils.ApiService,
	}, nil)

	err := s.env.Run()
	if err != nil {
		errClose := s.env.Close()
		s.Assert().NoErrorf(errClose, "could not close environment after error")
		s.Require().NoErrorf(err, "could not run environment")
	}
}

func (s *AllezonIntegrationTestSuite) TearDownTest() {
	err := s.env.Close()
	s.Require().NoErrorf(err, "could not close environment")
	s.env = nil
}

func (s *AllezonIntegrationTestSuite) TestFoo() {
	s.T().Log("foo")
}
