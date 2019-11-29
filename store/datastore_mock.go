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

package store

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendersoftware/workflows/model"
)

// DataStoreMock is a mocked data storage service
type DataStoreMock struct {
	// Jobs contains the list of queued jobs
	Jobs    []model.Job
	channel chan *model.Job
}

// NewDataStoreMock initializes a DataStore mock object
func NewDataStoreMock() *DataStoreMock {

	return &DataStoreMock{
		channel: make(chan *model.Job),
	}
}

// InsertJob inserts the job in the queue
func (db *DataStoreMock) InsertJob(ctx context.Context, job *model.Job) (*model.Job, error) {
	job.ID = primitive.NewObjectID().Hex()
	db.Jobs = append(db.Jobs, *job)

	return job, nil
}

// GetJobs returns a channel of Jobs
func (db *DataStoreMock) GetJobs(ctx context.Context) <-chan *model.Job {
	return db.channel
}

// Shutdown shuts down the datastore GetJobs process
func (db *DataStoreMock) Shutdown() {

}
