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

package workflow

// Workflow stores the definition of a workflow
type Workflow struct {
	Name            string
	Description     string
	Version         int
	SchemaVersion   int
	Tasks           []Task
	InputParameters []string
}

// Task stores the definition of a task within a workflow
type Task struct {
	Name string
	Type string
	HTTP HTTPParams
}

// HTTPParams stores the parameters of the HTTP calls for a WorkflowTask
type HTTPParams struct {
	URI               string
	Method            string
	Payload           string
	Headers           map[string]string
	ConnectionTimeOut int
	ReadTimeOut       int
}
