package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
