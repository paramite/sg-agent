package lib

import (
	"testing"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionStatus(t *testing.T) {
	status := ExecutionStatus(2)

	t.Run("Test methods of ExecutionStatus", func(t *testing.T) {
		assert.Equal(t, []string{"success", "warning", "error"}, status.List())
		assert.Equal(t, "error", status.String())
		assert.Equal(t, data.CRITICAL, status.ToSeverity())

		require.True(t, status.SetFromString("warning"))
		assert.Equal(t, data.WARNING, status.ToSeverity())
		require.True(t, status.SetFromString("success"))
		assert.Equal(t, data.INFO, status.ToSeverity())
	})
}

func TestReaction(t *testing.T) {
	reaction := Reaction{
		OfTask:    "test1",
		Condition: "status=error",
		Reaction:  "test2",
		Instructions: ExecutionInstruction{
			Timeout:  0,
			Retries:  2,
			CoolDown: 0,
			MuteOn:   []int{1, 2, 3},
		},
	}

	execution := Execution{
		Task: Task{
			Name:    "test",
			Command: "test",
		},
		Requested: 0,
		Requestor: "test",
		Executor:  "test",
		Attempts: []ExecutionAttempt{
			ExecutionAttempt{
				Executed:   1,
				Duration:   0.1,
				ReturnCode: 1,
				StdOut:     "test",
				StdErr:     "test",
			},
		},
		Status: "warning",
	}

	t.Run("Test methods of Reaction", func(t *testing.T) {
		// test status condition
		assert.Equal(t, false, reaction.Required(execution))
		execution.Status = "error"
		assert.Equal(t, true, reaction.Required(execution))
		// test RC condition
		reaction.Condition = "rc=2"
		assert.Equal(t, false, reaction.Required(execution))
		(execution.Attempts[0]).ReturnCode = 2
		assert.Equal(t, true, reaction.Required(execution))
		reaction.Condition = "rc=woof"
		assert.Equal(t, false, reaction.Required(execution))
		// test duration condition
		reaction.Condition = "duration=2s"
		assert.Equal(t, false, reaction.Required(execution))
		(execution.Attempts[0]).Duration = (3 * time.Second).Seconds()
		assert.Equal(t, true, reaction.Required(execution))
		reaction.Condition = "duration=woof"
		assert.Equal(t, false, reaction.Required(execution))
		// test stdout condition
		reaction.Condition = "stdout=lubba"
		assert.Equal(t, false, reaction.Required(execution))
		(execution.Attempts[0]).StdOut = "Wubba lubba dub dub"
		assert.Equal(t, true, reaction.Required(execution))
		reaction.Condition = "stdout=]["
		assert.Equal(t, false, reaction.Required(execution))
		// test stderr condition
		reaction.Condition = "stderr=dub"
		assert.Equal(t, false, reaction.Required(execution))
		(execution.Attempts[0]).StdErr = "Wubba lubba dub dub"
		assert.Equal(t, true, reaction.Required(execution))
		reaction.Condition = "stderr=]["
		assert.Equal(t, false, reaction.Required(execution))
		// test invalid condition
		reaction.Condition = "foo=bar"
		assert.Equal(t, false, reaction.Required(execution))
	})
}
