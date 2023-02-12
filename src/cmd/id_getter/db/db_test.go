package db

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	"github.com/TomaszDomagala/Allezon/src/pkg/container/containerutils"
)

// DBSuite is a suite for db integration tests.
type DBSuite struct {
	suite.Suite
	logger *zap.Logger

	// env is created for each test case.
	env *container.Environment
}

// TestIDGetterDBSuite is an entry point for running all tests in this package.
func TestIDGetterDBSuite(t *testing.T) {
	suite.Run(t, new(DBSuite))
}

func (s *DBSuite) SetupSuite() {
	var err error

	s.logger, err = zap.NewDevelopment()
	s.Require().NoErrorf(err, "could not create logger")
}

func (s *DBSuite) SetupTest() {
	s.env = container.NewEnvironment(s.T().Name(), s.logger, []*container.Service{containerutils.AerospikeService}, nil)
	err := s.env.Run()
	s.Require().NoErrorf(err, "could not run environment")
}

func (s *DBSuite) TearDownTest() {
	err := s.env.Close()
	s.Require().NoErrorf(err, "could not close environment")
	s.env = nil
}

func (s *DBSuite) newClient() Client {
	hostPort := s.env.GetService("aerospike").ExposedHostPort()
	cl, err := NewClientFromAddresses(hostPort)
	s.Require().NoErrorf(err, "failed to create client")
	return cl
}

func (s *DBSuite) TestNewClient() {
	cl := s.newClient()
	runtime.KeepAlive(cl)
}

func (s *DBSuite) Test_Ids() {
	c := s.newClient()

	const name = "foobar"
	t := "foo"

	l, err := c.AppendElement(name, t)
	s.Require().NoErrorf(err, "failed to create record")
	s.Require().Equal(1, l, "list length mismatch")

	got, err := c.GetElements(name)
	s.Require().NoErrorf(err, "failed to get record")
	s.Require().Equal([]string{t}, got)

	t2 := "bar"

	l2, err := c.AppendElement(name, t2)
	s.Require().NoErrorf(err, "failed to update record")
	s.Require().Equal(2, l2, "list length mismatch")

	updated, err := c.GetElements(name)
	s.Require().NoErrorf(err, "failed to get record")
	s.Require().Equal([]string{t, t2}, updated)
}

func (s *DBSuite) Test_Ids_ErrorOnDuplicate() {
	c := s.newClient()

	const name = "foobar"
	t := "foo"

	_, err := c.AppendElement(name, t)
	s.Require().NoErrorf(err, "failed to create record")

	got, err := c.GetElements(name)
	s.Require().NoErrorf(err, "failed to get record")
	s.Require().Equal([]string{t}, got)

	t2 := "foo"

	_, err = c.AppendElement(name, t2)
	s.Require().ErrorIs(err, ElementExists, "no error on element exists")
}
