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

package http

import (
	"github.com/gin-gonic/gin"
	"github.com/mendersoftware/workflows/store"
)

// API URL used by the HTTP router
const (
	APIURLStatus = "/status"

	APIURLWorkflow   = "/api/workflow/:name"
	APIURLWorkflowID = "/api/workflow/:name/:id"

	APIURLWorkflows = "/api/metadata/workflows"
)

// NewRouter returns the gin router
func NewRouter(dataStore store.DataStore) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	status := NewStatusController()
	router.GET(APIURLStatus, status.Status)

	workflow := NewWorkflowController(dataStore)
	router.POST(APIURLWorkflow, workflow.StartWorkflow)
	router.GET(APIURLWorkflowID, workflow.GetWorkflowByNameAndID)

	router.POST(APIURLWorkflows, workflow.RegisterWorkflow)
	router.GET(APIURLWorkflows, workflow.GetWorkflows)

	return router
}
