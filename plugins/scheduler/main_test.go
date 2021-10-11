package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-agent/lib"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testConf = `
logActions: true
logIndexPrefix: unit-test
tasks:
  - name: test1
    command: "echo 'test1'"
  - name: test2
    command: "echo 'test2'"
schedule:
  - task: test1
    interval: "1s"
    instructions:
      muteOn:
        - 3
  - task: test2
    interval: "2s"
    instructions:
      retries: 4
      coolDown: 1
reactions:
  - ofTask: test1
    condition: status=error
    reaction: test2
    instructions:
      retries: 3
      coolDown: 2
  - ofTask: test2
    condition: rc=3
    reaction: test1
  - ofTask: test1
    condition: duration=10ms
    reaction: test2
  - ofTask: test2
    condition: stdout=test
    reaction: test1
  - ofTask: test1
    condition: stderr=foo
    reaction: test2
`
)

func TestScheduler(t *testing.T) {
	tmpdir, err := ioutil.TempDir(".", "scheduler_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, logger.Destroy())
	}()

	scheduler := (New(logger, func(e data.Event) {
		task, ok := e.Labels["task"].(lib.Task)
		require.True(t, ok)
		assert.Equal(t, "test2", task.Name)

		instr, ok := e.Labels["instructions"].(lib.ExecutionInstruction)
		require.True(t, ok)
		assert.Equal(t, lib.ExecutionInstruction{
			Retries:  1,
			CoolDown: 1,
		}, instr)
	})).(*TaskScheduler)

	t.Run("Test configuration", func(t *testing.T) {
		err = scheduler.Config([]byte(testConf))
		require.NoError(t, err)
		assert.Equal(t, true, scheduler.conf.LogActions)
		assert.Equal(t, "unit-test", scheduler.conf.LogIndexPrefix)
		assert.Equal(t, map[string]string{"test1": "1s", "test2": "2s"}, scheduler.schedule.GetSchedule())

		react := map[string][]lib.Reaction{
			"test1": []lib.Reaction{
				lib.Reaction{
					OfTask:       "test1",
					Condition:    "status=error",
					Reaction:     "test2",
					Instructions: lib.ExecutionInstruction{Retries: 3, CoolDown: 2},
				},
				lib.Reaction{
					OfTask:    "test1",
					Condition: "duration=10ms",
					Reaction:  "test2",
				},
				lib.Reaction{
					OfTask:    "test1",
					Condition: "stderr=foo",
					Reaction:  "test2",
				},
			},
			"test2": []lib.Reaction{
				lib.Reaction{
					OfTask:    "test2",
					Condition: "rc=3",
					Reaction:  "test1",
				},
				lib.Reaction{
					OfTask:    "test2",
					Condition: "stdout=test",
					Reaction:  "test1",
				},
			},
		}
		assert.Equal(t, react, scheduler.reactions)
	})

	t.Run("Test execution request", func(t *testing.T) {
		requestExec(scheduler, lib.ScheduleItem{
			Task:     "test2",
			Interval: "2s",
			Instructions: lib.ExecutionInstruction{
				Retries:  0,
				CoolDown: 1,
			},
		})

		requestExec(scheduler, lib.Reaction{
			OfTask:    "test1",
			Condition: "status=error",
			Reaction:  "test2",
			Instructions: lib.ExecutionInstruction{
				Retries:  0,
				CoolDown: 1,
			},
		})
	})
}
