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

func TestProcessJobCLI(t *testing.T) {
	context := context.Background()
	dataStore := store.NewDataStoreMock()

	taskdef := model.CLITask{
		Command: []string{
			"echo",
			"TEST",
		},
	}
	taskdefJSON, _ := json.Marshal(taskdef)

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			model.Task{
				Name:    "task_1",
				Type:    "cli",
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

	err := processJob(context, job, dataStore)

	assert.Nil(t, err)

	job, _ = dataStore.GetJobByNameAndID(context, job.WorkflowName, job.ID)
	assert.Equal(t, model.StatusDone, job.Status)
	assert.Len(t, job.Results, 1)
	assert.True(t, job.Results[0].Success)
	assert.Equal(t, taskdef.Command, job.Results[0].CLI.Command)
	assert.Equal(t, "TEST\n", job.Results[0].CLI.Output)
	assert.Equal(t, "", job.Results[0].CLI.Error)
	assert.Equal(t, 0, job.Results[0].CLI.ExitCode)
}

func TestProcessJobCLIWrongExitCode(t *testing.T) {
	context := context.Background()
	dataStore := store.NewDataStoreMock()

	taskdef := model.CLITask{
		Command: []string{
			"bash",
			"-c",
			"exit 10",
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
				Type:    "cli",
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

	err := processJob(context, job, dataStore)
	assert.Nil(t, err)

	job, _ = dataStore.GetJobByNameAndID(context, job.WorkflowName, job.ID)
	assert.Equal(t, model.StatusFailure, job.Status)
	assert.Len(t, job.Results, 1)
	assert.False(t, job.Results[0].Success)
	assert.Equal(t, taskdef.Command, job.Results[0].CLI.Command)
	assert.Equal(t, "", job.Results[0].CLI.Output)
	assert.Equal(t, "", job.Results[0].CLI.Error)
	assert.Equal(t, 10, job.Results[0].CLI.ExitCode)
}

func TestProcessJobCLTimeOut(t *testing.T) {
	context := context.Background()
	dataStore := store.NewDataStoreMock()

	taskdef := model.CLITask{
		Command: []string{
			"bash",
			"-c",
			"sleep 10",
		},
		ExecutionTimeOut: 2,
	}
	taskdefJSON, _ := json.Marshal(taskdef)

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			model.Task{
				Name:    "task_1",
				Type:    "cli",
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

	err := processJob(context, job, dataStore)
	assert.Nil(t, err)

	job, _ = dataStore.GetJobByNameAndID(context, job.WorkflowName, job.ID)
	assert.Equal(t, model.StatusFailure, job.Status)
	assert.Len(t, job.Results, 1)
	assert.False(t, job.Results[0].Success)
	assert.Equal(t, taskdef.Command, job.Results[0].CLI.Command)
	assert.Equal(t, "", job.Results[0].CLI.Output)
	assert.Equal(t, "", job.Results[0].CLI.Error)
	assert.Equal(t, -1, job.Results[0].CLI.ExitCode)
}

func TestProcessJobCLIFailedIncompatibleDefinition(t *testing.T) {
	context := context.Background()
	dataStore := store.NewDataStoreMock()

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			model.Task{
				Name:    "task_1",
				Type:    "cli",
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
	assert.EqualError(t, err, "Error: Task definition incompatible with specified type (cli)")

	job, _ = dataStore.GetJobByNameAndID(context, job.WorkflowName, job.ID)
	assert.Equal(t, model.StatusFailure, job.Status)
}
