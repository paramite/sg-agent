package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-agent/lib"
	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testConf = `
logActions: true
logIndexPrefix: unit-test
workDirectory: %s
shellPath: /bin/sh
workers: 5
`
	testScript = `#!/bin/sh
echo test
`
)

func TestExecutor(t *testing.T) {
	tmpdir, err := ioutil.TempDir(".", "executor_test_tmp")
	require.NoError(t, err)
	workdir := path.Join(tmpdir, "sg-agent")
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, logger.Destroy())
	}()

	ebus := bus.EventBus{}
	executor := (New(logger, ebus.Publish)).(*Executor)

	t.Run("Test configuration", func(t *testing.T) {
		_, err := os.Stat(workdir)
		assert.True(t, os.IsNotExist(err))

		err = executor.Config([]byte(fmt.Sprintf(testConf, workdir)))
		require.NoError(t, err)

		assert.Equal(t, true, executor.conf.LogActions)
		assert.Equal(t, "unit-test", executor.conf.LogIndexPrefix)
		assert.Equal(t, workdir, executor.conf.WorkDirectory)
		assert.Equal(t, "/bin/sh", executor.conf.ShellPath)
		assert.Equal(t, 5, executor.conf.Workers)

		// check tmpdir creation
		stat, err := os.Stat(workdir)
		assert.NoError(t, err)
		assert.True(t, stat.IsDir())
	})

	t.Run("Test script creation", func(t *testing.T) {
		task := lib.Task{Name: "test", Command: "echo test"}
		script, err := executor.getScript(&task)
		require.NoError(t, err)

		// script is created in workdir under script subdirectory
		assert.Regexp(t, fmt.Sprintf("^%s/script", executor.conf.WorkDirectory), script)

		_, err = os.Stat(script)
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(script)
		require.NoError(t, err)
		assert.Equal(t, testScript, string(data))
	})

	t.Run("Test job execution with timeout", func(t *testing.T) {
		job := lib.Job{
			Execution: lib.Execution{
				Task: lib.Task{
					Name:    "test",
					Command: "sleep 10",
				},
				Requested: 0,
				Requestor: "test",
				Executor:  "test",
				Attempts:  []lib.ExecutionAttempt{},
				Status:    "",
			},
			Instructions: lib.ExecutionInstruction{
				Timeout: 1,
			},
		}

		start := time.Now()
		executor.executeJob(&job)
		assert.NotNil(t, job.CurrentRun)
		job.CurrentRun.Command.Wait()
		end := time.Now()
		assert.WithinDuration(t, start, end, 1500*time.Millisecond)
	})

	t.Run("Test retry evaluation", func(t *testing.T) {
		job := lib.Job{
			Execution: lib.Execution{
				Task: lib.Task{
					Name:    "test",
					Command: "sleep 1",
				},
				Requested: 0,
				Requestor: "test",
				Executor:  "test",
				Attempts: []lib.ExecutionAttempt{
					lib.ExecutionAttempt{
						ReturnCode: 3,
					},
				},
				Status: "",
			},
			Instructions: lib.ExecutionInstruction{
				Timeout: 1,
				Retries: 3,
				MuteOn:  []int{2},
			},
		}

		// failure then success -> warning
		assert.True(t, executor.retry(&job))
		assert.Equal(t, "error", job.Execution.Status)
		job.Execution.Attempts = append(job.Execution.Attempts, lib.ExecutionAttempt{ReturnCode: 0})
		assert.False(t, executor.retry(&job))
		assert.Equal(t, "warning", job.Execution.Status)
		// failure then muted -> warning
		job.Execution.Attempts = []lib.ExecutionAttempt{lib.ExecutionAttempt{ReturnCode: 3}, lib.ExecutionAttempt{ReturnCode: 2}}
		job.Execution.Status = "error"
		assert.False(t, executor.retry(&job))
		assert.Equal(t, "warning", job.Execution.Status)
		// success
		job.Execution.Attempts = []lib.ExecutionAttempt{lib.ExecutionAttempt{ReturnCode: 0}}
		job.Execution.Status = "success" // initialization value
		assert.False(t, executor.retry(&job))
		assert.Equal(t, "success", job.Execution.Status)
	})

	t.Run("Test event receiving", func(t *testing.T) {
		executor := (New(logger, ebus.Publish)).(*Executor)
		err = executor.Config([]byte(fmt.Sprintf(testConf, workdir)))
		require.NoError(t, err)

		task := lib.Task{
			Name:    "test",
			Command: "sleep 1",
		}
		instructions := lib.ExecutionInstruction{
			Timeout: 1,
			Retries: 3,
			MuteOn:  []int{2},
		}
		evt := data.Event{
			Type:      data.TASK,
			Publisher: "unit-test",
			Severity:  data.INFO,
			Labels: map[string]interface{}{
				"task":         task,
				"instructions": instructions,
			},
		}

		go func() {
			job := <-executor.jobs
			assert.Equal(t, task, job.Execution.Task)
			assert.Equal(t, instructions, job.Instructions)
			assert.Equal(t, 1, len(job.Execution.Attempts))
			assert.NotNil(t, job.CurrentRun.Command)
		}()

		executor.ReceiveEvent(evt)
		assert.Equal(t, map[string]struct{}{"test": struct{}{}}, executor.runList)

	})
}
