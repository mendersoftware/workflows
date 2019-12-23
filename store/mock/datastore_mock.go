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

package mock

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
)

// DataStoreMock is a mocked data storage service
type DataStoreMock struct {
	// Jobs contains the list of queued jobs
	Jobs      []model.Job
	Workflows map[string]*model.Workflow
	channel   chan *model.Job
}

// NewDataStoreMock initializes a DataStore mock object
func NewDataStoreMock() *DataStoreMock {

	return &DataStoreMock{
		channel:   make(chan *model.Job),
		Workflows: make(map[string]*model.Workflow),
	}
}

func (db *DataStoreMock) InsertWorkflows(workflows ...model.Workflow) (int, error) {
	for _, workflow := range workflows {
		db.Workflows[workflow.Name] = &workflow
	}
	return len(workflows), nil
}

func (db *DataStoreMock) GetWorkflowByName(
	workflowName string) (*model.Workflow, error) {
	return db.Workflows[workflowName], nil
}

func (db *DataStoreMock) GetWorkflows() []model.Workflow {
	workflows := make([]model.Workflow, len(db.Workflows))
	i := 0
	for _, workflow := range db.Workflows {
		workflows[i] = *workflow
		i++
	}
	return workflows
}

// InsertJob inserts the job in the queue
func (db *DataStoreMock) InsertJob(ctx context.Context, job *model.Job) (*model.Job, error) {
	job.ID = primitive.NewObjectID().Hex()
	if wf, ok := db.Workflows[job.WorkflowName]; ok {
		if err := job.Validate(wf); err != nil {
			return nil, err
		}
	} else {
		return nil, store.ErrWorkflowNotFound
	}
	db.Jobs = append(db.Jobs, *job)

	return job, nil
}

// GetJobs returns a channel of Jobs
func (db *DataStoreMock) GetJobs(ctx context.Context) <-chan *model.Job {
	return db.channel
}

// AquireJob gets given job and updates it's status to StatusProcessing.
func (db *DataStoreMock) AquireJob(ctx context.Context,
	job *model.Job) (*model.Job, error) {
	return nil, nil
}

// UpdateJobAddResult add a task execution result to a job status
func (db *DataStoreMock) UpdateJobAddResult(ctx context.Context,
	job *model.Job, result *model.TaskResult) error {
	return nil
}

// UpdateJobStatus set the task execution status for a job status
func (db *DataStoreMock) UpdateJobStatus(ctx context.Context, job *model.Job,
	status int) error {
	return nil
}

// GetJobStatusByNameAndID get the task execution status for a job status bu Name and ID
func (db *DataStoreMock) GetJobByNameAndID(ctx context.Context,
	name string, ID string) (*model.Job, error) {
	for _, job := range db.Jobs {
		if job.WorkflowName == name && job.ID == ID {
			return &job, nil
		}
	}
	return nil, nil
}

// Shutdown shuts down the datastore GetJobs process
func (db *DataStoreMock) Shutdown() {

}
