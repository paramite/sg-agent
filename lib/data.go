package lib

import (
	"strings"

	"github.com/infrawatch/sg-core/pkg/data"
)

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

// SetFromString resets value according to given human readable identification. Returns false if invalid identification was given.
func (es *ExecutionStatus) SetFromString(input string) bool {
	var ok bool
	*es, ok = (map[string]ExecutionStatus{"success": SUCCESS, "warning": WARNING, "error": ERROR})[strings.ToLower(input)]
	return ok
}

// ToSeverity converts execution status to sg-core event severity
func (es ExecutionStatus) ToSeverity() data.EventSeverity {
	return map[ExecutionStatus]data.EventSeverity{
		SUCCESS: data.INFO,
		WARNING: data.WARNING,
		ERROR:   data.CRITICAL,
	}[es]
}

// Task holds data about commands to be run either as scheduled task or reaction task
type Task struct {
	Name    string `validate:"required"`
	Command string `validate:"required"`
}

// ExecutionInstruction hold instructions for job execution
type ExecutionInstruction struct {
	Timeout int
	// how many times to retry and how long to wait before next try and task run timeout
	Retries  int
	CoolDown int `yaml:"coolDown"`
	// which return codes should not be considered as task failure but as warning only
	MuteOn []int `yaml:"muteOn"`
}

// ScheduleItem holds task execution schedule
type ScheduleItem struct {
	Task         string               `validate:"required"`
	Interval     string               `validate:"required"`
	Instructions ExecutionInstruction `validate:"dive"`
}

// Reaction holds information on which task result sg-agent should react and by which tasks
// execution should be reacted
type Reaction struct {
	OnStatus     ExecutionStatus `yaml:"onStatus"`
	OnReturnCode int
	OfTask       string `yaml:"ofTask" validate:"required"`
	ReactionTask string
	Timeout      int
	// how many times to retry and how long to wait before next try and task run timeout
	Retries  int
	CoolDown int `yaml:"coolDown"`
}

// ExecutionAttempt holds data about command
type ExecutionAttempt struct {
	Executed   float64
	Duration   float64
	ReturnCode int    `yaml:"returnCode"`
	StdOut     string `yaml:"stdout"`
	StdErr     string `yaml:"stderr"`
}

// Execution holds data gathered after execution of the task by executor plugin
type Execution struct {
	Task      Task
	Requested float64
	Requestor string
	Executor  string
	Attempts  []ExecutionAttempt
	Status    string
}

// Job is used for following actual command run
type Job struct {
	Execution    Execution
	Instructions ExecutionInstruction
	Cancel       func()
}
