package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StatusController contains status-related end-points
type StatusController struct{}

// Status responds to GET /status
func (h StatusController) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
