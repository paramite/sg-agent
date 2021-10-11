package lib

import (
	"testing"

	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/stretchr/testify/assert"
)

func TestCreateLogEvent(t *testing.T) {
	task := Task{
		Name:    "test",
		Command: "exit",
	}
	job := Job{
		Execution: Execution{
			Task: task,
			Attempts: []ExecutionAttempt{
				ExecutionAttempt{
					ReturnCode: 3,
				},
			},
		},
	}

	t.Run("Test scheduler log event generation", func(t *testing.T) {
		generated := CreateLogEvent("foo", "bar", task)
		assert.Equal(t, FormatIndex("foo"), generated.Index)
		assert.Equal(t, data.LOG, generated.Type)
		assert.Equal(t, FormatPublisher("bar"), generated.Publisher)
		assert.Equal(t, data.INFO, generated.Severity)
		assert.Equal(t, map[string]interface{}{"name": "test", "command": "exit"}, generated.Labels)
		assert.Equal(t, "Scheduled task execution request submitted for execution.", generated.Message)
	})

	t.Run("Test executor log event generation", func(t *testing.T) {
		generated := CreateLogEvent("foo", "bar", &job)
		assert.Equal(t, FormatIndex("foo"), generated.Index)
		assert.Equal(t, data.LOG, generated.Type)
		assert.Equal(t, FormatPublisher("bar"), generated.Publisher)
		assert.Equal(t, data.INFO, generated.Severity)
		assert.Equal(t, map[string]interface{}{"name": "test", "command": "exit"}, generated.Labels)
		assert.Equal(t, "Task execution request fulfilled. 1. attempt -> RC: 3", generated.Message)
	})
}
