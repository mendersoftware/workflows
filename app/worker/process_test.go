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

package worker

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mendersoftware/workflows/model"
	store "github.com/mendersoftware/workflows/store/mock"
	"github.com/stretchr/testify/assert"
)

func TestProcessJob(t *testing.T) {
	context := context.Background()
	dataStore := store.NewDataStoreMock()

	taskdef := model.HTTPTask{
		URI:    "http://localhost",
		Method: "GET",
		Headers: map[string]string{
			"X-Header": "Value",
		},
	}
	taskdefJSON, _ := json.Marshal(taskdef)

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			model.Task{
				Name:    "task_1",
				Type:    "http",
				Taskdef: taskdefJSON,
			},
		},
	}
	dataStore.InsertWorkflows(*workflow)

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}
	job, _ = dataStore.InsertJob(context, job)

	makeHTTPRequestOriginal := makeHTTPRequest
	requestBody := "BODY"
	makeHTTPRequest = func(req *http.Request, timeout time.Duration) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(requestBody)),
		}
		return resp, nil
	}
	err := processJob(context, job, dataStore)
	makeHTTPRequest = makeHTTPRequestOriginal

	assert.Nil(t, err)

	job, _ = dataStore.GetJobByNameAndID(context, job.WorkflowName, job.ID)
	assert.Equal(t, model.StatusDone, job.Status)
	assert.Len(t, job.Results, 1)
	assert.True(t, job.Results[0].Success)
	assert.Equal(t, job.Results[0].Request.URI, taskdef.URI)
	assert.Equal(t, job.Results[0].Request.Method, taskdef.Method)
	assert.Equal(t, job.Results[0].Request.Headers, []string{
		"X-Header: Value",
	})
	assert.Equal(t, job.Results[0].Response.StatusCode, 200)
	assert.Equal(t, job.Results[0].Response.Body, requestBody)
}

func TestProcessJobValidStatusCode(t *testing.T) {
	context := context.Background()
	dataStore := store.NewDataStoreMock()

	taskdef := model.HTTPTask{
		URI:    "http://localhost",
		Method: "GET",
		Headers: map[string]string{
			"X-Header": "Value",
		},
		StatusCodes: []int{
			200,
		},
	}
	taskdefJSON, _ := json.Marshal(taskdef)

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			model.Task{
				Name:    "task_1",
				Type:    "http",
				Taskdef: taskdefJSON,
			},
		},
	}
	dataStore.InsertWorkflows(*workflow)

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}
	job, _ = dataStore.InsertJob(context, job)

	makeHTTPRequestOriginal := makeHTTPRequest
	requestBody := "BODY"
	makeHTTPRequest = func(req *http.Request, timeout time.Duration) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(requestBody)),
		}
		return resp, nil
	}
	err := processJob(context, job, dataStore)
	makeHTTPRequest = makeHTTPRequestOriginal

	assert.Nil(t, err)

	job, _ = dataStore.GetJobByNameAndID(context, job.WorkflowName, job.ID)
	assert.Equal(t, model.StatusDone, job.Status)
	assert.Len(t, job.Results, 1)
	assert.True(t, job.Results[0].Success)
	assert.Equal(t, job.Results[0].Request.URI, taskdef.URI)
	assert.Equal(t, job.Results[0].Request.Method, taskdef.Method)
	assert.Equal(t, job.Results[0].Request.Headers, []string{
		"X-Header: Value",
	})
	assert.Equal(t, job.Results[0].Response.StatusCode, 200)
	assert.Equal(t, job.Results[0].Response.Body, requestBody)
}

func TestProcessJobWrongStatusCode(t *testing.T) {
	context := context.Background()
	dataStore := store.NewDataStoreMock()

	taskdef := model.HTTPTask{
		URI:    "http://localhost",
		Method: "GET",
		Headers: map[string]string{
			"X-Header": "Value",
		},
		StatusCodes: []int{
			200,
			201,
		},
	}
	taskdefJSON, _ := json.Marshal(taskdef)

	taskdef2 := model.HTTPTask{
		URI:    "http://localhost",
		Method: "GET",
		Headers: map[string]string{
			"X-Header": "Value",
		},
		StatusCodes: []int{
			200,
			201,
		},
	}
	taskdef2JSON, _ := json.Marshal(taskdef2)

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			model.Task{
				Name:    "task_1",
				Type:    "http",
				Taskdef: taskdefJSON,
			},
			model.Task{
				Name:    "task_2",
				Type:    "http",
				Taskdef: taskdef2JSON,
			},
		},
	}
	dataStore.InsertWorkflows(*workflow)

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}
	job, _ = dataStore.InsertJob(context, job)

	makeHTTPRequestOriginal := makeHTTPRequest
	requestBody := "BODY"
	makeHTTPRequest = func(req *http.Request, timeout time.Duration) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: 400,
			Body:       ioutil.NopCloser(strings.NewReader(requestBody)),
		}
		return resp, nil
	}
	err := processJob(context, job, dataStore)
	makeHTTPRequest = makeHTTPRequestOriginal

	assert.Nil(t, err)

	job, _ = dataStore.GetJobByNameAndID(context, job.WorkflowName, job.ID)
	assert.Equal(t, model.StatusFailure, job.Status)
	assert.Len(t, job.Results, 1)
	assert.False(t, job.Results[0].Success)
	assert.Equal(t, job.Results[0].Request.URI, taskdef.URI)
	assert.Equal(t, job.Results[0].Request.Method, taskdef.Method)
	assert.Equal(t, job.Results[0].Request.Headers, []string{
		"X-Header: Value",
	})
	assert.Equal(t, job.Results[0].Response.StatusCode, 400)
	assert.Equal(t, job.Results[0].Response.Body, requestBody)
}

func TestProcessJobFailedWorkflowDoesNotExist(t *testing.T) {
	context := context.Background()
	dataStore := store.NewDataStoreMock()

	job := &model.Job{
		WorkflowName: "does_not_exist",
		Status:       model.StatusPending,
	}
	job = dataStore.AppendJob(job)

	err := processJob(context, job, dataStore)
	assert.Nil(t, err)

	job, _ = dataStore.GetJobByNameAndID(context, job.WorkflowName, job.ID)
	assert.Equal(t, model.StatusFailure, job.Status)
}

func TestProcessJobFailedJobIsNotPending(t *testing.T) {
	context := context.Background()
	dataStore := store.NewDataStoreMock()

	taskdef := model.HTTPTask{
		URI:    "http://localhost",
		Method: "GET",
		Headers: map[string]string{
			"X-Header": "Value",
		},
	}
	taskdefJSON, _ := json.Marshal(taskdef)

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			model.Task{
				Name:    "task_1",
				Type:    "http",
				Taskdef: taskdefJSON,
			},
		},
	}
	dataStore.InsertWorkflows(*workflow)

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusDone,
	}
	job, _ = dataStore.InsertJob(context, job)

	err := processJob(context, job, dataStore)
	assert.Nil(t, err)

	job, _ = dataStore.GetJobByNameAndID(context, job.WorkflowName, job.ID)
	assert.Equal(t, model.StatusFailure, job.Status)
}

func TestProcessJobFailedIncompatibleDefinition(t *testing.T) {
	context := context.Background()
	dataStore := store.NewDataStoreMock()

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			model.Task{
				Name:    "task_1",
				Type:    "http",
				Taskdef: json.RawMessage(""),
			},
		},
	}
	dataStore.InsertWorkflows(*workflow)

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}
	job, _ = dataStore.InsertJob(context, job)

	err := processJob(context, job, dataStore)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Error: Task definition incompatible with specified type (http)")

	job, _ = dataStore.GetJobByNameAndID(context, job.WorkflowName, job.ID)
	assert.Equal(t, model.StatusFailure, job.Status)
}
