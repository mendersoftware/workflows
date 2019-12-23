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
	"testing"

	"github.com/mendersoftware/workflows/model"
	store "github.com/mendersoftware/workflows/store/mock"
	"github.com/stretchr/testify/assert"
)

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
