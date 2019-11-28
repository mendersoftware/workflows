package server

import (
	// "encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// WorkflowController container for end-points
type WorkflowController struct{}

type workflowRequest map[string]string

// StartWorkflow responds to POST /api/workflow/:name
func (h WorkflowController) StartWorkflow(c *gin.Context) {
	var name string = c.Param("name")
	if Workflows[name] == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Workflow not found: %s", name),
		})
		return
	}

	var inputParameters workflowRequest
	if err := c.BindJSON(&inputParameters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to parse the input parameters",
		})
		return
	}

	var workflow = Workflows[name]
	if err := workflow.Launch(inputParameters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to launch the workflow, please try again",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":    workflow.Name,
		"success": true,
	})
}
