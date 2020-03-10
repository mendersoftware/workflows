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
	"io/ioutil"
	"net/smtp"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	mocklib "github.com/stretchr/testify/mock"

	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/store/mock"
)

func TestProcessJobSMTP(t *testing.T) {
	var mockedSMTPClient = new(SMTPClientMock)
	var originalSMTPClient = smtpClient
	smtpClient = mockedSMTPClient

	mockedSMTPClient.On("SendMail",
		"",
		mocklib.MatchedBy(
			func(_ smtp.Auth) bool {
				return true
			}),
		"no-reply@mender.io",
		[]string{
			"user@mender.io",
			"support@mender.io",
			"archive@mender.io",
		},
		mocklib.MatchedBy(
			func(_ []byte) bool {
				return true
			}),
	).Return(nil)

	ctx := context.Background()
	dataStore := mock.NewDataStore()

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeSMTP,
				SMTP: &model.SMTPTask{
					From:    "no-reply@mender.io",
					To:      []string{"user@mender.io"},
					Cc:      []string{"support@mender.io"},
					Bcc:     []string{"archive@mender.io"},
					Subject: "Subject",
					Body:    "Body",
				},
			},
		},
	}

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}

	dataStore.On("GetWorkflowByName",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		workflow.Name,
	).Return(workflow, nil)

	dataStore.On("AcquireJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
	).Return(job, nil)

	dataStore.On("UpdateJobStatus",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		model.StatusDone,
	).Return(nil)

	dataStore.On("UpdateJobAddResult",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		mocklib.MatchedBy(
			func(taskResult *model.TaskResult) bool {
				assert.True(t, taskResult.Success)
				assert.Equal(t, workflow.Tasks[0].SMTP.From, taskResult.SMTP.Sender)
				assert.Equal(t, "", taskResult.SMTP.Error)

				return true
			}),
	).Return(nil)

	err := processJob(ctx, job, dataStore)

	assert.Nil(t, err)

	dataStore.AssertExpectations(t)
	mockedSMTPClient.AssertExpectations(t)

	smtpClient = originalSMTPClient
}

func TestProcessJobSMTPLoadFromFile(t *testing.T) {
	var mockedSMTPClient = new(SMTPClientMock)
	var originalSMTPClient = smtpClient
	smtpClient = mockedSMTPClient

	mockedSMTPClient.On("SendMail",
		"",
		mocklib.MatchedBy(
			func(_ smtp.Auth) bool {
				return true
			}),
		"no-reply@mender.io",
		[]string{
			"user@mender.io",
			"support@mender.io",
			"archive@mender.io",
		},
		mocklib.MatchedBy(
			func(_ []byte) bool {
				return true
			}),
	).Return(nil)

	ctx := context.Background()
	dataStore := mock.NewDataStore()
	tmpFile,err := ioutil.TempFile("","mail.body")
	assert.Nil(t, err)

	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte("Hello\n\n This is the TestProcessJobSMTPLoadFromFile" +
		"sedning greetings.\n\nTestProcessJobSMTPLoadFromFile"))
	assert.Nil(t, err)

	err = tmpFile.Close()
	assert.Nil(t, err)

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeSMTP,
				SMTP: &model.SMTPTask{
					From:    "no-reply@mender.io",
					To:      []string{"user@mender.io"},
					Cc:      []string{"support@mender.io"},
					Bcc:     []string{"archive@mender.io"},
					Subject: "Subject",
					Body:    "@" + tmpFile.Name(),
				},
			},
		},
	}

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}

	dataStore.On("GetWorkflowByName",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		workflow.Name,
	).Return(workflow, nil)

	dataStore.On("AcquireJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
	).Return(job, nil)

	dataStore.On("UpdateJobStatus",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		model.StatusDone,
	).Return(nil)

	dataStore.On("UpdateJobAddResult",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		mocklib.MatchedBy(
			func(taskResult *model.TaskResult) bool {
				assert.True(t, taskResult.Success)
				assert.Equal(t, workflow.Tasks[0].SMTP.From, taskResult.SMTP.Sender)
				assert.Equal(t, "", taskResult.SMTP.Error)

				return true
			}),
	).Return(nil)

	err = processJob(ctx, job, dataStore)

	assert.Nil(t, err)

	dataStore.AssertExpectations(t)
	mockedSMTPClient.AssertExpectations(t)

	smtpClient = originalSMTPClient
}

func TestProcessJobSMTPLoadFromFileFailed(t *testing.T) {
	var mockedSMTPClient = new(SMTPClientMock)
	var originalSMTPClient = smtpClient
	smtpClient = mockedSMTPClient

	ctx := context.Background()
	dataStore := mock.NewDataStore()

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeSMTP,
				SMTP: &model.SMTPTask{
					From:    "no-reply@mender.io",
					To:      []string{"user@mender.io"},
					Cc:      []string{"support@mender.io"},
					Bcc:     []string{"archive@mender.io"},
					Subject: "Subject",
					Body:    "@/this/file/does/not/exits/for/sure",
				},
			},
		},
	}

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}

	dataStore.On("GetWorkflowByName",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		workflow.Name,
	).Return(workflow, nil)

	dataStore.On("AcquireJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
	).Return(job, nil)

	dataStore.On("UpdateJobStatus",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		model.StatusFailure,
	).Return(nil)

	err := processJob(ctx, job, dataStore)

	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "cant load file /this/file/does/not/exits/for/sure:" +
		" open /this/file/does/not/exits/for/sure: no such file or directory")

	dataStore.AssertExpectations(t)
	mockedSMTPClient.AssertExpectations(t)

	smtpClient = originalSMTPClient
}

func TestProcessJobSMTPFailure(t *testing.T) {
	var mockedSMTPClient = new(SMTPClientMock)
	var originalSMTPClient = smtpClient
	smtpClient = mockedSMTPClient

	smtpError := errors.New("smtp error")

	mockedSMTPClient.On("SendMail",
		"",
		mocklib.MatchedBy(
			func(_ smtp.Auth) bool {
				return true
			}),
		"no-reply@mender.io",
		[]string{
			"user@mender.io",
			"support@mender.io",
			"archive@mender.io",
		},
		mocklib.MatchedBy(
			func(_ []byte) bool {
				return true
			}),
	).Return(smtpError)

	ctx := context.Background()
	dataStore := mock.NewDataStore()

	workflow := &model.Workflow{
		Name: "test",
		Tasks: []model.Task{
			{
				Name: "task_1",
				Type: model.TaskTypeSMTP,
				SMTP: &model.SMTPTask{
					From:    "no-reply@mender.io",
					To:      []string{"user@mender.io"},
					Cc:      []string{"support@mender.io"},
					Bcc:     []string{"archive@mender.io"},
					Subject: "Subject",
					Body:    "Body",
				},
			},
		},
	}

	job := &model.Job{
		WorkflowName: workflow.Name,
		Status:       model.StatusPending,
	}

	dataStore.On("GetWorkflowByName",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		workflow.Name,
	).Return(workflow, nil)

	dataStore.On("AcquireJob",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
	).Return(job, nil)

	dataStore.On("UpdateJobStatus",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		model.StatusFailure,
	).Return(nil)

	dataStore.On("UpdateJobAddResult",
		mocklib.MatchedBy(
			func(_ context.Context) bool {
				return true
			}),
		job,
		mocklib.MatchedBy(
			func(taskResult *model.TaskResult) bool {
				assert.False(t, taskResult.Success)
				assert.Equal(t, smtpError.Error(), taskResult.SMTP.Error)

				return true
			}),
	).Return(nil)

	err := processJob(ctx, job, dataStore)

	assert.Nil(t, err)

	dataStore.AssertExpectations(t)
	mockedSMTPClient.AssertExpectations(t)

	smtpClient = originalSMTPClient
}
