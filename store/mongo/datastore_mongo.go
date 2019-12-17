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

package mongo

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"
	dconfig "github.com/mendersoftware/workflows/config"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
)

const (
	// JobQueueCollectionName refers to the collection of pending jobs
	JobQueueCollectionName = "job_queue"

	// JobCollectionName refers to the collection of finished or
	// jobs in progress.
	JobsCollectionName = "jobs"

	// WorkflowCollectionName refers to the collection of stored workflows
	WorkflowCollectionName = "workflows"
)

// MongoClient is a package specific mongo client
type MongoClient struct {
	mongo.Client
}

// NewMongoClient returns a mongo client
func NewMongoClient(ctx context.Context, c config.Reader) (*MongoClient, error) {

	clientOptions := mopts.Client()
	mongoURL := c.GetString(dconfig.SettingMongo)
	if !strings.Contains(mongoURL, "://") {
		return nil, errors.Errorf("Invalid mongoURL %q: missing schema.",
			mongoURL)
	}
	clientOptions.ApplyURI(mongoURL)

	username := c.GetString(dconfig.SettingDbUsername)
	if username != "" {
		credentials := mopts.Credential{
			Username: c.GetString(dconfig.SettingDbUsername),
		}
		password := c.GetString(dconfig.SettingDbPassword)
		if password != "" {
			credentials.Password = password
			credentials.PasswordSet = true
		}
		clientOptions.SetAuth(credentials)
	}

	if c.GetBool(dconfig.SettingDbSSL) {
		tlsConfig := &tls.Config{}
		tlsConfig.InsecureSkipVerify = c.GetBool(dconfig.SettingDbSSLSkipVerify)
		clientOptions.SetTLSConfig(tlsConfig)
	}

	// Set writeconcern to acknowlage after write has propagated to the
	// mongod instance and commited to the file system journal.
	var wc *writeconcern.WriteConcern
	wc.WithOptions(writeconcern.W(1), writeconcern.J(true))
	clientOptions.SetWriteConcern(wc)

	if clientOptions.ReplicaSet != nil {
		clientOptions.SetReadConcern(readconcern.Linearizable())
	}

	// Set 10s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to mongo server")
	}

	// Validate connection
	if err = client.Ping(ctx, nil); err != nil {
		return nil, errors.Wrap(err, "Error reaching mongo server")
	}

	mongoClient := MongoClient{Client: *client}
	return &mongoClient, nil
}

// DataStoreMongo is the data storage service
type DataStoreMongo struct {
	// client holds the reference to the client used to communicate with the
	// mongodb server.
	client *MongoClient
	// dbName contains the name of the workflow database.
	dbName string
	// workflows holds a local cache of workflows - a worker should NEVER
	// access this cache directly, but through
	// DataStoreMongo.GetWorkflowByName.
	workflows map[string]*model.Workflow
	// shutdown is used to shut down the job scheduler routine.
	shutdown bool
}

// NewDataStoreMongoWithClient initializes a DataStore object
func NewDataStoreWithClient(client *MongoClient, c config.Reader) *DataStoreMongo {
	dbName := c.GetString(dconfig.SettingDbName)
	ctx := context.Background()

	// Maybe initialize workflows
	var findResults []*model.Workflow
	workflows := make(map[string]*model.Workflow)
	database := client.Database(dbName)
	collWflows := database.Collection(WorkflowCollectionName)
	collQueue := database.Collection(JobQueueCollectionName)
	cur, err := collWflows.Find(ctx, bson.M{})
	if err == nil {
		if err = cur.All(ctx, &findResults); err == nil {
			for _, workflow := range findResults {
				workflows[workflow.Name] = workflow
			}
		}
	}
	// If the message bus collection is empty, the trailing cursor dies
	collQueue.InsertOne(ctx, model.Job{WorkflowName: "noop", ID: "0"})

	return &DataStoreMongo{
		client:    client,
		dbName:    dbName,
		workflows: workflows,
		shutdown:  false,
	}
}

// Inserts a workflow to the database and cache and returns the number of
// inserted elements or an error for the first error generated.
func (db *DataStoreMongo) InsertWorkflows(workflows ...model.Workflow) (int, error) {
	var tmp model.Workflow
	ctx := context.Background()
	database := db.client.Database(db.dbName)
	collWflows := database.Collection(WorkflowCollectionName)
	for i, workflow := range workflows {
		if workflow.Name == "" {
			return i, store.ErrWorkflowMissingName
		} else if _, ok := db.workflows[workflow.Name]; ok {
			return i, store.ErrWorkflowAlreadyExists
		}
		tmp = workflow
		if _, err := collWflows.InsertOne(ctx, tmp); err != nil {
			if strings.Contains(err.Error(), "Duplicate key error") {
				return i, store.ErrWorkflowAlreadyExists
			}
			return i, err
		}
		db.workflows[workflow.Name] = &tmp
	}
	return len(workflows), nil
}

// GetWorkflowByName gets the workflow with the given name - either from the
// cache, or searches the database if the workflow is not cached.
func (db *DataStoreMongo) GetWorkflowByName(workflowName string) (*model.Workflow, error) {
	workflow, ok := db.workflows[workflowName]
	if !ok {
		ctx := context.Background()
		var result model.Workflow
		database := db.client.Database(db.dbName)
		collWflows := database.Collection(WorkflowCollectionName)
		err := collWflows.FindOne(ctx, bson.M{"_id": workflowName}).
			Decode(&result)
		if err != nil {
			return nil, err
		}
		db.workflows[result.Name] = &result
		return &result, err
	}
	return workflow, nil
}

// GetWorkflows gets all workflows from the cache as a list
// (should only be used by the server process)
func (db *DataStoreMongo) GetWorkflows() []model.Workflow {
	workflows := make([]model.Workflow, len(db.workflows))
	var i int
	for _, workflow := range db.workflows {
		workflows[i] = *workflow
		i++
	}

	return workflows
}

// InsertJob inserts the job in the queue
func (db *DataStoreMongo) InsertJob(
	ctx context.Context, job *model.Job) (*model.Job, error) {

	if workflow, err := db.GetWorkflowByName(job.WorkflowName); err == nil {
		if err := job.Validate(workflow); err != nil {
			return nil, err
		}
	} else {
		return nil, store.ErrWorkflowNotFound
	}

	id := primitive.NewObjectID()
	job.ID = id.Hex()
	job.Status = model.StatusPending

	database := db.client.Database(db.dbName)
	collQueue := database.Collection(JobQueueCollectionName)
	collJobs := database.Collection(JobsCollectionName)

	// insert the Job in the capped transaction we use as message queue
	if _, err := collQueue.InsertOne(ctx, job); err != nil {
		return nil, errors.Wrap(err,
			"Error inserting job to message queue")
	}
	// insert the same pending job into the global collection
	if _, err := collJobs.InsertOne(ctx, job); err != nil {
		return nil, errors.Wrap(err,
			"Error inserting job into jobs collection")
	}

	return job, nil
}

// GetJobs initializes the job scheduler and returns a receive channel from
// the scheduler routine.
func (db *DataStoreMongo) GetJobs(ctx context.Context) <-chan *model.Job {
	var channel = make(chan *model.Job)

	go func() {
		l := log.FromContext(ctx)
		findOptions := &mopts.FindOptions{}
		findOptions.SetCursorType(mopts.TailableAwait)
		findOptions.SetMaxTime(10 * time.Second)
		findOptions.SetBatchSize(100)

		query := bson.M{"status": model.StatusPending}

		database := db.client.Database(db.dbName)
		collQueue := database.Collection(JobQueueCollectionName)
		cur, err := collQueue.Find(ctx, query, findOptions)
		if err != nil {
			channel <- nil
			l.Error(err.Error())
			return
		}

		defer cur.Close(ctx)
		l.Info("Job scheduler listening to message bus")

		for {
			for cur.TryNext(ctx) {
				job := new(model.Job)
				cur.Decode(job)
				l.Infof("Message bus: New job (%s) with "+
					"workflow %s", job.ID, job.WorkflowName)
				if job != nil &&
					job.Status == model.StatusPending {
					// NOTE: We should probably create an
					//       index for the status key, and
					//       filter the query.
					channel <- job
				}
			}
			if cur.ID() == 0 {
				l.Error("Cursor died!")
				break
			}
			if db.shutdown {
				l.Info("Job scheduler shutting down...")
				break
			}
		}

		channel <- nil
	}()

	return channel
}

// AquireJob gets given job and updates it's status to StatusProcessing.
// On success, the updated job is returned - if the job does not exist nil
// is returned, otherwise a mongo error is returned.
func (db *DataStoreMongo) AquireJob(ctx context.Context,
	job *model.Job) (*model.Job, error) {

	var aquiredJob *model.Job = new(model.Job)

	database := db.client.Database(db.dbName)
	collJobs := database.Collection(JobsCollectionName)
	collQueue := database.Collection(JobQueueCollectionName)

	query := bson.M{
		"_id":    job.ID,
		"status": model.StatusPending,
	}
	update := bson.M{
		"$set": bson.M{"status": model.StatusProcessing},
	}

	err := collQueue.FindOneAndUpdate(ctx, query, update).Decode(aquiredJob)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	updateOpts := mopts.Update()
	updateOpts.SetUpsert(true)
	_, err = collJobs.UpdateOne(ctx, query, update, updateOpts)
	if err != nil {
		return nil, err
	}

	return aquiredJob, nil
}

// UpdateJobAddResult add a task execution result to a job status
func (db *DataStoreMongo) UpdateJobAddResult(ctx context.Context,
	job *model.Job, result *model.TaskResult) error {
	collection := db.client.Database(db.dbName).
		Collection(JobsCollectionName)
	var update bson.M
	if job.Results == nil {
		update = bson.M{"$set": bson.M{"results": bson.A{result}}}
	} else {
		update = bson.M{"$addToSet": bson.M{"results": result}}
	}
	_, err := collection.UpdateOne(ctx, bson.M{"_id": job.ID}, update)
	if err != nil {
		return err
	}

	return nil
}

// UpdateJobStatus set the task execution status for a job status
func (db *DataStoreMongo) UpdateJobStatus(
	ctx context.Context, job *model.Job, status int) error {

	if model.StatusToString(status) == "unknown" {
		return model.ErrInvalidStatus
	}
	collection := db.client.Database(db.dbName).
		Collection(JobsCollectionName)
	_, err := collection.UpdateOne(ctx, bson.M{
		"_id": job.ID,
	}, bson.M{
		"$set": bson.M{
			"status": status,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// GetJobStatusByNameAndID get the task execution status for a job
// by workflow name and ID
func (db *DataStoreMongo) GetJobByNameAndID(
	ctx context.Context, name string, ID string) (*model.Job, error) {
	collection := db.client.Database(db.dbName).
		Collection(JobsCollectionName)
	cur := collection.FindOne(ctx, bson.M{
		"_id":           ID,
		"workflow_name": name,
	})
	var job model.Job
	err := cur.Decode(&job)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &job, nil
}

// Shutdown shuts down the datastore GetJobs process
func (db *DataStoreMongo) Shutdown() {
	db.shutdown = true
}
