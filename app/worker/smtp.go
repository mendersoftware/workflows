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
	"github.com/pkg/errors"
	"io/ioutil"
	"net/smtp"
	"strings"

	"github.com/mendersoftware/go-lib-micro/config"
	dconfig "github.com/mendersoftware/workflows/config"
	"github.com/mendersoftware/workflows/model"
)

var smtpClient SMTPClientInterface = new(SMTPClient)

func processSMTPTask(smtpTask *model.SMTPTask, job *model.Job,
	workflow *model.Workflow) (*model.TaskResult, error) {
	var result *model.TaskResult = &model.TaskResult{
		SMTP: &model.TaskResultSMTP{},
	}

	recipients := make([]string, 0, 10)

	to := make([]string, 0, 10)
	for _, address := range smtpTask.To {
		address := processJobString(address, workflow, job)
		recipients = append(recipients, address)
		to = append(to, address)
	}

	cc := make([]string, 0, 10)
	for _, address := range smtpTask.Cc {
		address := processJobString(address, workflow, job)
		recipients = append(recipients, address)
		cc = append(to, address)
	}

	for _, address := range smtpTask.Bcc {
		address := processJobString(address, workflow, job)
		recipients = append(recipients, address)
	}

	from := processJobString(smtpTask.From, workflow, job)
	subject := processJobString(smtpTask.Subject, workflow, job)
	body := processJobString(smtpTask.Body, workflow, job)
	if strings.HasPrefix(body,"@") {
		filePath:=body[1:]
		buffer,err:=ioutil.ReadFile(filePath)
		if err!=nil {
			result.Success = false
			result.SMTP.Error = err.Error()
			return result, errors.Wrap(err,"cant load file " + filePath)
		}
		body = processJobString(string(buffer), workflow, job)
	}

	msg := []byte("From: " + from + "\r\n" +
		"To: " + strings.Join(to, ", ") + "\r\n" +
		"Cc: " + strings.Join(cc, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	result.SMTP.Sender = from
	result.SMTP.Recipients = recipients
	result.SMTP.Message = string(msg)

	// Set up authentication information
	smtpHostname := config.Config.GetString(dconfig.SettingSMTPHost)
	smtpUsername := config.Config.GetString(dconfig.SettingSMTPUsername)
	smtpPassword := config.Config.GetString(dconfig.SettingSMTPPassword)
	smtpAuthMechanism := config.Config.GetString(dconfig.SettingSMTPAuthMechanism)
	var auth smtp.Auth
	if smtpAuthMechanism == "CRAM-MD5" {
		auth = smtp.CRAMMD5Auth(smtpUsername, smtpPassword)
	} else {
		auth = smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHostname)
	}

	err := smtpClient.SendMail(smtpHostname, auth, from, recipients, msg)
	if err != nil {
		result.Success = false
		result.SMTP.Error = err.Error()
	} else {
		result.Success = true
	}

	return result, nil
}
