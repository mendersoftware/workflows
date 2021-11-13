// Copyright 2021 Northern.tech AS
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
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/thedevsaddam/gojsonq"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendersoftware/workflows/model"
)

const (
	workflowEnvVariable   = "env."
	workflowInputVariable = "workflow.input."
	regexVariable         = `\$\{([^\}\|]+)(?:\|([^\}]+))?}`
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
		defaultValue := submatch[2]
		if strings.HasPrefix(match, workflowInputVariable) &&
			len(match) > len(workflowInputVariable) {
			// Replace ${workflow.input.KEY} with the KEY input variable
			paramName := match[len(workflowInputVariable):]
			found := false
			for _, param := range job.InputParameters {
				if param.Name == paramName {
					value := param.Value
					data = strings.ReplaceAll(data, submatch[0], value)
					found = true
					break
				}
			}
			if !found && defaultValue != "" {
				data = strings.ReplaceAll(data, submatch[0], defaultValue)
			}
		} else if strings.HasPrefix(match, workflowEnvVariable) &&
			len(match) > len(workflowEnvVariable) {
			// Replace ${env.KEY} with the KEY environment variable
			envName := match[len(workflowEnvVariable):]
			envValue := os.Getenv(envName)
			if envValue == "" {
				envValue = defaultValue
			}
			data = strings.ReplaceAll(data, submatch[0], envValue)
		} else if output := reExpressionOutput.FindStringSubmatch(match); len(output) > 0 {
			// Replace ${TASK_NAME.json.JSONPATH} with the value of the JSONPATH expression from the
			// JSON output of the previous task with name TASK_NAME. If the output is not a valid
			// JSON or the JSONPATH does not resolve to a value, replace with empty string
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
						if varValueString == "" {
							varValueString = defaultValue
						}
						data = strings.ReplaceAll(data, submatch[0], varValueString)
					}
					break
				}
			}
		}
	}

	return data
}

// maybeExecuteGoTemplate tries to parse and execute data as a go template
// if it fails to do so, data is returned.
func maybeExecuteGoTemplate(data string, input map[string]interface{}) string {
	tmpl, err := template.New("go-template").Parse(data)
	if err != nil {
		return data
	}
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, input)
	if err != nil {
		return data
	}
	return buf.String()
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

func processJobJSON(data interface{}, workflow *model.Workflow, job *model.Job) interface{} {
	switch value := data.(type) {
	case []interface{}:
		result := make([]interface{}, len(value))
		for i, item := range value {
			result[i] = processJobJSON(item, workflow, job)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, item := range value {
			result[key] = processJobJSON(item, workflow, job)
		}
		return result
	case string:
		if len(value) > 3 && value[0:2] == "${" && value[len(value)-1:] == "}" {
			key := value[2 : len(value)-1]
			if strings.HasPrefix(key, workflowInputVariable) &&
				len(key) > len(workflowInputVariable) {
				key = key[len(workflowInputVariable):]
				for _, param := range job.InputParameters {
					if param.Name == key && param.Raw != nil {
						return processJobJSON(param.Raw, workflow, job)
					}
				}
				return nil
			}
		}
		return processJobString(value, workflow, job)
	case primitive.D:
		result := make(map[string]interface{})
		for key, item := range value.Map() {
			result[key] = processJobJSON(item, workflow, job)
		}
		return result
	case []primitive.D:
		result := make([]interface{}, len(value))
		for i, item := range value {
			result[i] = processJobJSON(item, workflow, job)
		}
		return result
	}
	return data
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
