package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (s server) aggregatesHandler(c *gin.Context) {
	// Currently acts as echo server

	req, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.logger.Debug("aggregatesHandler", zap.ByteString("req", req))

	c.Data(http.StatusOK, "application/json", req)
}
