// Copyright 2019 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
)

// WorkflowController container for end-points
type WorkflowController struct {
	dataStore store.DataStoreInterface
}

// NewWorkflowController returns a new StatusController
func NewWorkflowController(dataStore store.DataStoreInterface) *WorkflowController {
	return &WorkflowController{
		dataStore: dataStore,
	}
}

// StartWorkflow responds to POST /api/workflow/:name
func (h WorkflowController) StartWorkflow(c *gin.Context) {
	var name string = c.Param("name")
	if Workflows[name] == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Workflow not found: %s", name),
		})
		return
	}

	var inputParameters map[string]string
	if err := c.BindJSON(&inputParameters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to parse the input parameters",
		})
		return
	}

	var workflow = Workflows[name]
	var missing []string
	for _, key := range workflow.InputParameters {
		if inputParameters[key] == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Missing input parameters: %s", strings.Join(missing, ", ")),
		})
		return
	}

	var jobInputParameters []model.InputParameter
	for key, value := range inputParameters {
		jobInputParameters = append(jobInputParameters, model.InputParameter{
			Name:  key,
			Value: value,
		})
	}

	ctx := context.Background()
	job := &model.Job{
		WorkflowName:    workflow.Name,
		InputParameters: jobInputParameters,
	}

	job, err := h.dataStore.InsertJob(ctx, job)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to launch the workflow, please try again",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      job.ID,
		"name":    workflow.Name,
		"success": true,
	})
}

// GetWorkflowByNameAndID responds to GET /api/workflow/:name/:id
func (h WorkflowController) GetWorkflowByNameAndID(c *gin.Context) {
	var name string = c.Param("name")
	var id string = c.Param("id")

	ctx := context.Background()
	jobStatus, err := h.dataStore.GetJobStatusByNameAndID(ctx, name, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "not found",
		})
		return
	}

	c.JSON(http.StatusOK, jobStatus)
}
