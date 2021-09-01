package main

import (
	"bytes"
	"context"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/apputils/scheduler"
	"github.com/infrawatch/sg-agent/lib"

	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
)

const (
	appname = "agent-scheduler"
)

// SchedulerConfig holds configuration for the plugin
type SchedulerConfig struct {
	LogActions     bool              `yaml:"logActions"`
	LogIndexPrefix string            `yaml:"logIndexPrefix"`
	Schedule       []lib.TaskRequest `validate:"dive"`
	Reactions      []lib.TaskRequest `validate:"dive"`
}

func requestExec(ts *TaskScheduler, task *lib.TaskRequest) {
	event := data.Event{
		Time:      lib.GetTimestamp(),
		Type:      data.TASK,
		Publisher: lib.FormatPublisher(appname),
		Severity:  data.INFO,
		Labels:    map[string]interface{}{"task": task},
	}
	ts.emit(event)

	ts.logger.Metadata(logging.Metadata{"plugin": appname, "task": task})
	ts.logger.Debug("task execution request emitted")
}

// TaskScheduler plugin saves events to Elasticsearch database
type TaskScheduler struct {
	conf     *SchedulerConfig
	logger   *logging.Logger
	schedule *scheduler.Scheduler
	emit     bus.EventPublishFunc
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
		schedule: sched,
		emit:     sendEvent,
	}
}

// ReceiveEvent listens for task results and reacts on them if necessary according
// to configured scenario, eg. reactor part
func (ts *TaskScheduler) ReceiveEvent(event data.Event) {
	switch event.Type {
	case data.RESULT:
		// TODO: React on task result
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
				record := lib.CreateLogEvent(ts.conf.LogIndexPrefix, appname, req)
				if record != nil {
					ts.emit(*record)
				} else {
					ts.logger.Metadata(logging.Metadata{"plugin": appname, "request": req})
					ts.logger.Warn("failed format log record from execution request")
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

	// register each task to a schedule
	for _, task := range ts.conf.Schedule {
		data := task
		err := ts.schedule.RegisterTask(data.Name, data.Interval, 0,
			func(ctx context.Context, log *logging.Logger) (interface{}, error) {
				requestExec(ts, &data)
				ts.logger.Metadata(logging.Metadata{"plugin": appname, "task": data})
				ts.logger.Debug("task execution requested")
				return data, nil
			})
		if err != nil {
			ts.logger.Metadata(logging.Metadata{"plugin": appname, "task": data})
			ts.logger.Debug("failed to register task execution")
		}
	}

	return nil
}
