package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mendersoftware/workflows/workflow"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowNotFound(t *testing.T) {
	router := NewRouter()

	w := httptest.NewRecorder()

	payload := `{
      "key": "value"
	}`

	req, _ := http.NewRequest("POST", "/api/workflow/test", strings.NewReader(payload))
	router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestWorkflowFound(t *testing.T) {
	router := NewRouter()

	w := httptest.NewRecorder()
	Workflows = map[string]*workflow.Workflow{
		"test": &workflow.Workflow{
			Name: "Test workflow",
		},
	}

	payload := `{
      "key": "value"
	}`

	req, _ := http.NewRequest("POST", "/api/workflow/test", strings.NewReader(payload))
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}
