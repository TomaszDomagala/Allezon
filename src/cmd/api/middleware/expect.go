package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"

	"github.com/TomaszDomagala/Allezon/src/pkg/dto"
)

const (
	userProfilesFullPath = "/user_profiles/:cookie"
	aggregatesFullPath   = "/aggregates"
)

// expectationValidatorEndpoints is a list of endpoints that should be validated.
var expectationValidatorEndpoints = []string{userProfilesFullPath, aggregatesFullPath}

// ExpectationValidator is a middleware that validates the request body against the response body.
// For some endpoints, the testing platform sends the expected response body in the request body. This middleware
// validates that the response body is the same as the request body and logs an error if it's not.
func ExpectationValidator(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !slices.Contains(expectationValidatorEndpoints, c.FullPath()) {
			c.Next()
			return
		}

		var err error
		var responseCopy, requestCopy bytes.Buffer

		// Hijack the response writer to copy the response body.
		c.Writer = newMultiResponseWriter(c.Writer, &responseCopy)

		// Read the request body and copy it to the requestCopy.
		c.Request.Body, err = copyRequestBody(c.Request.Body, &requestCopy, logger)
		if err != nil {
			logger.Error("error reading request body in a expectation validator", zap.Error(err), zap.String("endpoint", c.FullPath()))
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Next()

		loggerWith := logger.With(zap.String("endpoint", c.FullPath()), zap.String("query", c.Request.URL.RawQuery))

		checkExpectation(c.FullPath(), loggerWith, requestCopy, responseCopy)
	}
}

func checkExpectation(fullPath string, logger *zap.Logger, requestCopy, responseCopy bytes.Buffer) {
	var expected, actual any
	var err error

	switch fullPath {
	case userProfilesFullPath:
		var expectedUserProfile, actualUserProfile dto.UserProfileDTO
		if err = json.Unmarshal(requestCopy.Bytes(), &expectedUserProfile); err != nil {
			logger.Error("error unmarshalling expected user profile response body", zap.Error(err), zap.String("requestBody", requestCopy.String()))
			return
		}
		if err = json.Unmarshal(responseCopy.Bytes(), &actualUserProfile); err != nil {
			logger.Error("error unmarshalling actual user profile response body", zap.Error(err), zap.String("responseBody", responseCopy.String()))
			return
		}
		expected = expectedUserProfile
		actual = actualUserProfile
	case aggregatesFullPath:
		var expectedAggregate, actualAggregate dto.AggregatesDTO
		if err = json.Unmarshal(requestCopy.Bytes(), &expectedAggregate); err != nil {
			logger.Error("error unmarshalling expected aggregate response body", zap.Error(err), zap.String("requestBody", requestCopy.String()))
			return
		}
		if err = json.Unmarshal(responseCopy.Bytes(), &actualAggregate); err != nil {
			logger.Error("error unmarshalling actual aggregate response body", zap.Error(err), zap.String("responseBody", responseCopy.String()))
			return
		}
		expected = expectedAggregate
		actual = actualAggregate
	default:
		logger.Error("unexpected endpoint in expectation validator")
		return
	}

	if diff := cmp.Diff(expected, actual); diff != "" {
		logger.Error("response body does not match the expected response body",
			zap.String("difference", diff),
			zap.String("expected", requestCopy.String()),
			zap.String("actual", responseCopy.String()),
		)
	}
}

// multiResponseWriter is a wrapper around gin.ResponseWriter that writes to multiple writers.
type multiResponseWriter struct {
	gin.ResponseWriter
	// writer is the underlying writer that writes to the original gin.ResponseWriter and the copy.
	writer io.Writer
}

// newMultiResponseWriter creates a new gin.ResponseWriter that writes to the original gin.ResponseWriter and the copy.
func newMultiResponseWriter(original gin.ResponseWriter, copy io.Writer) gin.ResponseWriter {
	return &multiResponseWriter{
		ResponseWriter: original,
		writer:         io.MultiWriter(original, copy),
	}
}

func (w *multiResponseWriter) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

// copyRequestBody copies the body of a request and returns a new io.ReadCloser with the same content.
func copyRequestBody(body io.ReadCloser, copy io.Writer, logger *zap.Logger) (io.ReadCloser, error) {
	defer func() {
		if err := body.Close(); err != nil {
			logger.Error("error closing request body", zap.Error(err))
		}
	}()

	var buf bytes.Buffer
	writer := io.MultiWriter(&buf, copy)
	_, err := io.Copy(writer, body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return io.NopCloser(&buf), nil
}
