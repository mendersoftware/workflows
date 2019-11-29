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

// Job defines the execution job a workflow
type Job struct {
	// Id is the ID of the job
	ID string `json:"id" bson:"_id"`

	// WorkflowName contains the name of the workflow
	WorkflowName string `json:"workflow_name" bson:"workflow_name"`

	// InputParameters contains the name of the workflow
	InputParameters []InputParameter `json:"input_parameters" bson:"input_parameters"`
}

// InputParameter defines the input parameter of a job
type InputParameter struct {
	// Name of the parameter
	Name string

	// Value of the input parameter
	Value string
}
