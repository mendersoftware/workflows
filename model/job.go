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

package model

import (
	"github.com/pkg/errors"
)

const (
	StatusDone = iota
	StatusPending
	StatusProcessing
	StatusFailure

	// Error messages
	ErrMsgMissingParamF = "Missing input parameters: %s"
)

var (
	// Errors
	ErrInvalidStatus = errors.New("Invalid status")
)

// Job defines the execution job a workflow
type Job struct {
	// Id is the ID of the job
	ID string `json:"id" bson:"_id"`

	// WorkflowName contains the name of the workflow
	WorkflowName string `json:"workflowName" bson:"workflow_name"`

	// InputParameters contains the name of the workflow
	InputParameters []InputParameter `json:"inputParameters" bson:"input_parameters"`

	// Enumerated status of the Job and string field used for unmarshalling
	Status       int    `json:"-" bson:"status"`
	StatusString string `json:"status" bson:"-"`

	// Results produced by a finished job. If status is not "done" this
	// field will always be nil.
	Results []TaskResult `json:"results" bson:"results"`
}

// InputParameter defines the input parameter of a job
type InputParameter struct {
	// Name of the parameter
	Name string `json:"name" bson:"name"`

	// Value of the input parameter
	Value string `json:"value" bson:"value"`
}

// TaskResult contains the result of the execution of a task
type TaskResult struct {
	Request  TaskResultRequest  `json:"request" bson:"request"`
	Response TaskResultResponse `json:"response" bson:"response"`
}

// TaskResultRequest contains the request
type TaskResultRequest struct {
	URI     string   `json:"uri" bson:"uri"`
	Method  string   `json:"method" bson:"method"`
	Body    string   `json:"body" bson:"body"`
	Headers []string `json:"headers" bson:"headers"`
}

// TaskResultResponse contains the response
type TaskResultResponse struct {
	StatusCode int    `json:"statusCode" bson:"status_code"`
	Body       string `json:"body" bson:"body"`
}

// Validate job against workflow. Check that all required parameters are present.
func (job *Job) Validate(workflow *Workflow) error {
	var missing []string
	inputParameters := make(map[string]string)
	for _, param := range job.InputParameters {
		inputParameters[param.Name] = param.Value
	}
	for _, key := range workflow.InputParameters {
		if _, ok := inputParameters[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return errors.Errorf(ErrMsgMissingParamF, missing)
	}
	return nil
}

// GetStatus returns the job's status as a string
func StatusToString(status int) string {
	var ret string
	switch status {
	case StatusPending:
		ret = "pending"
	case StatusProcessing:
		ret = "processing"
	case StatusDone:
		ret = "done"
	case StatusFailure:
		ret = "failed"
	default:
		ret = "unknown"
	}
	return ret
}
