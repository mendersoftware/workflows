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

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"path/filepath"
)

// ParseWorkflowFromJSON parse a JSON string and returns a Workflow struct
func ParseWorkflowFromJSON(jsonData []byte) (*Workflow, error) {
	var workflow Workflow
	if err := json.Unmarshal(jsonData, &workflow); err != nil {
		return nil, errors.New("unable to parse the JSON")
	}
	return &workflow, nil
}

// GetWorkflowsFromPath parse the workflows stored as JSON files in a directory and returns them
func GetWorkflowsFromPath(path string) map[string]*Workflow {
	var workflows = make(map[string]*Workflow)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil
	}

	for _, f := range files {
		fn := filepath.Join(path, f.Name())
		if data, err := ioutil.ReadFile(fn); err == nil {
			workflow, err := ParseWorkflowFromJSON([]byte(data))
			if err != nil {
				continue
			}
			if workflows[workflow.Name] == nil || workflows[workflow.Name].Version <= workflow.Version {
				workflows[workflow.Name] = workflow
			}
		}
	}
	return workflows
}
