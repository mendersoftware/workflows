package workflow

// Workflow stores the definition of a workflow
type Workflow struct {
	Name            string
	Description     string
	Version         int
	SchemaVersion   int
	Tasks           []Task
	InputParameters []string
}

// Task stores the definition of a task within a workflow
type Task struct {
	Name string
	Type string
	HTTP HTTPParams
}

// HTTPParams stores the parameters of the HTTP calls for a WorkflowTask
type HTTPParams struct {
	URI               string
	Method            string
	Headers           map[string]string
	ConnectionTimeOut int
	ReadTimeOut       int
}

// Launch queues the workflow for execution
func (*Workflow) Launch(inputParameters map[string]string) error {
	return nil
}
