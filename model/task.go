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

package model

// Type of tasks
const (
	TaskTypeCLI  = "cli"
	TaskTypeHTTP = "http"
)

// Task stores the definition of a task within a workflow
type Task struct {
	Name string    `json:"name" bson:"name"`
	Type string    `json:"type" bson:"type"`
	HTTP *HTTPTask `json:"http,omitempty" bson:"http,omitempty"`
	CLI  *CLITask  `json:"cli,omitempty" bson:"cli,omitempty"`
}

// HTTPTask stores the parameters of the HTTP calls for a WorkflowTask
type HTTPTask struct {
	URI               string            `json:"uri" bson:"uri"`
	Method            string            `json:"method" bson:"method"`
	ContentType       string            `json:"contentType,omitempty" bson:"contentType"`
	Body              string            `json:"body,omitempty" bson:"body"`
	StatusCodes       []int             `json:"statusCodes,omitempty" bson:"statusCodes"`
	Headers           map[string]string `json:"headers" bson:"headers"`
	ConnectionTimeOut int               `json:"connectionTimeOut" bson:"connectionTimeOut"`
	ReadTimeOut       int               `json:"readTimeOut" bson:"readTimeOut"`
}

// CLITask stores the parameters of the CLI commands for a WorkflowTask
type CLITask struct {
	Command          []string `json:"command" bson:"command"`
	ExecutionTimeOut int      `json:"executionTimeOut" bson:"executionTimeOut"`
}