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

// JobStatus defines the status of the execution of a job
type JobStatus struct {
	// Id is the ID of the job
	ID string `json:"id" bson:"_id"`

	// WorkflowName contains the name of the workflow
	WorkflowName string `json:"workflow_name" bson:"workflow_name"`

	// InputParameters contains the name of the workflow
	InputParameters []InputParameter `json:"input_parameters" bson:"input_parameters"`

	// WorkflowName contains the name of the workflow
	Status string `json:"status" bson:"status"`

	Results []TaskResult `json:"results" bson:"results"`
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
	Payload string   `json:"payload" bson:"payload"`
	Headers []string `json:"headers" bson:"headers"`
}

// TaskResultResponse contains the response
type TaskResultResponse struct {
	StatusCode string `json:"statuscode" bson:"statuscode"`
	Body       string `json:"body" bson:"body"`
}
