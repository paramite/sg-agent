package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-agent/lib"

	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
)

const (
	appname  = "executor"
	waitTime = 100 // miliseconds sleep time in loops
)

// ExecutorConfig holds configuration for the plugin
type ExecutorConfig struct {
	LogActions     bool   `yaml:"logActions"`
	LogIndexPrefix string `yaml:"logIndexPrefix"`
	WorkDirectory  string `yaml:"workDirectory"`
	ShellPath      string `yaml:"shellPath"`
	Workers        int
}

// TaskExecutor plugin saves events to Elasticsearch database
type TaskExecutor struct {
	conf       *ExecutorConfig
	logger     *logging.Logger
	scriptLock sync.Mutex
	scripts    map[string]string
	emit       bus.EventPublishFunc
	tasks      chan *lib.Task
}

// New constructor
func New(logger *logging.Logger, sendEvent bus.EventPublishFunc) application.Application {
	return &TaskExecutor{
		logger:  logger,
		tasks:   make(chan *lib.Task),
		emit:    sendEvent,
		scripts: make(map[string]string),
	}
}

// getScript returns script path for appropriate command
func (te *TaskExecutor) getScript(request *lib.TaskRequest) (string, error) {
	script := ""
	te.scriptLock.Lock()
	if _, ok := te.scripts[request.Command]; !ok {
		scriptFile, err := ioutil.TempFile(te.conf.WorkDirectory, "script")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary file for script: %s", err)
		}
		_, err = scriptFile.Write([]byte(fmt.Sprintf("#!%s\n%s\n", te.conf.ShellPath, request.Command)))
		if err != nil {
			return "", fmt.Errorf("failed to write script content to temporary file: %s", err)
		}
		te.scripts[request.Command] = scriptFile.Name()
		scriptFile.Close()
		te.logger.Metadata(logging.Metadata{"plugin": appname, "command": request.Name, "path": scriptFile.Name()})
		te.logger.Debug("created task script.")
	}
	script = te.scripts[request.Command]
	te.scriptLock.Unlock()
	return script, nil
}

// ReceiveEvent listens for task results and reacts on them if necessary according
// to configured scenario, eg. reactor part
func (te *TaskExecutor) ReceiveEvent(event data.Event) {
	switch event.Type {
	case data.RESULT:
		// NOTE: Do not react on own emits
	case data.LOG:
		// NOTE: Do not react on own emits
	case data.TASK:
		if req, ok := event.Labels["task"]; ok {
			if treq, ok := req.(*lib.TaskRequest); ok {
				// prepare task
				var ctx context.Context
				var cancel context.CancelFunc
				if treq.Timeout > 0 {
					ctx, cancel = context.WithTimeout(context.Background(), time.Duration(treq.Timeout)*time.Second)
				} else {
					ctx, cancel = context.WithCancel(context.Background())
				}
				script, err := te.getScript(treq)
				if err != nil {
					te.logger.Metadata(logging.Metadata{"plugin": appname, "request": treq})
					te.logger.Warn("failed to get script file")
					cancel() // just in sake of making vet happy
					return
				}
				task := lib.Task{
					Execution: lib.TaskExecution{
						Request:   *treq,
						Requestor: event.Publisher,
						Executor:  lib.FormatPublisher(appname),
						Requested: event.Time,
						Attempts:  make([]lib.ExecutionAttempt, 0, treq.Retries),
						Status:    lib.SUCCESS,
					},
					Command: exec.CommandContext(ctx, te.conf.ShellPath, script),
					Cancel:  cancel,
				}
				te.tasks <- &task
			} else {
				te.logger.Metadata(logging.Metadata{"plugin": appname, "type": fmt.Sprintf("%T", req)})
				te.logger.Debug("unknow type of task data")
			}
		} else {
			te.logger.Metadata(logging.Metadata{"plugin": appname, "event": event})
			te.logger.Debug("missing task in event data")
		}
	default:
		te.logger.Metadata(logging.Metadata{"plugin": appname, "event": event})
		te.logger.Debug("received unknown event")
		return
	}
}

// Run creates task requests according to schedule, eg. scheduler part
func (te *TaskExecutor) Run(ctx context.Context, done chan bool) {
	te.logger.Metadata(logging.Metadata{"plugin": appname})
	te.logger.Info("executor started")

	// spawn index workers
	wg := sync.WaitGroup{}
	for i := 0; i < te.conf.Workers; i++ {
		te.logger.Metadata(logging.Metadata{"plugin": appname, "worker-id": i})
		te.logger.Debug("spawning worker")
		wg.Add(1)

		go func(te *TaskExecutor, wg *sync.WaitGroup, i int) {
			defer wg.Done()
			for task := range te.tasks {
			attempts:
				for i := 0; i < task.Execution.Request.Retries; i++ {
					stdout := bytes.Buffer{}
					stderr := bytes.Buffer{}
					task.Command.Stdout = &stdout
					task.Command.Stderr = &stderr
					if err := task.Command.Run(); err != nil {
						te.logger.Metadata(logging.Metadata{"plugin": appname, "worker-id": i, "error": err})
						te.logger.Warn("failed to run task")
						continue
					}
					// record the attempt
					rc := task.Command.ProcessState.ExitCode()
					task.Execution.Attempts = append(task.Execution.Attempts, lib.ExecutionAttempt{
						Executed:   lib.GetTimestamp(),
						Duration:   (task.Command.ProcessState.SystemTime() + task.Command.ProcessState.UserTime()).Seconds(),
						ReturnCode: rc,
						StdOut:     stdout.String(),
						StdErr:     stderr.String(),
					})

					// evaluate overall task status
					if rc == 0 {
						if task.Execution.Status == lib.SUCCESS {
							task.Execution.Status = lib.SUCCESS
						} else {
							// previous attempt failed
							task.Execution.Status = lib.WARNING
						}
						// no need to continue with attempts
						break attempts
					} else {
						for _, mute := range task.Execution.Request.MuteOn {
							if rc == mute {
								task.Execution.Status = lib.WARNING
								break
							}
						}
					}
					task.Execution.Status = lib.ERROR
				}

				te.emit(data.Event{
					Time:      lib.GetTimestamp(),
					Type:      data.RESULT,
					Publisher: lib.FormatPublisher(appname),
					Severity:  data.INFO,
					Labels:    map[string]interface{}{"result": task.Execution},
				})
			}
		}(te, &wg, i)
	}

	<-ctx.Done()
	close(te.tasks)
	wg.Wait()
	te.logger.Metadata(logging.Metadata{"plugin": appname})
	te.logger.Info("exited")
}

// Config implements application.Application
func (te *TaskExecutor) Config(c []byte) error {
	te.conf = &ExecutorConfig{
		LogActions:     true,
		LogIndexPrefix: "agentlogs",
		WorkDirectory:  "/var/lib/sg-agent",
		ShellPath:      "/bin/bash",
		Workers:        3,
	}
	err := config.ParseConfig(bytes.NewReader(c), te.conf)
	if err != nil {
		return err
	}

	if _, err := os.Stat(te.conf.WorkDirectory); os.IsNotExist(err) {
		err := os.MkdirAll(te.conf.WorkDirectory, 0700)
		if err != nil {
			return err
		}
	}
	return nil
}
