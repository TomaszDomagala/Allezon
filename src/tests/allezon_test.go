package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	"github.com/TomaszDomagala/Allezon/src/pkg/container/containerutils"
	"github.com/TomaszDomagala/Allezon/src/pkg/dto"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
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

func (s *AllezonIntegrationTestSuite) dumpLogs() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get working directory: %w", err)
	}
	logsPath := path.Join(wd, "logs")

	var errs []error

	for _, service := range s.env.Services {
		stdout, stderr, err := s.env.GetLogs(service.Name)
		if err != nil {
			errs = append(errs, fmt.Errorf("could not get logs for service %s: %w", service.Name, err))
			// don't exit, we want to write files anyway
		}

		if err = os.WriteFile(path.Join(logsPath, fmt.Sprintf("%s.stdout.log", service.Name)), []byte(stdout), 0644); err != nil {
			errs = append(errs, fmt.Errorf("could not write stdout logs for service %s: %w", service.Name, err))
		}
		if err = os.WriteFile(path.Join(logsPath, fmt.Sprintf("%s.stderr.log", service.Name)), []byte(stderr), 0644); err != nil {
			errs = append(errs, fmt.Errorf("could not write stderr logs for service %s: %w", service.Name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while dumping logs: %v", errs)
	}
	return nil
}

func (s *AllezonIntegrationTestSuite) TearDownTest() {
	err := s.dumpLogs()
	s.Assert().NoErrorf(err, "could not dump logs")

	err = s.env.Close()
	s.Assert().NoErrorf(err, "could not close environment")
	s.env = nil
}

func (s *AllezonIntegrationTestSuite) TestSendUserTagsSingleCookie() {
	now, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
	s.Require().NoErrorf(err, "could not parse time")
	now = now.Add(time.Millisecond * 123)

	cookie := "cookie"

	newTag := func(timestamp time.Time, action string) dto.UserTagDTO {
		return dto.UserTagDTO{
			Cookie:  cookie,
			Time:    timestamp.Format(dto.UserTagTimeLayout),
			Device:  "PC",
			Action:  action,
			Country: "PL",
			Origin:  "https://www.google.com/",
			ProductInfo: dto.ProductInfo{
				ProductId:  1337,
				BrandId:    "adidas",
				CategoryId: "shoes",
				Price:      100,
			},
		}
	}

	userTags := []dto.UserTagDTO{
		newTag(now, "VIEW"),
		newTag(now.Add(1*time.Hour), "VIEW"),
		newTag(now.Add(2*time.Hour), "VIEW"),
	}

	profileRequests := []struct {
		cookie   string
		from, to time.Time
		limit    int

		expected dto.UserProfileDTO
	}{
		{
			from:   now,
			to:     now.Add(1 * time.Hour),
			cookie: cookie,

			expected: dto.UserProfileDTO{
				Cookie: cookie,
				Views:  []dto.UserTagDTO{userTags[0]},
				Buys:   []dto.UserTagDTO{},
			},
		},
	}

	client := http.Client{Timeout: 5 * time.Second}

	hostport := s.env.GetService("api").ExposedHostPort()
	s.Require().NotEmptyf(hostport, "could not get hostport of api service")
	address := "http://" + hostport

	for _, tag := range userTags {
		var body bytes.Buffer
		err := json.NewEncoder(&body).Encode(tag)
		s.Require().NoErrorf(err, "could not encode tag to json")

		res, err := client.Post(address+"/user_tags", "application/json", &body)
		s.Assert().NoErrorf(err, "could not send request")
		s.Assert().Equalf(http.StatusNoContent, res.StatusCode, "unexpected status code")
	}

	const workersWaitTime = 10 * time.Second
	s.logger.Info("Waiting for workers to process tags", zap.Duration("time", workersWaitTime))
	time.Sleep(workersWaitTime)

	for _, profileReq := range profileRequests {
		params := url.Values{}

		from := profileReq.from.Format(dto.TimeRangeMilliPrecisionLayout)
		to := profileReq.to.Format(dto.TimeRangeMilliPrecisionLayout)
		params.Add("time_range", fmt.Sprintf("%s_%s", from, to))

		if profileReq.limit > 0 {
			params.Add("limit", strconv.Itoa(profileReq.limit))
		}

		reqUrl := address + "/user_profiles/" + profileReq.cookie + "?" + params.Encode()
		req, err := http.NewRequest(http.MethodPost, reqUrl, nil)
		s.Require().NoErrorf(err, "could not create request")

		res, err := client.Do(req)
		s.Assert().NoErrorf(err, "could not send request")
		if res.StatusCode != http.StatusOK {
			body, err := io.ReadAll(res.Body)
			s.Assert().Equalf(http.StatusOK, res.StatusCode, `unexpected status code, status: %s, body "%s".\nRequest Params: %s.`, res.Status, body, spew.Sprintln(params), params.Encode())
			s.Assert().NoErrorf(err, "error file reading request body")
		}

		var profile dto.UserProfileDTO
		err = json.NewDecoder(res.Body).Decode(&profile)
		s.Assert().NoErrorf(err, "could not decode response body")
		s.Assert().Equalf(profileReq.expected, profile, "unexpected profile")
	}
}
