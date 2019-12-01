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

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowNotFound(t *testing.T) {
	dataStore := store.NewDataStoreMock()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()

	payload := `{
      "key": "value"
	}`

	req, _ := http.NewRequest("POST", "/api/workflow/test", strings.NewReader(payload))
	router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestWorkflowFoundButMissingParameters(t *testing.T) {
	dataStore := store.NewDataStoreMock()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()
	Workflows = map[string]*model.Workflow{
		"test": {
			Name: "Test workflow",
			InputParameters: []string{
				"param1",
				"param2",
				"param3",
			},
		},
	}

	payload := `{
      "key": "value"
	}`

	req, _ := http.NewRequest("POST", "/api/workflow/test", strings.NewReader(payload))
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	var response map[string]string
	err := json.Unmarshal([]byte(w.Body.String()), &response)
	value, ok := response["error"]

	assert.Nil(t, err)
	assert.True(t, ok)

	expectedBody := gin.H{
		"error": "Missing input parameters: param1, param2, param3",
	}
	assert.Equal(t, expectedBody["error"], value)
}

func TestWorkflowFoundAndLaunchedWithParameters(t *testing.T) {
	dataStore := store.NewDataStoreMock()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()
	Workflows = map[string]*model.Workflow{
		"test": {
			Name: "Test workflow",
			InputParameters: []string{
				"key",
			},
		},
	}

	payload := `{
      "key": "value"
	}`

	req, _ := http.NewRequest("POST", "/api/workflow/test", strings.NewReader(payload))
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, 1, len(dataStore.Jobs))
}
