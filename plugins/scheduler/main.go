package main

import (
	"bytes"
	"context"
	"fmt"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/apputils/scheduler"
	"github.com/infrawatch/sg-agent/lib"

	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
)

const (
	appname = "scheduler"
)

// SchedulerConfig holds configuration for the plugin
type SchedulerConfig struct {
	LogActions     bool               `yaml:"logActions"`
	LogIndexPrefix string             `yaml:"logIndexPrefix"`
	Tasks          []lib.Task         `validate:"dive"`
	Schedule       []lib.ScheduleItem `validate:"dive"`
	Reactions      []lib.Reaction     `validate:"dive"`
}

func requestExec(ts *TaskScheduler, item *lib.ScheduleItem) {
	if item.Instructions.Retries < 1 {
		item.Instructions.Retries = 1
	}
	event := data.Event{
		Time:      lib.GetTimestamp(),
		Type:      data.TASK,
		Publisher: lib.FormatPublisher(appname),
		Severity:  data.INFO,
		Labels: map[string]interface{}{
			"task":         ts.tasks[item.Task],
			"instructions": item.Instructions,
		},
	}
	ts.emit(event)

	ts.logger.Metadata(logging.Metadata{"plugin": appname, "task": item.Task})
	ts.logger.Debug("task execution request emitted")
}

// TaskScheduler plugin saves events to Elasticsearch database
type TaskScheduler struct {
	conf      *SchedulerConfig
	logger    *logging.Logger
	tasks     map[string]lib.Task
	schedule  *scheduler.Scheduler
	reactions map[string]lib.Reaction
	emit      bus.EventPublishFunc
}

// New constructor
func New(logger *logging.Logger, sendEvent bus.EventPublishFunc) application.Application {
	sched, err := scheduler.New(logger)
	if err != nil {
		logger.Metadata(logging.Metadata{"plugin": appname, "error": err})
		logger.Warn("error during initialization")
	}
	return &TaskScheduler{
		logger:   logger,
		tasks:    make(map[string]lib.Task),
		schedule: sched,
		emit:     sendEvent,
	}
}

// ReceiveEvent listens for task results and reacts on them if necessary according
// to configured scenario, eg. reactor part
func (ts *TaskScheduler) ReceiveEvent(event data.Event) {
	switch event.Type {
	case data.RESULT:
		/*
			{                                                                                                                                                                                                                              [9/1792]
			  "Index": "",
			  "Type": "result",
			  "Publisher": "lenovo-p720-rdo-13.tpb.lab.eng.brq.redhat.com-executor",
			  "Severity": 2,
			  "Message": "",
			  "Labels": {
			    "result": {
			      "Request": {
			        "Name": "test1",
			        "Command": "echo 'test1'",
			        "Interval": "1s",
			        "Timeout": 0,
			        "MuteOn": null,
			        "Retries": 1,
			        "CoolDown": 0,
			        "Type": "internal"
			      },
			      "Requestor": "lenovo-p720-rdo-13.tpb.lab.eng.brq.redhat.com-scheduler",
			      "Requested": 1632345705,
			      "Executor": "lenovo-p720-rdo-13.tpb.lab.eng.brq.redhat.com-executor",
			      "Attempts": [
			        {
			          "Executed": 1632345705,
			          "Duration": 0.001396,
			          "ReturnCode": 0,
			          "StdOut": "test1\n",
			          "StdErr": ""
			        }
			      ],
			      "Status": 0
			    }
			  },
			  "Annotations": null
			}

		*/
	case data.LOG:
		// NOTE: Do not react on own emits
	case data.TASK:
		// NOTE: ditto
	default:
		ts.logger.Metadata(logging.Metadata{"plugin": appname, "event": event})
		ts.logger.Debug("received unknown event")
		return
	}

	// TODO: 1. browse reactions for specific task
	//       2. if reactions for the task are configured, search for specific scenario (success/failure/warning)
}

// Run creates task requests according to schedule, eg. scheduler part
func (ts *TaskScheduler) Run(ctx context.Context, done chan bool) {
	ts.logger.Metadata(logging.Metadata{"plugin": appname, "schedule": ts.schedule.GetSchedule()})
	ts.logger.Debug("scheduler starting")

	scheduleQueue := ts.schedule.Start(1, false)
	ts.logger.Metadata(logging.Metadata{"plugin": appname})
	ts.logger.Info("task schedule started")

	for {
		select {
		case <-ctx.Done():
			goto done
		case req, _ := <-scheduleQueue:
			if ts.conf.LogActions {
				record := lib.CreateLogEvent(ts.conf.LogIndexPrefix, appname, ts.tasks[req.Task])
				if record != nil {
					ts.emit(*record)
				} else {
					ts.logger.Metadata(logging.Metadata{"plugin": appname, "task": ts.tasks[req.Task]})
					ts.logger.Warn("failed format log record from task")
				}
			}
			ts.logger.Metadata(logging.Metadata{"plugin": appname, "request": req})
			ts.logger.Debug("task execution request sent")
		}
	}

done:
	ts.schedule.Stop(true)
	ts.logger.Metadata(logging.Metadata{"plugin": appname})
	ts.logger.Info("exited")
}

// Config implements application.Application
func (ts *TaskScheduler) Config(c []byte) error {
	ts.conf = &SchedulerConfig{
		LogActions:     true,
		LogIndexPrefix: "agentlogs",
	}
	err := config.ParseConfig(bytes.NewReader(c), ts.conf)
	if err != nil {
		return err
	}

	// gather tasks
	for _, task := range ts.conf.Tasks {
		ts.tasks[task.Name] = task
	}

	// register each task to a schedule
	for _, item := range ts.conf.Schedule {
		data := item
		if _, ok := ts.tasks[data.Task]; !ok {
			return fmt.Errorf("scheduled task %s was not found in task list", data.Task)
		}

		err := ts.schedule.RegisterTask(data.Task, data.Interval, 0,
			func(ctx context.Context, log *logging.Logger) (interface{}, error) {
				requestExec(ts, &data)
				ts.logger.Metadata(logging.Metadata{"plugin": appname, "task": data.Task})
				ts.logger.Debug("task execution requested")
				return data, nil
			})
		if err != nil {
			ts.logger.Metadata(logging.Metadata{"plugin": appname, "task": data.Task})
			ts.logger.Debug("failed to register task execution")
		}
	}

	return nil
}
