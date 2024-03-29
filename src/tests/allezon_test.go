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
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/container"
	"github.com/TomaszDomagala/Allezon/src/pkg/container/containerutils"
	"github.com/TomaszDomagala/Allezon/src/pkg/dto"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
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
	s.Require().NoErrorf(containerutils.BuildAPIImage(), "could not build api image")
	s.Require().NoErrorf(containerutils.BuildWorkerImage(), "could not build worker image")
	s.Require().NoErrorf(containerutils.BuildIDGetterImage(), "could not build idgetter image")
	s.logger.Info("finished building images")
}

func (s *AllezonIntegrationTestSuite) SetupTest() {
	s.env = container.NewEnvironment(s.T().Name(), s.logger, []*container.Service{
		containerutils.RedpandaService,
		containerutils.AerospikeService,
		containerutils.IDGetterService,
		containerutils.WorkerService,
		containerutils.APIService,
	}, nil)

	err := s.env.Run()
	if err != nil {
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

func minuteAlign(min time.Time) time.Time {
	return min.Add(-(time.Duration(min.Nanosecond()) + time.Second*time.Duration(min.Second()))) // Round to exactly a minute.
}

func (s *AllezonIntegrationTestSuite) TestSendUserTagsSingleCookie() {
	now, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
	s.Require().NoErrorf(err, "could not parse time")
	now = now.Add(time.Millisecond * 123)

	cookie := "cookie"

	newTag := func(timestamp time.Time, action string) dto.UserTagDTO {
		id := 1337
		return dto.UserTagDTO{
			Cookie:  cookie,
			Time:    timestamp.Format(dto.UserTagTimeLayout),
			Device:  "PC",
			Action:  action,
			Country: "PL",
			Origin:  "https://www.google.com/",
			ProductInfo: dto.ProductInfo{
				ProductID:  &id,
				BrandID:    "adidas",
				CategoryID: "shoes",
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
		{
			from:   now,
			to:     now.Add(2 * time.Hour),
			cookie: cookie,
			expected: dto.UserProfileDTO{
				Cookie: cookie,
				Views:  []dto.UserTagDTO{userTags[1], userTags[0]}, // DESC order
				Buys:   []dto.UserTagDTO{},
			},
		},
		{
			from:   now,
			to:     now.Add(2 * time.Hour).Add(1 * time.Millisecond),
			cookie: cookie,
			expected: dto.UserProfileDTO{
				Cookie: cookie,
				Views:  []dto.UserTagDTO{userTags[2], userTags[1], userTags[0]}, // DESC order
				Buys:   []dto.UserTagDTO{},
			},
		},
	}

	maNow := minuteAlign(now)
	aggregatesRequests := []struct {
		from, to   time.Time
		aggregates []types.Aggregate
		action     types.Action
		origin     *string
		categoryId *string
		brandId    *string

		expected dto.AggregatesDTO
	}{
		{
			from:       maNow,
			to:         maNow.Add(1 * time.Minute),
			action:     types.View,
			aggregates: []types.Aggregate{types.Sum, types.Count},
			expected: dto.AggregatesDTO{
				Columns: []string{"1m_bucket", "action", strings.ToLower(types.Sum.String()), strings.ToLower(types.Count.String())},
				Rows: [][]string{
					{maNow.Format(dto.TimeRangeSecPrecisionLayout), "VIEW", "100", "1"},
				},
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

	for index, profileReq := range profileRequests {
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
		s.Assert().NoErrorf(err, "could not decode response body of profile request %d", index)
		s.Assert().Equalf(profileReq.expected, profile, "profile request mismatch %d", index)
	}

	for index, aggReq := range aggregatesRequests {
		params := url.Values{}

		from := aggReq.from.Format(dto.TimeRangeSecPrecisionLayout)
		to := aggReq.to.Format(dto.TimeRangeSecPrecisionLayout)
		params.Add("time_range", fmt.Sprintf("%s_%s", from, to))

		params.Add("action", aggReq.action.String())
		for _, a := range aggReq.aggregates {
			params.Add("aggregates", a.String())
		}

		if aggReq.origin != nil {
			params.Add("origin", *aggReq.origin)
		}
		if aggReq.categoryId != nil {
			params.Add("category_id", *aggReq.categoryId)
		}
		if aggReq.brandId != nil {
			params.Add("brand_id", *aggReq.brandId)
		}

		reqUrl := address + "/aggregates/?" + params.Encode()
		req, err := http.NewRequest(http.MethodPost, reqUrl, nil)
		s.Require().NoErrorf(err, "could not create request")

		res, err := client.Do(req)
		s.Assert().NoErrorf(err, "could not send request")
		if res.StatusCode != http.StatusOK {
			body, err := io.ReadAll(res.Body)
			s.Assert().Equalf(http.StatusOK, res.StatusCode, `unexpected status code, status: %s, body "%s".\nRequest Params: %s.`, res.Status, body, spew.Sprintln(params), params.Encode())
			s.Assert().NoErrorf(err, "error file reading request body")
		}

		var aggr dto.AggregatesDTO
		err = json.NewDecoder(res.Body).Decode(&aggr)
		s.Assert().NoErrorf(err, "could not decode response body of profile request %d", index)
		s.Assert().Equalf(aggReq.expected, aggr, "aggregates response mismatch %d", index)
	}
}
