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

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
)

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

	acquiredJob, err := dataStore.AquireJob(ctx, job)
	if err != nil {
		l.Error(err.Error())
		return err
	} else if acquiredJob == nil {
		l.Warnf("The job with given ID (%s) does not exist", job.ID)
		err := dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
		if err != nil {
			return err
		}
		return nil
	}
	job = acquiredJob

	l.Infof("%s: started, %s", job.ID, job.WorkflowName)

	success := true
	for _, task := range workflow.Tasks {
		var result *model.TaskResult
		switch task.Type {
		case "http":
			var httpTask model.HTTPTask
			err := json.Unmarshal(task.Taskdef, &httpTask)
			if err != nil {
				err := dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
				if err != nil {
					return err
				}
				return fmt.Errorf(
					"Error: Task definition incompatible " +
						"with specified type (http)")
			}
			result, err = processHTTPTask(&httpTask, job, workflow)
			if err != nil {
				dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
				return err
			}
		case "cli":
			var cliTask model.CLITask
			err := json.Unmarshal(task.Taskdef, &cliTask)
			if err != nil {
				err := dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
				if err != nil {
					return err
				}
				return fmt.Errorf(
					"Error: Task definition incompatible " +
						"with specified type (cli)")
			}
			result, err = processCLITask(&cliTask, job, workflow)
			if err != nil {
				dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
				return err
			}
		}
		err = dataStore.UpdateJobAddResult(ctx, job, result)
		if err != nil {
			l.Errorf("Error uploading results: %s", err.Error())
		}
		if !result.Success {
			success = false
			break
		}
	}

	var newStatus string
	if success {
		err = dataStore.UpdateJobStatus(ctx, job, model.StatusDone)
		newStatus = "done"
	} else {
		err = dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
		newStatus = "failed"
	}
	if err != nil {
		l.Warn(fmt.Sprintf("Unable to set job status to %s", newStatus))
	}

	l.Infof("%s: done", job.ID)
	return nil
}
