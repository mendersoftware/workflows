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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/mendersoftware/workflows/model"
)

var makeHTTPRequest = func(req *http.Request, timeout time.Duration) (*http.Response, error) {
	var httpClient = &http.Client{
		Timeout: timeout,
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
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
	res, err := makeHTTPRequest(req, time.Duration(httpTask.ReadTimeOut)*time.Second)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	resBody, _ := ioutil.ReadAll(res.Body)

	var success bool
	if len(httpTask.StatusCodes) == 0 {
		success = true
	} else {
		success = false
		for _, statusCode := range httpTask.StatusCodes {
			if statusCode == res.StatusCode {
				success = true
				break
			}
		}
	}

	result := &model.TaskResult{
		Success: success,
		HTTPRequest: model.TaskResultHTTPRequest{
			URI:     uri,
			Method:  httpTask.Method,
			Body:    payloadString,
			Headers: headersToBeSent,
		},
		HTTPResponse: model.TaskResultHTTPResponse{
			StatusCode: res.StatusCode,
			Body:       string(resBody),
		},
	}

	return result, nil
}
