package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

// expectationValidatorEndpoints is a list of endpoints that should be validated.
var expectationValidatorEndpoints = []string{
	"/user_profiles/:cookie",
	"/aggregates",
}

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

		//opts := jsondiff.DefaultConsoleOptions()
		//diff, description := jsondiff.Compare(requestCopy.Bytes(), responseCopy.Bytes(), &opts)
		//if diff != jsondiff.FullMatch {
		//	logger.Error("response body does not match the expected response body",
		//		zap.String("endpoint", c.FullPath()),
		//		zap.String("difference", description),
		//		zap.String("expected", requestCopy.String()),
		//		zap.String("actual", responseCopy.String()),
		//	)
		//}

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
