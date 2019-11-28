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
