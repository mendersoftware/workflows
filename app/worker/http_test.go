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

package worker

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	mocklib "github.com/stretchr/testify/mock"

	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store/mock"
)

func TestProcessJobHTTP(t *testing.T) {
	ctx := context.Background()
	dataStore := mock.NewDataStore()

	requestBody := "BODY"

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeHTTP,
				HTTP: &model.HTTPTask{
					URI:    "http://localhost",
					Method: "GET",
					Headers: map[string]string{
						"X-Header": "Value",
					},
				},
			},
		},
	}

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}

	dataStore.On("GetWorkflowByName",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		workflow.Name,
	).Return(workflow, nil)

	dataStore.On("AquireJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
	).Return(job, nil)

	dataStore.On("UpdateJobStatus",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		model.StatusDone,
	).Return(nil)

	dataStore.On("UpdateJobAddResult",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		mocklib.MatchedBy(
			func(taskResult *model.TaskResult) bool {
				assert.True(t, taskResult.Success)
				assert.Equal(t, workflow.Tasks[0].HTTP.URI, taskResult.HTTPRequest.URI)
				assert.Equal(t, workflow.Tasks[0].HTTP.Method, taskResult.HTTPRequest.Method)
				assert.Equal(t, []string{
					"X-Header: Value",
				}, taskResult.HTTPRequest.Headers)
				assert.Equal(t, http.StatusOK, taskResult.HTTPResponse.StatusCode)
				assert.Equal(t, requestBody, taskResult.HTTPResponse.Body)

				return true
			}),
	).Return(nil)

	makeHTTPRequestOriginal := makeHTTPRequest
	makeHTTPRequest = func(req *http.Request, timeout time.Duration) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(requestBody)),
		}
		return resp, nil
	}
	err := processJob(ctx, job, dataStore)
	makeHTTPRequest = makeHTTPRequestOriginal

	assert.Nil(t, err)

	dataStore.AssertExpectations(t)
}

func TestProcessJobHTTPValidStatusCode(t *testing.T) {
	ctx := context.Background()
	dataStore := mock.NewDataStore()

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeHTTP,
				HTTP: &model.HTTPTask{
					URI:    "http://localhost",
					Method: "GET",
					Headers: map[string]string{
						"X-Header": "Value",
					},
					StatusCodes: []int{
						200,
					},
				},
			},
		},
	}

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}

	requestBody := "BODY"

	dataStore.On("GetWorkflowByName",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		workflow.Name,
	).Return(workflow, nil)

	dataStore.On("AquireJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
	).Return(job, nil)

	dataStore.On("UpdateJobStatus",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		model.StatusDone,
	).Return(nil)

	dataStore.On("UpdateJobAddResult",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		mocklib.MatchedBy(
			func(taskResult *model.TaskResult) bool {
				assert.True(t, taskResult.Success)
				assert.Equal(t, workflow.Tasks[0].HTTP.URI, taskResult.HTTPRequest.URI)
				assert.Equal(t, workflow.Tasks[0].HTTP.Method, taskResult.HTTPRequest.Method)
				assert.Equal(t, []string{
					"X-Header: Value",
				}, taskResult.HTTPRequest.Headers)
				assert.Equal(t, http.StatusOK, taskResult.HTTPResponse.StatusCode)
				assert.Equal(t, requestBody, taskResult.HTTPResponse.Body)

				return true
			}),
	).Return(nil)

	makeHTTPRequestOriginal := makeHTTPRequest
	makeHTTPRequest = func(req *http.Request, timeout time.Duration) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(requestBody)),
		}
		return resp, nil
	}
	err := processJob(ctx, job, dataStore)
	makeHTTPRequest = makeHTTPRequestOriginal

	assert.Nil(t, err)

	dataStore.AssertExpectations(t)
}

func TestProcessJobHTTPWrongStatusCode(t *testing.T) {
	ctx := context.Background()
	dataStore := mock.NewDataStore()

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeHTTP,
				HTTP: &model.HTTPTask{
					URI:    "http://localhost",
					Method: "GET",
					Headers: map[string]string{
						"X-Header": "Value",
					},
					StatusCodes: []int{
						200,
						201,
					},
				},
			},
			{
				Name: "task_2",
				Type: model.TaskTypeHTTP,
				HTTP: &model.HTTPTask{
					URI:    "http://localhost",
					Method: "GET",
					Headers: map[string]string{
						"X-Header": "Value",
					},
					StatusCodes: []int{
						200,
						201,
					},
				},
			},
		},
	}

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}

	requestBody := "BODY"

	dataStore.On("GetWorkflowByName",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		workflow.Name,
	).Return(workflow, nil)

	dataStore.On("AquireJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
	).Return(job, nil)

	dataStore.On("UpdateJobStatus",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		model.StatusFailure,
	).Return(nil)

	dataStore.On("UpdateJobAddResult",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		mocklib.MatchedBy(
			func(taskResult *model.TaskResult) bool {
				assert.False(t, taskResult.Success)
				assert.Equal(t, workflow.Tasks[0].HTTP.URI, taskResult.HTTPRequest.URI)
				assert.Equal(t, workflow.Tasks[0].HTTP.Method, taskResult.HTTPRequest.Method)
				assert.Equal(t, []string{
					"X-Header: Value",
				}, taskResult.HTTPRequest.Headers)
				assert.Equal(t, http.StatusBadRequest, taskResult.HTTPResponse.StatusCode)
				assert.Equal(t, requestBody, taskResult.HTTPResponse.Body)

				return true
			}),
	).Return(nil)

	makeHTTPRequestOriginal := makeHTTPRequest
	makeHTTPRequest = func(req *http.Request, timeout time.Duration) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(strings.NewReader(requestBody)),
		}
		return resp, nil
	}
	err := processJob(ctx, job, dataStore)
	makeHTTPRequest = makeHTTPRequestOriginal

	assert.Nil(t, err)

	dataStore.AssertExpectations(t)
}

func TestProcessJobHTTPFailedIncompatibleDefinition(t *testing.T) {
	ctx := context.Background()
	dataStore := mock.NewDataStore()

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeHTTP,
			},
		},
	}

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}

	dataStore.On("GetWorkflowByName",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		workflow.Name,
	).Return(workflow, nil)

	dataStore.On("AquireJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
	).Return(job, nil)

	dataStore.On("UpdateJobStatus",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		model.StatusFailure,
	).Return(nil)

	err := processJob(ctx, job, dataStore)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Error: Task definition incompatible with specified type (http)")

	dataStore.AssertExpectations(t)
}
