package store

import (
	"context"
	"errors"

	"github.com/mendersoftware/workflows/model"

	"go.mongodb.org/mongo-driver/bson"
)

var (
	ErrWorkflowNotFound      = errors.New("Workflow not found")
	ErrWorkflowMissingName   = errors.New("Workflow missing name")
	ErrWorkflowAlreadyExists = errors.New("Workflow already exists")
)

// DataStoreMongoInterface for DataStore  services
type DataStore interface {
	InsertWorkflows(workflow ...model.Workflow) (int, error)
	GetWorkflowByName(workflowName string) (*model.Workflow, error)
	GetWorkflows() []model.Workflow
	InsertJob(ctx context.Context, job *model.Job) (*model.Job, error)
	GetJobs(ctx context.Context) <-chan *model.Job
	AquireJob(ctx context.Context, job *model.Job) (*model.Job, error)
	UpdateJobAddResult(ctx context.Context, job *model.Job, data bson.M) error
	UpdateJobStatus(ctx context.Context, job *model.Job, status int) error
	GetJobByNameAndID(ctx context.Context, name string, ID string) (*model.Job, error)
	Shutdown()
}
