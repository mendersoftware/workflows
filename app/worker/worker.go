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
	"syscall"
	"time"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"
	dconfig "github.com/mendersoftware/workflows/config"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
)

// InitAndRun initializes the worker and runs it
func InitAndRun(conf config.Reader, dataStore store.DataStore) error {
	ctx := context.Background()
	channel := dataStore.GetJobs(ctx)
	l := log.FromContext(ctx)

	var job *model.Job
	concurrency := conf.GetInt(dconfig.SettingConcurrency)
	sem := make(chan bool, concurrency)
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case job = <-channel:

		case <-quit:
			l.Info("Shutdown Worker ...")
			dataStore.Shutdown()
			return nil
		}
		if job == nil {
			break
		}
		sem <- true
		go func(ctx context.Context, job *model.Job, dataStore store.DataStore) {
			defer func() { <-sem }()
			processJob(ctx, job, dataStore)
		}(ctx, job, dataStore)
	}

	return nil
}

func processJob(ctx context.Context, job *model.Job,
	dataStore store.DataStore) error {

	l := log.FromContext(ctx)
	workflow, err := dataStore.GetWorkflowByName(job.WorkflowName)
	if err != nil {
		l.Warnf("The workflow %q of job %s does not exist",
			job.WorkflowName, job.ID)
		err := dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
		if err != nil {
			return err
		}
		return nil
	}

	job, err = dataStore.AquireJob(ctx, job)
	if err != nil {
		l.Error(err.Error())
		return err
	} else if job == nil {
		l.Warnf("The job with given ID (%s) does not exist", job.ID)
		err := dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
		if err != nil {
			return err
		}
		return nil
	}

	l.Infof("%s: started, %s", job.ID, job.WorkflowName)

	for _, task := range workflow.Tasks {
		switch task.Type {
		case "http":
			var httpTask model.HTTPTask
			err := json.Unmarshal(task.Taskdef, &httpTask)
			if err != nil {
				return fmt.Errorf(
					"Error: Task definition incompatible " +
						"with specified type (http)")
			}
			results, err := processHTTPTask(&httpTask, job, workflow)
			if err != nil {
				dataStore.UpdateJobStatus(ctx, job,
					model.StatusFailure)
				return err
			}
			err = dataStore.UpdateJobAddResult(ctx, job, results)
			if err != nil {
				l.Errorf("Error uploading results: %s", err.Error())
			}
		}
	}

	err = dataStore.UpdateJobStatus(ctx, job, model.StatusDone)
	if err != nil {
		l.Warn("Unable to set job status to done")
	}

	l.Infof("%s: done", job.ID)
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

func processHTTPTask(httpTask *model.HTTPTask, job *model.Job,
	workflow *model.Workflow) (*model.TaskResult, error) {
	uri := processJobString(httpTask.URI, workflow, job)
	payloadString := processJobString(httpTask.Body, workflow, job)
	payload := strings.NewReader(payloadString)

	req, err := http.NewRequest(httpTask.Method, uri, payload)
	if err != nil {
		return nil, err
	}

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
		return nil, err
	}

	defer res.Body.Close()
	resBody, _ := ioutil.ReadAll(res.Body)

	result := &model.TaskResult{
		Request: model.TaskResultRequest{
			URI:     uri,
			Method:  httpTask.Method,
			Body:    payloadString,
			Headers: headersToBeSent,
		},
		Response: model.TaskResultResponse{
			StatusCode: res.StatusCode,
			Body:       string(resBody),
		},
	}

	return result, nil
}
