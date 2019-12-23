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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseWorkflowFromJSON(t *testing.T) {
	data := []byte(`
{
	"name": "decommission_device",
	"description": "Removes device info from all services.",
	"version": 4,
	"tasks": [
		{
			"name": "delete_device_inventory",
			"type": "http",
			"taskdef": {
				"uri": "http://mender-inventory:8080/api/0.1.0/devices/${workflow.input.device_id}",
				"method": "DELETE",
				"body": "Payload",
				"headers": {
					"X-MEN-RequestID": "${workflow.input.request_id}",
					"Authorization": "${workflow.input.authorization}"
				},
				"statusCodes": [
					200,
					201
				],
				"connectionTimeOut": 1000,
				"readTimeOut": 1000
			}
		}
	],
	"inputParameters": [
		"device_id",
		"request_id",
		"authorization"
	],
	"schemaVersion": 1
}`)

	var workflow, _ = ParseWorkflowFromJSON(data)
	assert.NotNil(t, workflow)
	assert.Equal(t, "decommission_device", workflow.Name)
	assert.Equal(t, "Removes device info from all services.", workflow.Description)
	assert.Equal(t, 4, workflow.Version)
	assert.Equal(t, 1, workflow.SchemaVersion)

	var inputParameters = workflow.InputParameters
	assert.Len(t, inputParameters, 3)
	assert.Equal(t, inputParameters[0], "device_id")
	assert.Equal(t, inputParameters[1], "request_id")
	assert.Equal(t, inputParameters[2], "authorization")

	var tasks = workflow.Tasks
	assert.Len(t, tasks, 1)
	assert.Equal(t, tasks[0].Name, "delete_device_inventory")
	assert.Equal(t, tasks[0].Type, "http")
	assert.NotNil(t, tasks[0].Taskdef)
	var httpTask HTTPTask
	err := json.Unmarshal(tasks[0].Taskdef, &httpTask)
	assert.NoError(t, err)
	assert.Equal(t, httpTask.URI, "http://mender-inventory:8080/api/0.1.0/devices/${workflow.input.device_id}")
	assert.Equal(t, httpTask.Method, "DELETE")
	assert.Equal(t, httpTask.Body, "Payload")
	assert.Len(t, httpTask.Headers, 2)
	assert.Len(t, httpTask.StatusCodes, 2)
	assert.Equal(t, httpTask.StatusCodes[0], 200)
	assert.Equal(t, httpTask.StatusCodes[1], 201)
	assert.Equal(t, httpTask.Headers["X-MEN-RequestID"], "${workflow.input.request_id}")
	assert.Equal(t, httpTask.Headers["Authorization"], "${workflow.input.authorization}")
	assert.Equal(t, httpTask.ConnectionTimeOut, 1000)
	assert.Equal(t, httpTask.ReadTimeOut, 1000)
}

func TestParseWorkflowFromInvalidJSON(t *testing.T) {
	data := []byte(`INVALID JSON`)

	var workflow, err = ParseWorkflowFromJSON(data)
	assert.Nil(t, workflow)
	assert.NotNil(t, err)
}

func TestGetWorkflowsFromPath(t *testing.T) {
	data := []byte(`
	{
		"name": "decommission_device",
		"description": "Removes device info from all services.",
		"version": 4,
		"tasks": [
			{
				"name": "delete_device_inventory",
				"type": "HTTP",
				"http": {
					"uri": "http://mender-inventory:8080/api/0.1.0/devices/${workflow.input.device_id}",
					"method": "DELETE",
					"body": "Payload",
					"headers": {
						"X-MEN-RequestID": "${workflow.input.request_id}",
						"Authorization": "${workflow.input.authorization}"
					},
					"connectionTimeOut": 1000,
					"readTimeOut": 1000
				}
			}
		],
		"inputParameters": [
			"device_id",
			"request_id",
			"authorization"
		],
		"schemaVersion": 1
	}`)

	dir, _ := ioutil.TempDir("", "example")
	defer os.RemoveAll(dir) // clean up

	tmpfn := filepath.Join(dir, "test.json")
	ioutil.WriteFile(tmpfn, data, 0666)

	workflows := GetWorkflowsFromPath(dir)
	assert.Len(t, workflows, 1)
	assert.Equal(t, "decommission_device", workflows["decommission_device"].Name)
}
