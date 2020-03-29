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

package worker

import (
	"fmt"
	"os"
	"testing"

	"github.com/mendersoftware/workflows/model"
	"github.com/stretchr/testify/assert"
)

func TestProcessJobString(t *testing.T) {
	workflow := &model.Workflow{
		Name: "test",
		InputParameters: []string{
			"key",
		},
	}
	job := &model.Job{
		InputParameters: []model.InputParameter{
			{
				Name:  "key",
				Value: "test",
			},
		},
	}

	res := processJobString("_${workflow.input.key}_", workflow, job)
	assert.Equal(t, "_test_", res)
}

func TestProcessJobStringEnvVariable(t *testing.T) {
	workflow := &model.Workflow{
		Name: "test",
	}
	job := &model.Job{}

	res := processJobString("_${env.SHELL}_", workflow, job)
	shell := os.Getenv("SHELL")
	expected := fmt.Sprintf("_%s_", shell)
	assert.Equal(t, expected, res)
}

func TestProcessJobStringJSONOutputFromPreviousResult(t *testing.T) {
	var tests = []struct {
		taskResult    model.TaskResult
		expression    string
		expectedValue string
	}{
		{
			taskResult: model.TaskResult{
				Name:    "task_1",
				Type:    model.TaskTypeHTTP,
				Success: true,
				HTTPResponse: &model.TaskResultHTTPResponse{
					StatusCode: 200,
					Body:       "{\"key\": \"value\"}",
				},
			},
			expression:    "_${task_1.json.key}_",
			expectedValue: "_value_",
		},
		{
			taskResult: model.TaskResult{
				Name:    "task_1",
				Type:    model.TaskTypeHTTP,
				Success: true,
				HTTPResponse: &model.TaskResultHTTPResponse{
					StatusCode: 200,
					Body:       "{\"key\": {\"subkey\": \"value\"}}",
				},
			},
			expression:    "_${task_1.json.key.subkey}_",
			expectedValue: "_value_",
		},
		{
			taskResult: model.TaskResult{
				Name:    "task_1",
				Type:    model.TaskTypeHTTP,
				Success: true,
				HTTPResponse: &model.TaskResultHTTPResponse{
					StatusCode: 200,
					Body:       "{\"key\": {\"subkey\": 1}}",
				},
			},
			expression:    "_${task_1.json.key.subkey}_",
			expectedValue: "_1_",
		},
		{
			taskResult: model.TaskResult{
				Name:    "task_1",
				Type:    model.TaskTypeHTTP,
				Success: true,
				HTTPResponse: &model.TaskResultHTTPResponse{
					StatusCode: 200,
					Body:       "{\"key\": {\"subkey\": [\"value\", \"value2\"]}}",
				},
			},
			expression:    "_${task_1.json.key.subkey.[1]}_",
			expectedValue: "_value2_",
		},
		{
			taskResult: model.TaskResult{
				Name:    "task_1",
				Type:    model.TaskTypeCLI,
				Success: true,
				CLI: &model.TaskResultCLI{
					ExitCode: 0,
					Output:   "dummy",
				},
			},
			expression:    "_${task_1.json.key}_",
			expectedValue: "__",
		},
	}

	for _, test := range tests {
		workflow := &model.Workflow{
			Name: "test",
		}
		job := &model.Job{
			Results: []model.TaskResult{test.taskResult},
		}

		res := processJobString(test.expression, workflow, job)
		assert.Equal(t, test.expectedValue, res)
	}
}
