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
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/mendersoftware/workflows/model"
	"github.com/thedevsaddam/gojsonq"
)

const (
	workflowEnvVariable   = "env."
	workflowInputVariable = "workflow.input."
	regexVariable         = `\$\{([^\}]+)\}`
	regexOutputVariable   = `(.*)\.json\.(.*)`
)

var reExpression = regexp.MustCompile(regexVariable)
var reExpressionOutput = regexp.MustCompile(regexOutputVariable)

func processJobString(data string, workflow *model.Workflow, job *model.Job) string {
	matches := reExpression.FindAllStringSubmatch(data, -1)

	// search for ${...} expressions in the data string
	for _, submatch := range matches {
		// content of the ${...} expression, without the brackets
		match := submatch[1]
		if strings.HasPrefix(match, workflowInputVariable) && len(match) > len(workflowInputVariable) {
			// Replace ${workflow.input.KEY} with the KEY input variable
			paramName := match[len(workflowInputVariable):len(match)]
			for _, param := range job.InputParameters {
				if param.Name == paramName {
					data = strings.ReplaceAll(data, submatch[0], param.Value)
					break
				}
			}
		} else if strings.HasPrefix(match, workflowEnvVariable) && len(match) > len(workflowEnvVariable) {
			// Replace ${env.KEY} with the KEY environment variable
			envName := match[len(workflowEnvVariable):len(match)]
			envValue := os.Getenv(envName)
			data = strings.ReplaceAll(data, submatch[0], envValue)
		} else if output := reExpressionOutput.FindStringSubmatch(match); len(output) > 0 {
			// Replace ${TASK_NAME.json.JSONPATH} with the value of the JSONPATH expression from the
			// JSON output of the previous task with name TASK_NAME. If the output is not a valid JSON
			// or the JSONPATH does not resolve to a value, replace with empty string
			for _, result := range job.Results {
				if result.Name == output[1] {
					varKey := output[2]
					var output string
					if result.Type == model.TaskTypeHTTP {
						output = result.HTTPResponse.Body
					} else if result.Type == model.TaskTypeCLI {
						output = result.CLI.Output
					} else {
						continue
					}
					varValue := gojsonq.New().FromString(output).Find(varKey)
					if varValue == nil {
						varValue = ""
					}
					varValueString, err := ConvertAnythingToString(varValue)
					if err == nil {
						data = strings.ReplaceAll(data, submatch[0], varValueString)
					}
					break
				}
			}
		}
	}

	return data
}

func processJobStringOrFile(data string, workflow *model.Workflow, job *model.Job) (string, error) {
	data = processJobString(data, workflow, job)
	if strings.HasPrefix(data, "@") {
		filePath := data[1:]
		buffer, err := ioutil.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		data = processJobString(string(buffer), workflow, job)
	}
	return data, nil
}

// ConvertAnythingToString returns the string representation of anything
func ConvertAnythingToString(value interface{}) (string, error) {
	valueString, ok := value.(string)
	if !ok {
		valueBytes, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		valueString = string(valueBytes)
	}
	return valueString, nil
}
