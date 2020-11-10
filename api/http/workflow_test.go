// Copyright 2020 Northern.tech AS
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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	mocklib "github.com/stretchr/testify/mock"

	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
	"github.com/mendersoftware/workflows/store/mock"
)

func TestHealthCheck(t *testing.T) {
	testCases := []struct {
		Name string

		DataStoreErr error
		HTTPStatus   int
		HTTPBody     map[string]interface{}
	}{{
		Name:       "ok",
		HTTPStatus: http.StatusNoContent,
	}, {
		Name:         "error, MongoDB not reachable",
		DataStoreErr: errors.New("connection refused"),
		HTTPStatus:   http.StatusServiceUnavailable,
		HTTPBody: map[string]interface{}{
			"error": "error reaching MongoDB: connection refused",
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			dataStore := mock.NewDataStore()
			router := NewRouter(dataStore)
			dataStore.On("Ping", mocklib.Anything).
				Return(tc.DataStoreErr)

			req, err := http.NewRequest(http.MethodGet, "http://localhost"+APIURLHealth, nil)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, tc.HTTPStatus, w.Code)
			if tc.HTTPStatus >= 400 {
				var bodyJSON map[string]interface{}
				decoder := json.NewDecoder(w.Body)
				err := decoder.Decode(&bodyJSON)
				if assert.NoError(t, err) {
					assert.Equal(t, tc.HTTPBody, bodyJSON)
				}
			} else {
				assert.Nil(t, w.Body.Bytes())
			}
		})
	}
}

func TestWorkflowNotFound(t *testing.T) {
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	dataStore.On("InsertJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		mocklib.AnythingOfType("*model.Job"),
	).Return(nil, store.ErrWorkflowNotFound)

	payload := `{
      "key": "value"
	}`

	const workflowName = "test"
	url := strings.Replace(APIURLWorkflow, ":name", workflowName, 1)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	dataStore.AssertExpectations(t)
}

func TestWorkflowFoundButMissingParameters(t *testing.T) {
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	dataStore.On("InsertJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		mocklib.AnythingOfType("*model.Job"),
	).Return(nil, errors.Errorf(model.ErrMsgMissingParamF, []string{"param1", "param2", "param3"}))

	payload := `{
      "key": "value"
	}`

	const workflowName = "test"
	url := strings.Replace(APIURLWorkflow, ":name", workflowName, 1)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
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

	dataStore.AssertExpectations(t)
}

func TestWorkflowFoundAndStartedWithInvalidParameters(t *testing.T) {
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	payload := ``

	const workflowName = "test"
	url := strings.Replace(APIURLWorkflow, ":name", workflowName, 1)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
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

	dataStore.AssertExpectations(t)
}

func TestWorkflowFoundAndStartedWithParameters(t *testing.T) {
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	mockedJob := &model.Job{
		ID:           "1234567890",
		WorkflowName: "test",
	}

	dataStore.On("InsertJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		mocklib.MatchedBy(
			func(job *model.Job) bool {
				assert.Len(t, job.InputParameters, 1)
				assert.Equal(t, "key", job.InputParameters[0].Name)
				assert.Equal(t, "value", job.InputParameters[0].Value)

				return true
			}),
	).Return(mockedJob, nil)

	dataStore.On("GetJobByNameAndID",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		mockedJob.WorkflowName,
		mockedJob.ID,
	).Return(mockedJob, nil)

	payload := `{
      "key": "value"
	}`

	url := strings.Replace(APIURLWorkflow, ":name", mockedJob.WorkflowName, 1)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]string
	body := w.Body.Bytes()
	err = json.Unmarshal(body, &response)
	value, ok := response["id"]
	assert.True(t, ok)

	w = httptest.NewRecorder()
	url = strings.Replace(strings.Replace(APIURLWorkflowID, ":name", mockedJob.WorkflowName, 1), ":id", value, 1)
	req, _ = http.NewRequest(http.MethodGet, url, nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var job model.Job
	body = w.Body.Bytes()
	err = json.Unmarshal(body, &job)

	assert.Nil(t, err)
	assert.Equal(t, mockedJob.WorkflowName, job.WorkflowName)
	assert.Equal(t, 0, job.Status)

	dataStore.AssertExpectations(t)
}

func TestWorkflowFoundAndStartedWithNonStringParameter(t *testing.T) {
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	mockedJob := &model.Job{
		ID:           "1234567890",
		WorkflowName: "test",
	}

	dataStore.On("InsertJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		mocklib.MatchedBy(
			func(job *model.Job) bool {
				assert.Len(t, job.InputParameters, 1)
				assert.Equal(t, "key", job.InputParameters[0].Name)
				assert.Equal(t, "2", job.InputParameters[0].Value)

				return true
			}),
	).Return(mockedJob, nil)

	dataStore.On("GetJobByNameAndID",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		mockedJob.WorkflowName,
		mockedJob.ID,
	).Return(mockedJob, nil)

	payload := `{
      "key": "2"
	}`

	url := strings.Replace(APIURLWorkflow, ":name", mockedJob.WorkflowName, 1)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]string
	body := w.Body.Bytes()
	err = json.Unmarshal(body, &response)
	value, ok := response["id"]
	assert.True(t, ok)

	w = httptest.NewRecorder()
	url = strings.Replace(strings.Replace(APIURLWorkflowID, ":name", mockedJob.WorkflowName, 1), ":id", value, 1)
	req, _ = http.NewRequest(http.MethodGet, url, nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var job model.Job
	body = w.Body.Bytes()
	err = json.Unmarshal(body, &job)

	assert.Nil(t, err)
	assert.Equal(t, mockedJob.WorkflowName, job.WorkflowName)
	assert.Equal(t, 0, job.Status)

	dataStore.AssertExpectations(t)
}

func TestWorkflowFoundAndStartedWithListOfStringsParameter(t *testing.T) {
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	mockedJob := &model.Job{
		ID:           "1234567890",
		WorkflowName: "test",
	}

	dataStore.On("InsertJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		mocklib.MatchedBy(
			func(job *model.Job) bool {
				assert.Len(t, job.InputParameters, 1)
				assert.Equal(t, "key", job.InputParameters[0].Name)
				assert.Equal(t, "2,3,4", job.InputParameters[0].Value)

				return true
			}),
	).Return(mockedJob, nil)

	dataStore.On("GetJobByNameAndID",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		mockedJob.WorkflowName,
		mockedJob.ID,
	).Return(mockedJob, nil)

	payload := `{
      "key": ["2", 3, 4.0]
	}`

	url := strings.Replace(APIURLWorkflow, ":name", mockedJob.WorkflowName, 1)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]string
	body := w.Body.Bytes()
	err = json.Unmarshal(body, &response)
	value, ok := response["id"]
	assert.True(t, ok)

	w = httptest.NewRecorder()
	url = strings.Replace(strings.Replace(APIURLWorkflowID, ":name", mockedJob.WorkflowName, 1), ":id", value, 1)
	req, _ = http.NewRequest(http.MethodGet, url, nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var job model.Job
	body = w.Body.Bytes()
	err = json.Unmarshal(body, &job)

	assert.Nil(t, err)
	assert.Equal(t, mockedJob.WorkflowName, job.WorkflowName)
	assert.Equal(t, 0, job.Status)

	dataStore.AssertExpectations(t)
}

func TestWorkflowByNameAndIDNotFound(t *testing.T) {
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	const (
		workflowName = "test"
		jobID        = "dummy"
	)

	dataStore.On("GetJobByNameAndID",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		workflowName,
		jobID,
	).Return(nil, nil)

	url := strings.Replace(strings.Replace(APIURLWorkflowID, ":name", workflowName, 1), ":id", jobID, 1)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	dataStore.AssertExpectations(t)
}

func TestRegisterWorkflowBadRequestIncompletePayload(t *testing.T) {
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	payload := ``
	req, err := http.NewRequest(http.MethodPost, APIURLWorkflows, strings.NewReader(payload))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
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
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	payload := `{}`
	req, err := http.NewRequest(http.MethodPost, APIURLWorkflows, strings.NewReader(payload))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
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
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	dataStore.On("InsertWorkflows",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		mocklib.MatchedBy(
			func(workflows []model.Workflow) bool {
				assert.Len(t, workflows, 1)

				workflow := workflows[0]
				assert.Equal(t, "test_workflow", workflow.Name)
				assert.Equal(t, "Test workflow", workflow.Description)
				assert.Equal(t, 4, workflow.Version)
				assert.Equal(t, 1, workflow.SchemaVersion)

				assert.Len(t, workflow.Tasks, 1)

				task := workflow.Tasks[0]
				assert.Equal(t, "test_http_call", task.Name)
				assert.Equal(t, model.TaskTypeHTTP, task.Type)
				assert.Equal(t, "http://localhost:8000", task.HTTP.URI)

				return true
			}),
	).Return(1, nil)

	payload := `{
		"name": "test_workflow",
		"description": "Test workflow",
		"version": 4,
		"tasks": [
			{
				"name": "test_http_call",
				"type": "http",
				"http": {
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

	req, err := http.NewRequest(http.MethodPost, APIURLWorkflows, strings.NewReader(payload))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestGetWorkflows(t *testing.T) {
	task := model.Task{
		Name: "test_http_call",
		Type: model.TaskTypeHTTP,
		HTTP: &model.HTTPTask{
			URI:               "http://localhost:8000",
			Method:            http.MethodPost,
			Body:              "{\"device_id\": \"${workflow.input.device_id}\"}",
			ConnectionTimeOut: 1000,
			ReadTimeOut:       1000,
		},
	}
	workflow := model.Workflow{
		Name:        "test_workflow",
		Description: "Test workflow",
		Version:     4,
		Tasks:       []model.Task{task},
		InputParameters: []string{
			"device_id",
		},
		SchemaVersion: 1,
	}

	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	dataStore.On("GetWorkflows",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
	).Return([]model.Workflow{workflow}, nil)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, APIURLWorkflows, nil)
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
