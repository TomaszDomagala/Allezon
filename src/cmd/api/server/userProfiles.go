package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (s server) userProfilesHandler(c *gin.Context) {
	// Currently acts as echo server

	req, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/json", req)
}
