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

package mongo

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendersoftware/go-lib-micro/config"
	dconfig "github.com/mendersoftware/workflows/config"
	"github.com/mendersoftware/workflows/model"
	"github.com/stretchr/testify/assert"
)

func TestInsertWorkflows(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	workflow := model.Workflow{
		Name:          "TestInsertWorkflows",
		Description:   "Description",
		SchemaVersion: 1,
		Version:       1,
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeCLI,
			},
		},
		InputParameters: []string{
			"key1",
			"key2",
		},
	}

	ctx := context.Background()
	count, _ := testDataStore.InsertWorkflows(ctx, workflow, workflow)
	assert.Equal(t, 1, count)

	workflowFromDb, err := testDataStore.GetWorkflowByName(ctx, workflow.Name)
	assert.Nil(t, err)
	assert.Equal(t, workflow.Name, workflowFromDb.Name)
	assert.Equal(t, workflow.Description, workflowFromDb.Description)
	assert.Equal(t, workflow.SchemaVersion, workflowFromDb.SchemaVersion)
	assert.Equal(t, workflow.Version, workflowFromDb.Version)
	assert.Equal(t, workflow.InputParameters, workflowFromDb.InputParameters)

	workflows := testDataStore.GetWorkflows(ctx)
	assert.Len(t, workflows, 1)
	assert.Equal(t, workflow.Name, workflows[0].Name)
	assert.Equal(t, workflow.Description, workflows[0].Description)
	assert.Equal(t, workflow.SchemaVersion, workflows[0].SchemaVersion)
	assert.Equal(t, workflow.Version, workflows[0].Version)
	assert.Equal(t, workflow.InputParameters, workflows[0].InputParameters)

	count, err = testDataStore.InsertWorkflows(ctx, workflow)
	assert.NotNil(t, err)
	assert.Equal(t, 0, count)

	testDataStore.workflows = make(map[string]*model.Workflow)
	count, err = testDataStore.InsertWorkflows(ctx, workflow)
	assert.NotNil(t, err)
	assert.Equal(t, 0, count)

	workflowFromDb, err = testDataStore.GetWorkflowByName(ctx, workflow.Name)
	assert.Nil(t, err)
	assert.Equal(t, workflow.Name, workflowFromDb.Name)
	assert.Equal(t, workflow.Description, workflowFromDb.Description)
	assert.Equal(t, workflow.SchemaVersion, workflowFromDb.SchemaVersion)
	assert.Equal(t, workflow.Version, workflowFromDb.Version)
	assert.Equal(t, workflow.InputParameters, workflowFromDb.InputParameters)
}

func TestLoadWorkflows(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	data := []byte(`
{
	"name": "decommission_device_from_filesystem",
	"description": "Removes device info from all services.",
	"version": 4,
	"tasks": [
		{
			"name": "delete_device_inventory",
			"type": "http",
			"http": {
				"uri": "http://mender-inventory:8080/api/0.1.0/devices/${workflow.input.device_id}",
				"method": "DELETE",
				"body": "Payload",
				"headers": {
					"X-MEN-RequestID": "${workflow.input.request_id}",
					"Authorization": "${workflow.input.authorization}"
				},
				"connectionTimeOut": 1000,
				"readTimeOut": 1000
			}
		}
	],
	"inputParameters": [
		"device_id",
		"request_id",
		"authorization"
	],
	"schemaVersion": 1
}`)

	dir, _ := ioutil.TempDir("", "example")
	defer os.RemoveAll(dir) // clean up

	config.Config.Set(dconfig.SettingWorkflowsPath, dir)

	tmpfn := filepath.Join(dir, "test.json")
	ioutil.WriteFile(tmpfn, data, 0666)

	ctx := context.Background()
	err := testDataStore.LoadWorkflows(ctx)
	assert.Nil(t, err)

	workflowFromDb, err := testDataStore.GetWorkflowByName(ctx, "decommission_device_from_filesystem")
	assert.Nil(t, err)
	assert.Equal(t, "decommission_device_from_filesystem", workflowFromDb.Name)
	assert.Equal(t, "Removes device info from all services.", workflowFromDb.Description)
	assert.Equal(t, 1, workflowFromDb.SchemaVersion)
	assert.Equal(t, 4, workflowFromDb.Version)
}

func TestInsertWorkflowsMissingName(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	workflow := model.Workflow{
		Name:          "",
		Description:   "Description",
		SchemaVersion: 1,
		Version:       1,
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeCLI,
			},
		},
		InputParameters: []string{
			"key1",
			"key2",
		},
	}

	ctx := context.Background()
	count, err := testDataStore.InsertWorkflows(ctx, workflow)
	assert.NotNil(t, err)
	assert.Equal(t, 0, count)
}

func TestGetWorkflowByName(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	workflowFromDb, err := testDataStore.GetWorkflowByName(ctx, "TestGetWorkflowByName")
	assert.NotNil(t, err)
	assert.Nil(t, workflowFromDb)
}

func TestInsertJob(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	workflow := model.Workflow{
		Name:          "TestInsertJob",
		Description:   "Description",
		SchemaVersion: 1,
		Version:       1,
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeCLI,
			},
		},
		InputParameters: []string{
			"key1",
			"key2",
		},
	}

	ctx := context.Background()
	count, _ := testDataStore.InsertWorkflows(ctx, workflow)
	assert.Equal(t, 1, count)

	job := &model.Job{
		WorkflowName: workflow.Name,
		InputParameters: []model.InputParameter{
			{
				Name:  "key1",
				Value: "value1",
			},
			{
				Name:  "key2",
				Value: "value2",
			},
		},
	}

	inserted, err := testDataStore.InsertJob(ctx, job)
	assert.Nil(t, err)
	assert.Equal(t, job.WorkflowName, inserted.WorkflowName)
	assert.Equal(t, job.InputParameters, inserted.InputParameters)
	assert.Equal(t, model.StatusPending, inserted.Status)
}

func TestInsertJobWorkflowNotFound(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	job := &model.Job{
		WorkflowName: "TestInsertJobWorkflowNotFound",
		InputParameters: []model.InputParameter{
			{
				Name:  "key1",
				Value: "value1",
			},
			{
				Name:  "key2",
				Value: "value2",
			},
		},
	}

	ctx := context.Background()
	inserted, err := testDataStore.InsertJob(ctx, job)
	assert.Nil(t, inserted)
	assert.Error(t, err, "Workflow not found")
}

func TestAcquireJob(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	workflow := model.Workflow{
		Name:          "TestAcquireJob",
		Description:   "Description",
		SchemaVersion: 1,
		Version:       1,
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeCLI,
			},
		},
		InputParameters: []string{
			"key1",
			"key2",
		},
	}

	ctx := context.Background()
	count, _ := testDataStore.InsertWorkflows(ctx, workflow)
	assert.Equal(t, 1, count)

	job := &model.Job{
		WorkflowName: workflow.Name,
		InputParameters: []model.InputParameter{
			{
				Name:  "key1",
				Value: "value1",
			},
			{
				Name:  "key2",
				Value: "value2",
			},
		},
	}

	inserted, err := testDataStore.InsertJob(ctx, job)
	assert.Nil(t, err)

	aquired, err := testDataStore.AquireJob(ctx, inserted)
	assert.Nil(t, err)
	assert.NotNil(t, aquired)
	assert.Equal(t, model.StatusProcessing, aquired.Status)
}

func TestUpdateJobStatus(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	workflow := model.Workflow{
		Name:          "TestUpdateJobStatus",
		Description:   "Description",
		SchemaVersion: 1,
		Version:       1,
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeCLI,
			},
		},
		InputParameters: []string{
			"key1",
			"key2",
		},
	}

	ctx := context.Background()
	count, _ := testDataStore.InsertWorkflows(ctx, workflow)
	assert.Equal(t, 1, count)

	job := &model.Job{
		WorkflowName: workflow.Name,
		InputParameters: []model.InputParameter{
			{
				Name:  "key1",
				Value: "value1",
			},
			{
				Name:  "key2",
				Value: "value2",
			},
		},
	}

	inserted, err := testDataStore.InsertJob(ctx, job)
	assert.Nil(t, err)

	err = testDataStore.UpdateJobStatus(ctx, inserted, model.StatusFailure)
	assert.Nil(t, err)

	jobFromDb, err := testDataStore.GetJobByNameAndID(ctx, workflow.Name, inserted.ID)
	assert.Nil(t, err)
	assert.Equal(t, model.StatusFailure, jobFromDb.Status)
}

func TestUpdateJobStatusInvalid(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	workflow := model.Workflow{
		Name:          "TestUpdateJobStatusInvalid",
		Description:   "Description",
		SchemaVersion: 1,
		Version:       1,
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeCLI,
			},
		},
		InputParameters: []string{
			"key1",
			"key2",
		},
	}

	ctx := context.Background()
	count, _ := testDataStore.InsertWorkflows(ctx, workflow)
	assert.Equal(t, 1, count)

	job := &model.Job{
		WorkflowName: workflow.Name,
		InputParameters: []model.InputParameter{
			{
				Name:  "key1",
				Value: "value1",
			},
			{
				Name:  "key2",
				Value: "value2",
			},
		},
	}

	inserted, err := testDataStore.InsertJob(ctx, job)
	assert.Nil(t, err)

	err = testDataStore.UpdateJobStatus(ctx, inserted, 99999)
	assert.NotNil(t, err)

	jobFromDb, err := testDataStore.GetJobByNameAndID(ctx, workflow.Name, inserted.ID)
	assert.Nil(t, err)
	assert.Equal(t, model.StatusPending, jobFromDb.Status)
}

func TestUpdateJobAddResult(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	workflow := model.Workflow{
		Name:          "TestUpdateJobAddResult",
		Description:   "Description",
		SchemaVersion: 1,
		Version:       1,
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeCLI,
			},
		},
		InputParameters: []string{
			"key1",
			"key2",
		},
	}

	ctx := context.Background()
	count, _ := testDataStore.InsertWorkflows(ctx, workflow)
	assert.Equal(t, 1, count)

	job := &model.Job{
		WorkflowName: workflow.Name,
		InputParameters: []model.InputParameter{
			{
				Name:  "key1",
				Value: "value1",
			},
			{
				Name:  "key2",
				Value: "value2",
			},
		},
	}

	inserted, err := testDataStore.InsertJob(ctx, job)
	assert.Nil(t, err)

	result := &model.TaskResult{
		CLI: &model.TaskResultCLI{
			Command: []string{
				"command",
			},
			Output:   "output",
			Error:    "error",
			ExitCode: 0,
		},
	}
	err = testDataStore.UpdateJobAddResult(ctx, inserted, result)
	assert.Nil(t, err)

	jobFromDb, err := testDataStore.GetJobByNameAndID(ctx, workflow.Name, inserted.ID)
	assert.Nil(t, err)
	assert.Len(t, jobFromDb.Results, 1)
	assert.NotNil(t, jobFromDb.Results[0].CLI)
	assert.Equal(t, []string{"command"}, jobFromDb.Results[0].CLI.Command)
	assert.Equal(t, "output", jobFromDb.Results[0].CLI.Output)
	assert.Equal(t, "error", jobFromDb.Results[0].CLI.Error)
	assert.Equal(t, 0, jobFromDb.Results[0].CLI.ExitCode)
}

func TestGetJobs(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	workflow := model.Workflow{
		Name:          "TestGetJobs",
		Description:   "Description",
		SchemaVersion: 1,
		Version:       1,
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeCLI,
			},
		},
		InputParameters: []string{
			"key1",
			"key2",
		},
	}

	ctx := context.Background()
	count, _ := testDataStore.InsertWorkflows(ctx, workflow)
	assert.Equal(t, 1, count)

	job := &model.Job{
		WorkflowName: workflow.Name,
		InputParameters: []model.InputParameter{
			{
				Name:  "key1",
				Value: "value1",
			},
			{
				Name:  "key2",
				Value: "value2",
			},
		},
	}

	_, err := testDataStore.InsertJob(ctx, job)
	assert.Nil(t, err)

	channel, _ := testDataStore.GetJobs(ctx, []string{}, []string{})
	var jobReceived *model.Job
	for {
		jobReceived = (<-channel).(*model.Job)
		if jobReceived.WorkflowName == job.WorkflowName {
			break
		}
	}

	assert.NotNil(t, jobReceived)
	assert.Equal(t, job.WorkflowName, jobReceived.WorkflowName)
	assert.Equal(t, job.InputParameters, jobReceived.InputParameters)
	assert.Equal(t, model.StatusPending, jobReceived.Status)
}

func TestGetJobsIncludedWorkflowName(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	workflow := model.Workflow{
		Name:          "TestGetJobsIncludedWorkflowName",
		Description:   "Description",
		SchemaVersion: 1,
		Version:       1,
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeCLI,
			},
		},
		InputParameters: []string{
			"key1",
			"key2",
		},
	}

	ctx := context.Background()
	count, _ := testDataStore.InsertWorkflows(ctx, workflow)
	assert.Equal(t, 1, count)

	job := &model.Job{
		WorkflowName: workflow.Name,
		InputParameters: []model.InputParameter{
			{
				Name:  "key1",
				Value: "value1",
			},
			{
				Name:  "key2",
				Value: "value2",
			},
		},
	}

	_, err := testDataStore.InsertJob(ctx, job)
	assert.Nil(t, err)

	channel, _ := testDataStore.GetJobs(ctx, []string{"TestGetJobsIncludedWorkflowName"}, []string{})
	var jobReceived *model.Job
	for {
		jobReceived = (<-channel).(*model.Job)
		if jobReceived.WorkflowName == job.WorkflowName {
			break
		}
	}

	assert.NotNil(t, jobReceived)
	assert.Equal(t, job.WorkflowName, jobReceived.WorkflowName)
	assert.Equal(t, job.InputParameters, jobReceived.InputParameters)
	assert.Equal(t, model.StatusPending, jobReceived.Status)
}

func TestGetJobsExcludedWorkflowName(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	workflow := model.Workflow{
		Name:          "TestGetJobsExcludedWorkflowName",
		Description:   "Description",
		SchemaVersion: 1,
		Version:       1,
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeCLI,
			},
		},
		InputParameters: []string{
			"key1",
			"key2",
		},
	}

	ctx := context.Background()
	count, _ := testDataStore.InsertWorkflows(ctx, workflow)
	assert.Equal(t, 1, count)

	job := &model.Job{
		WorkflowName: workflow.Name,
		InputParameters: []model.InputParameter{
			{
				Name:  "key1",
				Value: "value1",
			},
			{
				Name:  "key2",
				Value: "value2",
			},
		},
	}

	_, err := testDataStore.InsertJob(ctx, job)
	assert.Nil(t, err)

	channel, _ := testDataStore.GetJobs(ctx, []string{}, []string{"dummy"})
	var jobReceived *model.Job
	for {
		jobReceived = (<-channel).(*model.Job)
		if jobReceived.WorkflowName == job.WorkflowName {
			break
		}
	}

	assert.NotNil(t, jobReceived)
	assert.Equal(t, job.WorkflowName, jobReceived.WorkflowName)
	assert.Equal(t, job.InputParameters, jobReceived.InputParameters)
	assert.Equal(t, model.StatusPending, jobReceived.Status)
}

func TestGetJobsIncludedAndExcludedWorkflowNames(t *testing.T) {
	flag.Parse()
	if testing.Short() {
		t.Skip()
	}

	workflow := model.Workflow{
		Name:          "TestGetJobsIncludedAndExcludedWorkflowNames",
		Description:   "Description",
		SchemaVersion: 1,
		Version:       1,
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeCLI,
			},
		},
		InputParameters: []string{
			"key1",
			"key2",
		},
	}

	ctx := context.Background()
	count, _ := testDataStore.InsertWorkflows(ctx, workflow)
	assert.Equal(t, 1, count)

	job := &model.Job{
		WorkflowName: workflow.Name,
		InputParameters: []model.InputParameter{
			{
				Name:  "key1",
				Value: "value1",
			},
			{
				Name:  "key2",
				Value: "value2",
			},
		},
	}

	_, err := testDataStore.InsertJob(ctx, job)
	assert.Nil(t, err)

	channel, _ := testDataStore.GetJobs(ctx, []string{"TestGetJobsIncludedAndExcludedWorkflowNames"}, []string{"dummy"})
	var jobReceived *model.Job
	for {
		jobReceived = (<-channel).(*model.Job)
		if jobReceived.WorkflowName == job.WorkflowName {
			break
		}
	}

	assert.NotNil(t, jobReceived)
	assert.Equal(t, job.WorkflowName, jobReceived.WorkflowName)
	assert.Equal(t, job.InputParameters, jobReceived.InputParameters)
	assert.Equal(t, model.StatusPending, jobReceived.Status)
}
