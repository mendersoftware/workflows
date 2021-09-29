// Copyright 2021 Northern.tech AS
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
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/workflows/app/worker"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
)

const (
	defaultTimeout = time.Second * 5
)

var (
	HeaderWorkflowMinVersion = "X-Workflows-Min-Version"
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

func (h WorkflowController) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	err := h.dataStore.Ping(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "error reaching MongoDB: " + err.Error(),
		})
		return
	}
	c.Status(http.StatusNoContent)
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
	_, err = h.dataStore.InsertWorkflows(c, workflow)
	if err != nil {
		httpStatus := http.StatusBadRequest
		if err == store.ErrWorkflowAlreadyExists {
			httpStatus = http.StatusConflict
		}
		c.JSON(httpStatus, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.Status(http.StatusCreated)
}

// GetWorkflows responds to GET /api/v1/metadata/workflows
func (h WorkflowController) GetWorkflows(c *gin.Context) {
	c.JSON(http.StatusOK, h.dataStore.GetWorkflows(c))
}

// StartWorkflow responds to POST /api/workflow/:name
func (h WorkflowController) StartWorkflow(c *gin.Context) {
	var name string = c.Param("name")
	var inputParameters map[string]interface{}
	var jobInputParameters []model.InputParameter

	l := log.FromContext(c.Request.Context())

	l.Infof("StartWorkflow starting workflow %s", name)
	if err := c.BindJSON(&inputParameters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Unable to parse the input parameters: %s",
				err.Error()),
		})
		return
	}

	workflowVersion := ""
	if values, _ := c.Request.Header[HeaderWorkflowMinVersion]; len(values) > 0 {
		workflowVersion = values[0]
	}
	for key, value := range inputParameters {
		valueSlice, ok := value.([]interface{})
		if ok {
			values := make([]string, 0, 10)
			for _, value := range valueSlice {
				valueString, err := worker.ConvertAnythingToString(value)
				if err == nil {
					values = append(values, valueString)
				}
			}
			jobInputParameters = append(jobInputParameters, model.InputParameter{
				Name:  key,
				Value: strings.Join(values, ","),
				Raw:   value,
			})
		} else {
			valueString, err := worker.ConvertAnythingToString(value)
			if err == nil {
				jobInputParameters = append(jobInputParameters, model.InputParameter{
					Name:  key,
					Value: valueString,
					Raw:   value,
				})
			}
		}
	}

	job := &model.Job{
		WorkflowName:    name,
		WorkflowVersion: workflowVersion,
		InputParameters: jobInputParameters,
	}

	job, err := h.dataStore.InsertJob(c, job)
	l.Infof("StartWorkflow db.InsertJob returned %v,%v", job, err)
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
	l.Infof("StartWorkflow starting workflow %s : StatusCreated", name)
}

// GetWorkflowByNameAndID responds to GET /api/workflow/:name/:id
func (h WorkflowController) GetWorkflowByNameAndID(c *gin.Context) {
	var name string = c.Param("name")
	var id string = c.Param("id")

	job, err := h.dataStore.GetJobByNameAndID(c, name, id)
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
