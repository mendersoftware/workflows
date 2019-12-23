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

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
)

// WorkflowController container for end-points
type WorkflowController struct {
	// dataStore provides an interface to the database
	dataStore store.DataStore
}

// NewWorkflowController returns a new StatusController
func NewWorkflowController(dataStore store.DataStore) *WorkflowController {

	return &WorkflowController{
		dataStore: dataStore,
	}
}

// RegisterWorkflow responds to POST /api/v1/metadata/workflows
func (h WorkflowController) RegisterWorkflow(c *gin.Context) {
	var workflow model.Workflow
	rawData, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Bad request",
		})
		return
	}
	if err = json.Unmarshal(rawData, &workflow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing JSON form: %s",
				err.Error()),
		})
		return
	}
	if workflow.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Workflow missing name",
		})
		return
	}
	_, err = h.dataStore.InsertWorkflows(workflow)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.Status(http.StatusCreated)
}

// GetWorkflows responds to GET /api/v1/metadata/workflows
func (h WorkflowController) GetWorkflows(c *gin.Context) {
	c.JSON(http.StatusOK, h.dataStore.GetWorkflows())
}

// StartWorkflow responds to POST /api/workflow/:name
func (h WorkflowController) StartWorkflow(c *gin.Context) {
	var name string = c.Param("name")
	var inputParameters map[string]string
	var jobInputParameters []model.InputParameter

	if err := c.BindJSON(&inputParameters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Unable to parse the input parameters: %s",
				err.Error()),
		})
		return
	}

	for key, value := range inputParameters {
		jobInputParameters = append(jobInputParameters, model.InputParameter{
			Name:  key,
			Value: value,
		})
	}

	ctx := context.Background()
	job := &model.Job{
		WorkflowName:    name,
		InputParameters: jobInputParameters,
	}

	job, err := h.dataStore.InsertJob(ctx, job)
	if err != nil {
		switch err {
		case store.ErrWorkflowNotFound:
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":   job.ID,
		"name": name,
	})
}

// GetWorkflowByNameAndID responds to GET /api/workflow/:name/:id
func (h WorkflowController) GetWorkflowByNameAndID(c *gin.Context) {
	var name string = c.Param("name")
	var id string = c.Param("id")

	ctx := context.Background()
	job, err := h.dataStore.GetJobByNameAndID(ctx, name, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	} else if job == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "not found",
		})
		return
	}

	job.StatusString = model.StatusToString(job.Status)
	c.JSON(http.StatusOK, job)
}
