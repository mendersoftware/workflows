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

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"
	dconfig "github.com/mendersoftware/workflows/config"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
)

// InitAndRun initializes the worker and runs it
func InitAndRun(conf config.Reader, dataStore store.DataStore) error {
	ctx, cancel := context.WithCancel(context.Background())
	channel, err := dataStore.GetJobs(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to start job scheduler")
	}
	l := log.FromContext(ctx)

	var msg interface{}
	concurrency := conf.GetInt(dconfig.SettingConcurrency)
	sem := make(chan bool, concurrency)
	quit := make(chan os.Signal)
	signal.Notify(quit, unix.SIGINT, unix.SIGTERM)
	for {
		select {
		case msg = <-channel:

		case <-quit:
			signal.Stop(quit)
			l.Info("Shutdown Worker ...")
			// Calling cancel() before returning should shut down
			// all workers. However, the new driver is not
			// particularly good at listening to the context in the
			// current state, but it'll be forced to shut down
			// eventually.
			cancel()
			return err
		}
		if msg == nil {
			break
		}
		switch msg.(type) {
		case *model.Job:
			job := msg.(*model.Job)
			sem <- true
			go func(ctx context.Context,
				job *model.Job, dataStore store.DataStore) {
				defer func() { <-sem }()
				processJob(ctx, job, dataStore, sem)
			}(ctx, job, dataStore)

		case error:
			cancel()
			return msg.(error)
		}
	}
	cancel()

	return err
}

func processJob(ctx context.Context, job *model.Job,
	dataStore store.DataStore, sem chan bool) error {
	sem <- true
	l := log.FromContext(ctx)
	workflow, err := dataStore.GetWorkflowByName(ctx, job.WorkflowName)
	if err != nil {
		l.Warnf("The workflow %q of job %s does not exist",
			job.WorkflowName, job.ID)
		// Mark the job as a failure
		dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
		return nil
	}

	aquiredJob, err := dataStore.AquireJob(ctx, job)
	if err != nil {
		l.Error(err.Error())
		return err
	} else if aquiredJob == nil {
		// Should "never" occur
		l.Warnf("The job with given ID (%s) does not exist", job.ID)
		return nil
	}

	job = aquiredJob
	l.Infof("%s: started, %s", job.ID, job.WorkflowName)

	for _, task := range workflow.Tasks {
		results, err := processTask(task, job, workflow)
		if err != nil {
			if results == nil {
				results = &model.TaskResult{}
			}
			results.Name = task.Name
			results.Error = err.Error()
		}
		err = dataStore.UpdateJobAddResult(ctx, job, results)
		if err != nil {
			l.Errorf("Error uploading results: %s", err.Error())
		}
	}

	err = dataStore.UpdateJobStatus(ctx, job, model.StatusDone)
	if err != nil {
		l.Warnf("Unable to set job status to done: %s", err.Error())
	}

	l.Infof("Job's done: %s", job.ID) // Yes mi lord!
	return nil
}

func processJobString(data string, workflow *model.Workflow, job *model.Job) string {
	for _, param := range job.InputParameters {
		data = strings.Replace(data,
			fmt.Sprintf("${workflow.input.%s}", param.Name),
			param.Value, 1)
	}

	return data
}

func processTask(task model.Task, job *model.Job,
	workflow *model.Workflow) (*model.TaskResult, error) {

	var result *model.TaskResult
	var err error

	switch task.Type {
	case "http":
		result, err = processHTTPTask(task, job, workflow)
	default:
		err = fmt.Errorf("Unrecognized task type: %s", task.Type)
	}
	if err != nil {
		if result == nil {
			result = &model.TaskResult{}
		}
		result.Name = task.Name
	}
	return result, err
}

// Processes a tash of type "http"
func processHTTPTask(task model.Task, job *model.Job,
	workflow *model.Workflow) (*model.TaskResult, error) {

	var httpTask model.HTTPTask
	results := model.HTTPResult{}
	results.Name = task.Name
	err := json.Unmarshal(task.Taskdef, &httpTask)
	if err != nil {
		results.Error = fmt.Errorf("Error: Task definition " +
			"incompatible with specified " +
			"type (http)").Error()
		return results.Marshal()
	}

	uri := processJobString(httpTask.URI, workflow, job)
	payloadString := processJobString(httpTask.Body, workflow, job)
	payload := strings.NewReader(payloadString)

	// Prepare request
	req, err := http.NewRequest(httpTask.Method, uri, payload)
	if err != nil {
		results.Error = "Internal error (500)"
		ret, _ := results.Marshal()
		return ret, err
	}

	// Prepare resquest headers
	var headersToBeSent []string
	for name, value := range httpTask.Headers {
		headerValue := processJobString(value, workflow, job)
		req.Header.Add(name, headerValue)
		headersToBeSent = append(headersToBeSent,
			fmt.Sprintf("%s: %s", name, headerValue))
	}
	var netClient = &http.Client{
		Timeout: time.Duration(httpTask.ReadTimeOut) * time.Second,
	}
	res, err := netClient.Do(req)
	if err != nil {
		results.Error = errors.Wrap(err,
			"Internal error performing request").Error()
		ret, _ := results.Marshal()
		return ret, err
	}

	defer res.Body.Close()
	// Build result response
	results.Response = model.HTTPResultResponse{
		StatusCode: res.StatusCode,
		Headers:    (map[string][]string)(res.Header.Clone()),
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		results.Error = errors.Wrap(err,
			"Error reading HTTP response body").Error()
	} else {
		results.Response.Body = string(resBody)
	}
	// Return generic TaskResult construct
	return results.Marshal()
}
