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

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
	"github.com/mendersoftware/workflows/store/mock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowNotFound(t *testing.T) {
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()

	dataStore.On("InsertJob", context.Background(), &model.Job{
		WorkflowName:    "test",
		InputParameters: []model.InputParameter{{Name: "key", Value: "value"}},
	}).Return(nil, store.ErrWorkflowNotFound)

	payload := `{
      "key": "value"
	}`

	req, _ := http.NewRequest("POST", "/api/v1/workflow/test", strings.NewReader(payload))
	router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestWorkflowFoundButMissingParameters(t *testing.T) {
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()
	workflow := model.Workflow{
		Name: "test",
		InputParameters: []string{
			"param1",
			"param2",
			"param3",
		},
	}
	ctx := context.Background()
	call := dataStore.On("InsertWorkflows", ctx, []model.Workflow{workflow})
	assert.NotNil(t, call)
	call.Return(1, nil)
	_, err := dataStore.InsertWorkflows(ctx, workflow)
	assert.NoError(t, err)

	w = httptest.NewRecorder()
	payload := `{
      "key": "value"
	}`

	req, err := http.NewRequest("POST", "/api/v1/workflow/test", strings.NewReader(payload))
	assert.NoError(t, err)

	dataStore.On("InsertJob", ctx, &model.Job{
		WorkflowName: "test",
		InputParameters: []model.InputParameter{{
			Name:  "key",
			Value: "value"}},
	}).Return(nil, errors.New("Missing input parameters: "+
		"[param1 param2 param3]"))

	router.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)

	var response map[string]string
	body := w.Body.Bytes()
	err = json.Unmarshal(body, &response)
	value, ok := response["error"]

	assert.Nil(t, err)
	assert.True(t, ok)

	expectedBody := gin.H{
		"error": "Missing input parameters: [param1 param2 param3]",
	}
	assert.Equal(t, expectedBody["error"], value)
}

func TestWorkflowFoundAndLaunchedWithParameters(t *testing.T) {
	dataStore := mock.NewDataStore()
	router := NewRouter(dataStore)

	w := httptest.NewRecorder()
	payload := `{
      "key": "value"
	}`
	job := model.Job{
		WorkflowName: "test",
		InputParameters: []model.InputParameter{{
			Name: "key", Value: "value",
		}},
	}
	ret := job
	ret.ID = "TWELVECHRSTR"
	ret.Status = model.StatusPending
	dataStore.On("InsertJob", context.Background(), &job).Return(&ret, nil)
	req, _ := http.NewRequest("POST", "/api/v1/workflow/test", strings.NewReader(payload))
	router.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)
}
