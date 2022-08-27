// Copyright 2022 Northern.tech AS
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

package processor

import (
	"bytes"
	"net/url"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/thedevsaddam/gojsonq"

	"github.com/mendersoftware/workflows/model"
	"github.com/mendersoftware/workflows/utils"
)

const (
	workflowEnvVariable   = "env."
	workflowInputVariable = "workflow.input."
	regexVariable         = `\$\{([^\}\|]+)(?:\|([^\}]+))?}`
	regexVariableFlags    = `(?:([a-zA-Z]+)+=([a-zA-Z0-9]+);)([^\}\|]+)(?:\|([^\}]+))?`
	regexOutputVariable   = `(.*)\.json\.(.*)`
	encodingFlag          = "encoding"
	urlEncodingFlag       = "url"
	plainEncodingFlag     = "plain"
)

var reExpression = regexp.MustCompile(regexVariable)
var reExpressionOutput = regexp.MustCompile(regexOutputVariable)
var reExpressionFlags = regexp.MustCompile(regexVariableFlags)

type Encoding int64

const (
	Plain Encoding = iota
	URL
)

type JobStringProcessor struct {
	workflow *model.Workflow
	job      *model.Job
}

type Options struct {
	Encoding Encoding
}

func NewJobStringProcessor(
	workflow *model.Workflow,
	job *model.Job,
) *JobStringProcessor {
	return &JobStringProcessor{
		workflow: workflow,
		job:      job,
	}
}

func getOption(options ...*Options) Options {
	option := Options{Encoding: Plain}
	for _, o := range options {
		if o == nil {
			continue
		}
		option.Encoding = o.Encoding
		break
	}
	return option
}

func getEncodingFromData(data string, defaultEncoding Encoding) (string, Encoding) {
	const (
		expectedMatches      = 4
		flagMatchIndex       = 1
		flagValueMatchIndex  = 2
		identifierMatchIndex = 3
	)
	matches := reExpressionFlags.FindAllStringSubmatch(data, -1)
	for _, m := range matches {
		if len(m) < expectedMatches {
			continue
		}
		if m[flagMatchIndex] == encodingFlag {
			switch m[flagValueMatchIndex] {
			case urlEncodingFlag:
				return m[identifierMatchIndex], URL
			case plainEncodingFlag:
				return m[identifierMatchIndex], Plain
			}
		}
	}
	return data, defaultEncoding
}

func (j *JobStringProcessor) ProcessJobString(data string, options ...*Options) string {
	matches := reExpression.FindAllStringSubmatch(data, -1)
	option := getOption(options...)

	// search for ${...} expressions in the data string
	for _, submatch := range matches {
		// content of the ${...} expression, without the brackets
		match := submatch[1]
		// now it is possible to override the encoding with flags: ${encoding=plain;identifier}
		// if encoding is supplied via flags, it takes precedence, we return the match
		// without the flags, otherwise fail back to original match and encoding
		match, option.Encoding = getEncodingFromData(match, option.Encoding)
		defaultValue := submatch[2]
		if strings.HasPrefix(match, workflowInputVariable) &&
			len(match) > len(workflowInputVariable) {
			// Replace ${workflow.input.KEY} with the KEY input variable
			paramName := match[len(workflowInputVariable):]
			found := false
			for _, param := range j.job.InputParameters {
				if param.Name == paramName {
					value := param.Value
					if option.Encoding == URL {
						value = url.QueryEscape(value)
					}
					data = strings.ReplaceAll(data, submatch[0], value)
					found = true
					break
				}
			}
			if !found {
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
			for _, result := range j.job.Results {
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
					varValueString, err := utils.ConvertAnythingToString(varValue)
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

// MaybeExecuteGoTemplate tries to parse and execute data as a go template
// if it fails to do so, data is returned.
func (j *JobStringProcessor) MaybeExecuteGoTemplate(data string) string {
	input := j.job.InputParameters.Map()
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
