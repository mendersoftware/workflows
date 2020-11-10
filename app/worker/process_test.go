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
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store/mock"
	"github.com/stretchr/testify/assert"
)

func TestProcessJobFailedWorkflowDoesNotExist(t *testing.T) {
	ctx := context.Background()
	dataStore := mock.NewDataStore()
	defer dataStore.AssertExpectations(t)

	job := &model.Job{
		ID:           "job",
		WorkflowName: "does_not_exist",
		Status:       model.StatusPending,
	}

	dataStore.On("GetWorkflowByName",
		ctx,
		job.WorkflowName,
	).Return(nil, errors.New("workflow not found"))

	dataStore.On("UpdateJobStatus",
		ctx,
		job,
		model.StatusFailure,
	).Return(nil)

	err := processJob(ctx, job, dataStore)
	assert.Nil(t, err)
}

func TestProcessJobFailedJobIsNotPending(t *testing.T) {
	ctx := context.Background()
	dataStore := mock.NewDataStore()
	defer dataStore.AssertExpectations(t)

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeHTTP,
				HTTP: &model.HTTPTask{
					URI:    "http://localhost",
					Method: http.MethodGet,
					Headers: map[string]string{
						"X-Header": "Value",
					},
				},
			},
		},
	}

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusDone,
	}

	dataStore.On("GetWorkflowByName",
		ctx,
		job.WorkflowName,
	).Return(workflow, nil)

	dataStore.On("AcquireJob",
		ctx,
		job,
	).Return(nil, errors.New("not found"))

	dataStore.On("UpdateJobStatus",
		ctx,
		job,
		model.StatusFailure,
	).Return(nil)

	err := processJob(ctx, job, dataStore)
	assert.Nil(t, err)
}

func TestProcessTaskSkipped(t *testing.T) {
	testCases := map[string]struct {
		workflow *model.Workflow
		job      *model.Job
		task     *model.Task
		skipped  bool
	}{
		"skipped, missing parameter": {
			workflow: &model.Workflow{
				Name: "test",
				InputParameters: []string{
					"request_id",
				},
			},
			job: &model.Job{
				InputParameters: []model.InputParameter{
					{
						Name:  "request_id",
						Value: "",
					},
				},
			},
			task: &model.Task{
				Name: "task_1",
				Type: model.TaskTypeHTTP,
				Requires: []string{
					"${workflow.input.request_id}",
				},
				HTTP: &model.HTTPTask{
					URI:    "http://localhost",
					Method: http.MethodGet,
					Headers: map[string]string{
						"X-Header": "Value",
					},
				},
			},
			skipped: true,
		},
		"executed, requires parameter": {
			workflow: &model.Workflow{
				Name: "test",
			},
			job: &model.Job{
				InputParameters: []model.InputParameter{
					{
						Name:  "request_id",
						Value: "value",
					},
				},
			},
			task: &model.Task{
				Name: "task_1",
				Type: model.TaskTypeHTTP,
				Requires: []string{
					"${workflow.input.request_id}",
				},
				HTTP: &model.HTTPTask{
					URI:    "http://localhost",
					Method: http.MethodGet,
					Headers: map[string]string{
						"X-Header": "Value",
					},
				},
			},
			skipped: false,
		},
		"executed, requires parameter but empty": {
			workflow: &model.Workflow{
				Name: "test",
			},
			job: &model.Job{
				InputParameters: []model.InputParameter{
					{
						Name:  "request_id",
						Value: "",
					},
				},
			},
			task: &model.Task{
				Name: "task_1",
				Type: model.TaskTypeHTTP,
				Requires: []string{
					"${workflow.input.request_id}",
				},
				HTTP: &model.HTTPTask{
					URI:    "http://localhost",
					Method: http.MethodGet,
					Headers: map[string]string{
						"X-Header": "Value",
					},
				},
			},
			skipped: true,
		},
		"executed, requires env": {
			workflow: &model.Workflow{
				Name: "test",
			},
			job: &model.Job{},
			task: &model.Task{
				Name: "task_1",
				Type: model.TaskTypeHTTP,
				Requires: []string{
					"${env.PWD}",
				},
				HTTP: &model.HTTPTask{
					URI:    "http://localhost",
					Method: http.MethodGet,
					Headers: map[string]string{
						"X-Header": "Value",
					},
				},
			},
			skipped: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			makeHTTPRequestOriginal := makeHTTPRequest
			makeHTTPRequest = func(req *http.Request, timeout time.Duration) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader([]byte(""))),
				}, nil
			}

			ctx := context.Background()
			l := log.FromContext(ctx)
			result, err := processTask(*tc.task, tc.job, tc.workflow, l)
			assert.NoError(t, err)
			assert.Equal(t, tc.skipped, result.Skipped)

			makeHTTPRequest = makeHTTPRequestOriginal
		})
	}
}
