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
	"context"
	"fmt"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store"
)

func processJob(ctx context.Context, job *model.Job,
	dataStore store.DataStore) error {
	l := log.FromContext(ctx)

	workflow, err := dataStore.GetWorkflowByName(ctx, job.WorkflowName)
	if err != nil {
		l.Warnf("The workflow %q of job %s does not exist",
			job.WorkflowName, job.ID)
		err := dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
		if err != nil {
			return err
		}
		return nil
	}

	acquiredJob, err := dataStore.AcquireJob(ctx, job)
	if err != nil {
		l.Error(err.Error())
		err := dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
		if err != nil {
			return err
		}
		return nil
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
		l.Infof("%s: started, %s task :%s", job.ID, job.WorkflowName, task.Name)
		result, err := processTask(task, job, workflow, l)
		if err != nil {
			dataStore.UpdateJobStatus(ctx, job, model.StatusFailure)
			if err != nil {
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
		return err
	}

	l.Infof("%s: done", job.ID)
	return nil
}

func processTask(task model.Task, job *model.Job,
	workflow *model.Workflow, l *log.Logger) (*model.TaskResult, error) {

	switch task.Type {
	case model.TaskTypeHTTP:
		var httpTask *model.HTTPTask = task.HTTP
		if httpTask == nil {
			return nil, fmt.Errorf(
				"Error: Task definition incompatible " +
					"with specified type (http)")
		}
		l.Infof("processTask: calling http task: %s %s", httpTask.Method, httpTask.URI)
		result, err := processHTTPTask(httpTask, job, workflow, l)
		return result, err
	case model.TaskTypeCLI:
		var cliTask *model.CLITask = task.CLI
		if cliTask == nil {
			return nil, fmt.Errorf(
				"Error: Task definition incompatible " +
					"with specified type (cli)")
		}
		result, err := processCLITask(cliTask, job, workflow)
		return result, err
	case model.TaskTypeSMTP:
		var smtpTask *model.SMTPTask = task.SMTP
		if smtpTask == nil {
			return nil, fmt.Errorf(
				"Error: Task definition incompatible " +
					"with specified type (smtp)")
		}
		l.Infof("processTask: calling smtp task: From: %s To: %s Subject: %s",
			smtpTask.From, smtpTask.To, smtpTask.Subject)
		result, err := processSMTPTask(smtpTask, job, workflow, l)
		return result, err
	default:
		err := fmt.Errorf("Unrecognized task type: %s", task.Type)
		return nil, err
	}
}
