package server

import (
	"github.com/gin-gonic/gin"
)

// NewRouter returns the gin router
func NewRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	status := new(StatusController)
	router.GET("/status", status.Status)

	workflow := new(WorkflowController)
	router.POST("/api/workflow/:name", workflow.StartWorkflow)

	return router
}
