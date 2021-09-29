package lib

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/go-playground/validator.v9"

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

func (es ExecutionStatus) List() []string {
	return []string{"success", "warning", "error"}
}

func (es ExecutionStatus) String() string {
	return (es.List())[es]
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

// TODO: remove once available in apputils
func intervalToDuration(interval string) (time.Duration, error) {
	var out time.Duration
	intervalRegex := regexp.MustCompile(`(\d*)([smhd])`)

	if match := intervalRegex.FindStringSubmatch(interval); match != nil {
		var units time.Duration
		switch match[2] {
		case "s":
			units = time.Second
		case "m":
			units = time.Minute
		case "h":
			units = time.Hour
		case "d":
			units = time.Hour * 24
		default:
			return out, fmt.Errorf("invalid interval units (%s)", match[2])
		}
		num, err := strconv.Atoi(match[1])
		if err != nil {
			return out, fmt.Errorf("invalid interval value (%s): %s", match[3], err)
		}
		out = time.Duration(int64(num) * int64(units))
	} else {
		return out, fmt.Errorf("invalid interval value (%s)", interval)
	}

	return out, nil
}

func conditionValidator(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	parts := strings.Split(value, "=")
	if len(parts) != 2 {
		return false
	}

	if parts[0] == "status" {
		for _, cond := range (ExecutionStatus(0)).List() {
			if parts[1] == cond {
				return true
			}
		}
	}

	if parts[0] == "rc" {
		if _, err := strconv.Atoi(parts[1]); err == nil {
			return true
		}
	}

	if parts[0] == "duration" {
		if _, err := intervalToDuration(parts[1]); err == nil {
			return true
		}
	}

	for _, cond := range []string{"stdout=", "stderr="} {
		if strings.HasPrefix(value, cond) {
			return true
		}
	}

	return false
}

// Reaction holds information on which task result sg-agent should react and by which tasks
// execution should be reacted
type Reaction struct {
	OfTask       string `yaml:"ofTask" validate:"required"`
	Condition    string `validate:"condition"`
	Reaction     string
	Instructions ExecutionInstruction
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
