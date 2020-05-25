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
	"bytes"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"
	dconfig "github.com/mendersoftware/workflows/config"
	"github.com/mendersoftware/workflows/model"
)

var smtpClient SMTPClientInterface = new(SMTPClient)

func processSMTPTask(smtpTask *model.SMTPTask, job *model.Job,
	workflow *model.Workflow, l *log.Logger) (*model.TaskResult, error) {
	var result *model.TaskResult = &model.TaskResult{
		SMTP: &model.TaskResultSMTP{},
	}

	l.Debugf("processSMTPTask starting")
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

	var err error
	var body, HTML string
	if body, err = processJobStringOrFile(smtpTask.Body, workflow, job); err != nil {
		result.Success = false
		result.SMTP.Error = err.Error()
		l.Infof("processSMTPTask error reading file: '%s'", err.Error())
		return result, err
	}
	l.Debugf("processSMTPTask body text: '\n%s\n'", body)

	if HTML, err = processJobStringOrFile(smtpTask.HTML, workflow, job); err != nil {
		result.Success = false
		result.SMTP.Error = err.Error()
		l.Infof("processSMTPTask error reading file: '%s'", err.Error())
		return result, err
	}
	l.Debugf("processSMTPTask body HTML: '\n%s\n'", HTML)

	altContent := &bytes.Buffer{}
	altWriter := multipart.NewWriter(altContent)
	if HTML != "" {
		childContent, _ := altWriter.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/html"}})
		childContent.Write([]byte(HTML))
	}
	if body != "" {
		childContent, _ := altWriter.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain"}})
		childContent.Write([]byte(body))
	}
	altWriter.Close()

	msgBuffer := &bytes.Buffer{}
	msgBuffer.WriteString("From: " + from + "\r\n" +
		"To: " + strings.Join(to, ", ") + "\r\n" +
		"Cc: " + strings.Join(cc, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/alternative; boundary=" + altWriter.Boundary() + "\r\n" +
		"\r\n" +
		altContent.String())

	result.SMTP.Sender = from
	result.SMTP.Recipients = recipients
	result.SMTP.Message = msgBuffer.String()

	// Set up authentication information
	smtpHostname := config.Config.GetString(dconfig.SettingSMTPHost)
	smtpUsername := config.Config.GetString(dconfig.SettingSMTPUsername)
	smtpPassword := config.Config.GetString(dconfig.SettingSMTPPassword)
	smtpAuthMechanism := config.Config.GetString(dconfig.SettingSMTPAuthMechanism)
	host, _, _ := net.SplitHostPort(smtpHostname)
	var auth smtp.Auth
	if smtpUsername != "" {
		if smtpAuthMechanism == "CRAM-MD5" {
			auth = smtp.CRAMMD5Auth(smtpUsername, smtpPassword)
		} else {
			auth = smtp.PlainAuth("", smtpUsername, smtpPassword, host)
		}
	}

	err = smtpClient.SendMail(smtpHostname, auth, from, recipients, msgBuffer.Bytes())
	l.Debugf("processSMTPTask: smtpClient.SendMail returned %v", err)
	if err != nil {
		l.Errorf("processSMTPTask: smtpClient.SendMail returned %v", err)
		result.Success = false
		result.SMTP.Error = err.Error()
	} else {
		l.Infof("processSMTPTask: email successfully sent to %v", recipients)
		result.Success = true
	}

	return result, nil
}
