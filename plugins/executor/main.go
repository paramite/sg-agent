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

func safeSend(jobs chan *lib.Job, job *lib.Job) (success bool) {
	defer func() {
		if recover() != nil {
			success = false
		}
	}()
	success = true
	jobs <- job
	return success
}

// ExecutorConfig holds configuration for the plugin
type ExecutorConfig struct {
	LogActions     bool   `yaml:"logActions"`
	LogIndexPrefix string `yaml:"logIndexPrefix"`
	WorkDirectory  string `yaml:"workDirectory"`
	ShellPath      string `yaml:"shellPath"`
	Workers        int
}

// Executor plugin saves events to Elasticsearch database
type Executor struct {
	conf       *ExecutorConfig
	logger     *logging.Logger
	scriptLock sync.Mutex
	scripts    map[string]string
	emit       bus.EventPublishFunc
	jobs       chan *lib.Job
}

// New constructor
func New(logger *logging.Logger, sendEvent bus.EventPublishFunc) application.Application {
	return &Executor{
		logger:  logger,
		jobs:    make(chan *lib.Job),
		emit:    sendEvent,
		scripts: make(map[string]string),
	}
}

// getScript returns script path for appropriate command
func (te *Executor) getScript(task *lib.Task) (string, error) {
	script := ""
	te.scriptLock.Lock()
	if _, ok := te.scripts[task.Command]; !ok {
		scriptFile, err := ioutil.TempFile(te.conf.WorkDirectory, "script")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary file for script: %s", err)
		}
		_, err = scriptFile.Write([]byte(fmt.Sprintf("#!%s\n%s\n", te.conf.ShellPath, task.Command)))
		if err != nil {
			return "", fmt.Errorf("failed to write script content to temporary file: %s", err)
		}
		te.scripts[task.Command] = scriptFile.Name()
		scriptFile.Close()
		te.logger.Metadata(logging.Metadata{"plugin": appname, "command": task.Name, "path": scriptFile.Name()})
		te.logger.Debug("created task script.")
	}
	script = te.scripts[task.Command]
	te.scriptLock.Unlock()
	return script, nil
}

// ReceiveEvent listens for task results and reacts on them if necessary according
// to configured scenario, eg. reactor part
func (te *Executor) ReceiveEvent(event data.Event) {
	switch event.Type {
	case data.RESULT:
		// NOTE: Do not react on own emits
	case data.LOG:
		// NOTE: Do not react on own emits
	case data.TASK:
		if req, ok := event.Labels["task"]; ok {
			if task, ok := req.(lib.Task); ok {
				// prepare job
				var instructions lib.ExecutionInstruction
				if instr, ok := event.Labels["instructions"]; !ok {
					te.logger.Metadata(logging.Metadata{"plugin": appname, "event": event})
					te.logger.Warn("missing execution instructions in task event")
					return
				} else if instructions, ok = instr.(lib.ExecutionInstruction); !ok {
					te.logger.Metadata(logging.Metadata{"plugin": appname, "event": event})
					te.logger.Warn("invalid type of execution instructions")
					return
				}

				job := lib.Job{
					Execution: lib.Execution{
						Task:      task,
						Requestor: event.Publisher,
						Executor:  lib.FormatPublisher(appname),
						Requested: event.Time,
						Attempts:  make([]lib.ExecutionAttempt, 0, instructions.Retries),
						Status:    lib.SUCCESS.String(),
					},
					Instructions: instructions,
				}

				if !safeSend(te.jobs, &job) {
					te.logger.Metadata(logging.Metadata{"plugin": appname, "task": job.Execution.Task})
					te.logger.Warn("did not manage to execute scheduled task")
				}
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
func (te *Executor) Run(ctx context.Context, done chan bool) {
	te.logger.Metadata(logging.Metadata{"plugin": appname})
	te.logger.Info("executor started")

	// spawn index workers
	wg := sync.WaitGroup{}
	for i := 0; i < te.conf.Workers; i++ {
		te.logger.Metadata(logging.Metadata{"plugin": appname, "worker-id": i})
		te.logger.Debug("spawning worker")
		wg.Add(1)

		go func(te *Executor, wg *sync.WaitGroup, i int) {
			defer wg.Done()
			for job := range te.jobs {
				status := lib.SUCCESS
			attempts:
				for i := 0; i < job.Instructions.Retries; i++ {
					var ctx context.Context
					var cancel context.CancelFunc
					if job.Instructions.Timeout > 0 {
						ctx, cancel = context.WithTimeout(context.Background(), time.Duration(job.Instructions.Timeout)*time.Second)
					} else {
						ctx, cancel = context.WithCancel(context.Background())
					}
					defer cancel()
					script, err := te.getScript(&job.Execution.Task)
					if err != nil {
						te.logger.Metadata(logging.Metadata{"plugin": appname, "task": job.Execution.Task})
						te.logger.Warn("failed to get script file")
						break
					}
					command := exec.CommandContext(ctx, te.conf.ShellPath, script)
					stdout := bytes.Buffer{}
					stderr := bytes.Buffer{}
					command.Stdout = &stdout
					command.Stderr = &stderr
					command.Run()

					// record the attempt
					rc := command.ProcessState.ExitCode()
					job.Execution.Attempts = append(job.Execution.Attempts, lib.ExecutionAttempt{
						Executed:   lib.GetTimestamp(),
						Duration:   (command.ProcessState.SystemTime() + command.ProcessState.UserTime()).Seconds(),
						ReturnCode: rc,
						StdOut:     stdout.String(),
						StdErr:     stderr.String(),
					})

					if te.conf.LogActions {
						record := lib.CreateLogEvent(te.conf.LogIndexPrefix, appname, job)
						if record != nil {
							te.emit(*record)
						} else {
							te.logger.Metadata(logging.Metadata{"plugin": appname, "job": job})
							te.logger.Warn("failed format log record from job")
						}
					}

					// evaluate overall task status
					if rc == 0 {
						if status == lib.SUCCESS {
							status = lib.SUCCESS
						} else {
							// previous attempt failed
							status = lib.WARNING
						}
						// no need to continue with attempts
						break attempts
					} else {
						for _, mute := range job.Instructions.MuteOn {
							if rc == mute {
								status = lib.WARNING
								break attempts
							}
						}
					}

					status = lib.ERROR
					if job.Instructions.CoolDown > 0 {
						time.Sleep(time.Duration(job.Instructions.CoolDown) * time.Second)
					}
				}

				job.Execution.Status = status.String()
				te.emit(data.Event{
					Time:      lib.GetTimestamp(),
					Type:      data.RESULT,
					Publisher: lib.FormatPublisher(appname),
					Severity:  status.ToSeverity(),
					Labels:    map[string]interface{}{"result": job.Execution},
				})
			}
		}(te, &wg, i)
	}

	<-ctx.Done()
	close(te.jobs)
	wg.Wait()
	te.logger.Metadata(logging.Metadata{"plugin": appname})
	te.logger.Info("exited")
}

// Config implements application.Application
func (te *Executor) Config(c []byte) error {
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
