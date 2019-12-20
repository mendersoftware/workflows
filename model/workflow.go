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
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/pkg/errors"
)

// Workflow stores the definition of a workflow
type Workflow struct {
	Name            string   `json:"name" bson:"_id"`
	Description     string   `json:"description" bson:"description"`
	Version         int      `json:"version" bson:"version"`
	SchemaVersion   int      `json:"schemaVersion" bson:"schema_version"`
	Tasks           []Task   `json:"tasks" bson:"tasks"`
	InputParameters []string `json:"inputParameters" bson:"input_parameters"`
}

// Task stores the definition of a task within a workflow
type Task struct {
	// Name of the task
	Name string `json:"name"`
	// Type of task (determines task def structure)
	Type string `json:"type"`
	// Definition of the task
	Taskdef json.RawMessage `json:"taskdef" bson:"taskdef"`
}

// HTTPTask stores the parameters of the HTTP calls for a WorkflowTask
type HTTPTask struct {
	URI               string            `json:"uri"`
	Method            string            `json:"method"`
	ContentType       string            `json:"contentType,omitempty"`
	Body              string            `json:"body,omitempty"`
	Headers           map[string]string `json:"headers"`
	ConnectionTimeOut int               `json:"connectionTimeOut"`
	ReadTimeOut       int               `json:"readTimeOut"`
}

// ParseWorkflowFromJSON parse a JSON string and returns a Workflow struct
func ParseWorkflowFromJSON(jsonData []byte) (*Workflow, error) {
	var workflow Workflow
	if err := json.Unmarshal(jsonData, &workflow); err != nil {
		return nil, errors.Wrap(err, "unable to parse the JSON")
	}
	return &workflow, nil
}

// GetWorkflowsFromPath parse the workflows stored as JSON files in a directory and returns them
func GetWorkflowsFromPath(path string) map[string]*Workflow {
	var workflows = make(map[string]*Workflow)
	l := log.NewEmpty()
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		fn := filepath.Join(path, f.Name())
		if data, err := ioutil.ReadFile(fn); err == nil {
			workflow, err := ParseWorkflowFromJSON([]byte(data))
			if err != nil {
				l.Warn(err.Error())
				continue
			}
			if workflows[workflow.Name] == nil || workflows[workflow.Name].Version <= workflow.Version {
				workflows[workflow.Name] = workflow
			}
		}
	}
	return workflows
}
