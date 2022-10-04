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
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendersoftware/workflows/model"
)

type JobProcessor struct {
	job *model.Job
}

type JsonOptions struct {
}

func NewJobProcessor(job *model.Job) *JobProcessor {
	return &JobProcessor{
		job: job,
	}
}

func (j JobProcessor) ProcessJSON(
	data interface{},
	ps *JobStringProcessor,
	options ...*JsonOptions,
) interface{} {
	switch value := data.(type) {
	case []interface{}:
		result := make([]interface{}, len(value))
		for i, item := range value {
			result[i] = j.ProcessJSON(item, ps)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, item := range value {
			result[key] = j.ProcessJSON(item, ps)
		}
		return result
	case string:
		if len(value) > 3 && value[0:2] == "${" && value[len(value)-1:] == "}" {
			key := value[2 : len(value)-1]
			if strings.HasPrefix(key, workflowInputVariable) &&
				len(key) > len(workflowInputVariable) {
				key = key[len(workflowInputVariable):]
				for _, param := range j.job.InputParameters {
					if param.Name == key && param.Raw != nil {
						return j.ProcessJSON(param.Raw, ps)
					}
				}
				return nil
			}
		}
		return ps.ProcessJobString(value)
	case primitive.D:
		result := make(map[string]interface{})
		for key, item := range value.Map() {
			result[key] = j.ProcessJSON(item, ps)
		}
		return result
	case []primitive.D:
		result := make([]interface{}, len(value))
		for i, item := range value {
			result[i] = j.ProcessJSON(item, ps)
		}
		return result
	}
	return data
}
