package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (s server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
