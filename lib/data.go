package lib

import "os/exec"

// ExecutionStatus represents final status of the execution
type ExecutionStatus int

// All possible statuses
const (
	SUCCESS ExecutionStatus = iota
	WARNING
	ERROR
)

func (es ExecutionStatus) String() string {
	return []string{"success", "warning", "error"}[es]
}

// TaskRequest holds data created for executor plugin by scheduler plugin
type TaskRequest struct {
	Name     string `validate:"required"`
	Command  string `validate:"required"`
	Interval string `validate:"required"`
	Timeout  int
	// which return codes should not be considered as task failure but as warning only
	MuteOn []int `yaml:"muteOn"`
	// how many times to retry and how long to wait before next try
	Retries  int
	CoolDown int `yaml:"coolDown"`
	// submit request internally or on bus through transport
	Type string `validate:"oneof=internal external"`
}

// ExecutionAttempt holds data about command
type ExecutionAttempt struct {
	Executed   float64
	Duration   float64
	ReturnCode int    `yaml:"returnCode"`
	StdOut     string `yaml:"stdout"`
	StdErr     string `yaml:"stderr"`
}

// TaskExecution holds data gathered after execution of the task by executor plugin
type TaskExecution struct {
	Request   TaskRequest
	Requestor string
	Requested float64
	Executor  string
	Attempts  []ExecutionAttempt
	Status    ExecutionStatus
}

// Task is used for following actual command run
type Task struct {
	Execution TaskExecution
	Command   *exec.Cmd
	Cancel    func()
}
