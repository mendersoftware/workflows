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
	"encoding/json"
)

// Task stores the definition of a task within a workflow
type Task struct {
	// Name of the task
	Name string `json:"name"`
	// Type of task (determines task def structure)
	Type string `json:"type"`
	// Definition of the task
	Taskdef json.RawMessage `json:"taskdef" bson:"taskdef"`
}

type Result interface {
	// Marshal converts a specific type of results to a generic one
	Marshal() (*TaskResult, error)
}

// Generic TaskResult type
type TaskResult struct {
	// Name of the completed task (necessary if execution is out of order)
	Name string `json:"name" bson:"name"`
	// Raw json of underlying results - may vary depending on type.
	Result []byte `json:"result" bson:"result"`
	// Error message if any errors occured during execution
	Error string `json:"error,omitempty" bson:"error"`
}

func (t *TaskResult) Marshal() (*TaskResult, error) {
	return t, nil
}

// HTTPTask stores the parameters of the HTTP calls for a WorkflowTask
type HTTPTask struct {
	URI               string            `json:"uri"`
	Method            string            `json:"method"`
	Headers           map[string]string `json:"headers"`
	Body              string            `json:"body,omitempty"`
	ConnectionTimeOut int               `json:"connectionTimeOut,omitempty"`
	ReadTimeOut       int               `json:"readTimeOut,omitempty"`
}

// TaskResult contains the result of the execution of a task
type HTTPResult struct {
	Name     string             `json:"name" bson:"name"`
	Response HTTPResultResponse `json:"response" bson:"response"`
	Error    string             `json:"-"`
}

// TaskResultResponse contains the response
type HTTPResultResponse struct {
	StatusCode int                 `json:"statusCode" bson:"status_code"`
	Headers    map[string][]string `json:"headers" bson:"headers"`
	Body       string              `json:"body" bson:"body"`
}

func (t *HTTPResult) Marshal() (*TaskResult, error) {
	var ret TaskResult
	buf, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	ret.Result = buf
	ret.Error = t.Error
	return &ret, nil
}
