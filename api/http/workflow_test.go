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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mendersoftware/workflows/model"
	store "github.com/mendersoftware/workflows/store/mock"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowNotFound(t *testing.T) {
	dataStore := store.NewDataStoreMock()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()

	payload := `{
      "key": "value"
	}`

	req, _ := http.NewRequest("POST", "/api/v1/workflow/test", strings.NewReader(payload))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWorkflowFoundButMissingParameters(t *testing.T) {
	dataStore := store.NewDataStoreMock()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()
	workflow := model.Workflow{
		Name: "test",
		InputParameters: []string{
			"param1",
			"param2",
			"param3",
		},
	}
	_, err := dataStore.InsertWorkflows(workflow)
	assert.NoError(t, err)

	w = httptest.NewRecorder()
	payload := `{
      "key": "value"
	}`

	req, err := http.NewRequest("POST", "/api/v1/workflow/test", strings.NewReader(payload))
	assert.NoError(t, err)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	body := w.Body.Bytes()
	err = json.Unmarshal(body, &response)
	value, ok := response["error"]

	assert.Nil(t, err)
	assert.True(t, ok)

	expectedBody := gin.H{
		"error": "Missing input parameters: [param1 param2 param3]",
	}
	assert.Equal(t, expectedBody["error"], value)
}

func TestWorkflowFoundAndStartedWithInvalidParameters(t *testing.T) {
	dataStore := store.NewDataStoreMock()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()
	workflow := model.Workflow{
		Name: "test",
		InputParameters: []string{
			"key",
		},
	}
	_, err := dataStore.InsertWorkflows(workflow)
	assert.NoError(t, err)

	payload := ``

	req, _ := http.NewRequest("POST", "/api/v1/workflow/test", strings.NewReader(payload))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	body := w.Body.Bytes()
	err = json.Unmarshal(body, &response)
	value, ok := response["error"]

	assert.Nil(t, err)
	assert.True(t, ok)

	expectedBody := gin.H{
		"error": "Unable to parse the input parameters: EOF",
	}
	assert.Equal(t, expectedBody["error"], value)
}

func TestWorkflowFoundAndStartedWithParameters(t *testing.T) {
	dataStore := store.NewDataStoreMock()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()
	workflow := model.Workflow{
		Name: "test",
		InputParameters: []string{
			"key",
		},
	}
	_, err := dataStore.InsertWorkflows(workflow)
	assert.NoError(t, err)

	payload := `{
      "key": "value"
	}`

	req, _ := http.NewRequest("POST", "/api/v1/workflow/test", strings.NewReader(payload))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, 1, len(dataStore.Jobs))

	var response map[string]string
	body := w.Body.Bytes()
	err = json.Unmarshal(body, &response)
	value, ok := response["id"]
	assert.True(t, ok)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/workflow/test/%s", value), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var job model.Job
	body = w.Body.Bytes()
	err = json.Unmarshal(body, &job)

	assert.Nil(t, err)
	assert.Equal(t, "test", job.WorkflowName)
	assert.Equal(t, 0, job.Status)
}

func TestWorkflowByNameAndIDNotFound(t *testing.T) {
	dataStore := store.NewDataStoreMock()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/workflow/test/dummy", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRegisterWorkflowBadRequestIncompletePayload(t *testing.T) {
	dataStore := store.NewDataStoreMock()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()

	payload := ``
	req, err := http.NewRequest("POST", "/api/v1/metadata/workflows", strings.NewReader(payload))
	assert.NoError(t, err)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	body := w.Body.Bytes()
	err = json.Unmarshal(body, &response)
	value, ok := response["error"]

	assert.Nil(t, err)
	assert.True(t, ok)

	expectedBody := gin.H{
		"error": "Error parsing JSON form: unexpected end of JSON input",
	}
	assert.Equal(t, expectedBody["error"], value)
}

func TestRegisterWorkflowBadRequestMissingName(t *testing.T) {
	dataStore := store.NewDataStoreMock()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()

	payload := `{}`
	req, err := http.NewRequest("POST", "/api/v1/metadata/workflows", strings.NewReader(payload))
	assert.NoError(t, err)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	body := w.Body.Bytes()
	err = json.Unmarshal(body, &response)
	value, ok := response["error"]

	assert.Nil(t, err)
	assert.True(t, ok)

	expectedBody := gin.H{
		"error": "Workflow missing name",
	}
	assert.Equal(t, expectedBody["error"], value)
}

func TestRegisterWorkflow(t *testing.T) {
	dataStore := store.NewDataStoreMock()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()
	payload := `{
		"name": "test_workflow",
		"description": "Test workflow",
		"version": 4,
		"tasks": [
			{
				"name": "test_http_call",
				"type": "http",
				"taskdef": {
					"uri": "http://localhost:8000",
					"method": "POST",
					"body": "{\"device_id\": \"${workflow.input.device_id}\"}",
					"connectionTimeOut": 1000,
					"readTimeOut": 1000
				}
			}
		],
		"inputParameters": [
			"device_id"
		],
		"schemaVersion": 1
	}`

	req, err := http.NewRequest("POST", "/api/v1/metadata/workflows", strings.NewReader(payload))
	assert.NoError(t, err)
	router.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)

	workflow, err := dataStore.GetWorkflowByName("test_workflow")
	assert.NotNil(t, workflow)
	assert.Equal(t, "test_workflow", workflow.Name)
	assert.Equal(t, "Test workflow", workflow.Description)
	assert.Equal(t, 4, workflow.Version)
	assert.Equal(t, 1, workflow.SchemaVersion)
}

func TestGetWorkflows(t *testing.T) {
	taskdef := model.HTTPTask{
		URI:               "http://localhost:8000",
		Method:            "POST",
		Body:              "{\"device_id\": \"${workflow.input.device_id}\"}",
		ConnectionTimeOut: 1000,
		ReadTimeOut:       1000,
	}
	taskdefJSON, _ := json.Marshal(taskdef)
	workflow := model.Workflow{
		Name:        "test_workflow",
		Description: "Test workflow",
		Version:     4,
		Tasks: []model.Task{
			model.Task{
				Name:    "test_http_call",
				Type:    "http",
				Taskdef: taskdefJSON,
			},
		},
		InputParameters: []string{
			"device_id",
		},
		SchemaVersion: 1,
	}

	dataStore := store.NewDataStoreMock()
	dataStore.InsertWorkflows(workflow)
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/v1/metadata/workflows", nil)
	assert.NoError(t, err)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var workflows []model.Workflow
	body := w.Body.Bytes()
	err = json.Unmarshal(body, &workflows)

	assert.Nil(t, err)
	assert.Len(t, workflows, 1)
	assert.Equal(t, "test_workflow", workflows[0].Name)
	assert.Equal(t, "Test workflow", workflows[0].Description)
	assert.Equal(t, 4, workflows[0].Version)
	assert.Equal(t, 1, workflows[0].SchemaVersion)
}
