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
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mendersoftware/go-lib-micro/config"
	dconfig "github.com/mendersoftware/workflows/config"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
	"github.com/mendersoftware/workflows/workflow"
	"github.com/urfave/cli"
)

// Workflows maps active workflow names and Workflow structs
var Workflows map[string]*workflow.Workflow

// InitAndRun initializes the worker and runs it
func InitAndRun(conf config.Reader, dataStore store.DataStoreInterface) error {
	var workflowsPath string = conf.GetString(dconfig.SettingWorkflowsPath)
	if workflowsPath == "" {
		return cli.NewExitError(
			"Please specify the workflows path in the configuration file",
			1)
	}
	Workflows = workflow.GetWorkflowsFromPath(workflowsPath)

	ctx := context.Background()
	channel := dataStore.GetJobs(ctx)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("Shutdown Worker ...")
		dataStore.Shutdown()
	}()

	for {
		job := <-channel
		if job == nil {
			break
		}
		processJob(job, dataStore)
	}

	return nil
}

func processJob(job *model.Job, dataStore store.DataStoreInterface) {
	workflow := Workflows[job.WorkflowName]
	if workflow == nil {
		return
	}

	for _, task := range workflow.Tasks {
		if task.Type == "HTTP" {
			uri := processJobString(task.HTTP.URI, workflow, job)
			payloadString := processJobString(task.HTTP.Payload, workflow, job)
			payload := strings.NewReader(payloadString)

			req, _ := http.NewRequest(task.HTTP.Method, uri, payload)

			for name, value := range task.HTTP.Headers {
				req.Header.Add(name, value)
			}
			var netClient = &http.Client{
				Timeout: time.Duration(task.HTTP.ReadTimeOut) * time.Second,
			}
			res, err := netClient.Do(req)
			if err != nil {
				break
			}

			defer res.Body.Close()
			resBody, _ := ioutil.ReadAll(res.Body)

			fmt.Println(res)
			fmt.Println(string(resBody))
		}
	}
}

func processJobString(data string, workflow *workflow.Workflow, job *model.Job) string {
	for _, param := range job.InputParameters {
		data = strings.ReplaceAll(data, fmt.Sprintf("${workflow.input.%s}", param.Name), param.Value)
	}

	return data
}
