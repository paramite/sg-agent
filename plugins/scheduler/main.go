package main

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/go-playground/validator.v9"

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

func requestExec(ts *TaskScheduler, item interface{}) {
	taskName := ""
	instr := lib.ExecutionInstruction{}
	switch request := item.(type) {
	case lib.ScheduleItem:
		taskName = request.Task
		instr = request.Instructions
	case lib.Reaction:
		taskName = request.Reaction
		instr = request.Instructions
	}

	if instr.Retries < 1 {
		instr.Retries = 1
	}
	event := data.Event{
		Time:      lib.GetTimestamp(),
		Type:      data.TASK,
		Publisher: lib.FormatPublisher(appname),
		Severity:  data.INFO,
		Labels: map[string]interface{}{
			"task":         ts.tasks[taskName],
			"instructions": instr,
		},
	}
	ts.emit(event)

	ts.logger.Metadata(logging.Metadata{"plugin": appname, "task": taskName})
	ts.logger.Debug("task execution request emitted")
}

// TaskScheduler plugin creates task execution requests according to configured schedule
type TaskScheduler struct {
	conf       *SchedulerConfig
	logger     *logging.Logger
	tasks      map[string]lib.Task
	schedule   *scheduler.Scheduler
	taskReacts map[string][]lib.Reaction
	metrReacts map[string][]lib.Reaction
	emit       bus.EventPublishFunc
}

// New constructor
func New(logger *logging.Logger, sendEvent bus.EventPublishFunc) application.Application {
	sched, err := scheduler.New(logger)
	if err != nil {
		logger.Metadata(logging.Metadata{"plugin": appname, "error": err})
		logger.Warn("error during initialization")
	}
	return &TaskScheduler{
		logger:     logger,
		tasks:      make(map[string]lib.Task),
		schedule:   sched,
		taskReacts: make(map[string][]lib.Reaction),
		metrReacts: make(map[string][]lib.Reaction),
		emit:       sendEvent,
	}
}

// TODO(mmagr): Once we will have possibility to let plugins communicate in sg-core via external meessage bus
//              (eg. via transports), we should move ReceiveEvent and ReceiveMetric to separate plugin called reactor,
//              so that user is able to run scheduling part and reacting part separately

// ReceiveEvent listens for task results and reacts on them if necessary according
// to configured scenario, eg. reactor part
func (ts *TaskScheduler) ReceiveEvent(event data.Event) {
	switch event.Type {
	case data.LOG:
		// NOTE: Do not react on own emits
	case data.TASK:
		// NOTE: Do not react on own emits
	case data.RESULT:
		if res, ok := event.Labels["result"]; ok {
			if result, ok := res.(lib.Execution); ok {
				if rList, ok := ts.taskReacts[result.Task.Name]; ok {
					for _, reaction := range rList {
						if reaction.RequiredOnResult(result) {
							requestExec(ts, reaction)
						}
					}
				} else {
					ts.logger.Metadata(logging.Metadata{"plugin": appname, "task": result.Task.Name})
					ts.logger.Debug("no reaction found for received task result")
				}
			} else {
				ts.logger.Metadata(logging.Metadata{"plugin": appname, "type": fmt.Sprintf("%T", res)})
				ts.logger.Debug("unknow type of result data")
			}
		} else {
			ts.logger.Metadata(logging.Metadata{"plugin": appname, "event": event})
			ts.logger.Debug("missing result in event data")
		}
	default:
		ts.logger.Metadata(logging.Metadata{"plugin": appname, "event": event})
		ts.logger.Debug("received unknown event")
		return
	}
}

// ReceiveMetric listens on
func (ts *TaskScheduler) ReceiveMetric(name string, t float64, typ data.MetricType, interval time.Duration, value float64, labelKeys []string, labelVals []string) {
	if rList, ok := ts.metrReacts[name]; ok {
		metric := data.Metric{
			Name:      name,
			Time:      t,
			Type:      typ,
			Interval:  interval,
			Value:     value,
			LabelKeys: labelKeys,
			LabelVals: labelVals,
		}
		for _, reaction := range rList {
			if reaction.RequiredOnMetric(&metric) {
				ts.logger.Metadata(logging.Metadata{"plugin": appname, "metric": metric, "reaction": reaction})
				ts.logger.Debug("received metric passed reaction condition")
				requestExec(ts, reaction)
			}
		}
	}
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

func conditionValidator(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	parts := strings.Split(value, "=")
	if len(parts) != 2 {
		return false
	}
	if parts[0] == "status" {
		for _, cond := range (lib.ExecutionStatus(0)).List() {
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
		if _, err := lib.IntervalToDuration(parts[1]); err == nil {
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

// Config implements application.Application
func (ts *TaskScheduler) Config(c []byte) error {
	config.Validate.RegisterValidation("condition", conditionValidator)

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

	// register schedule items
	for _, item := range ts.conf.Schedule {
		data := item
		if _, ok := ts.tasks[data.Task]; !ok {
			return fmt.Errorf("scheduled task %s was not found in task list", data.Task)
		}

		err := ts.schedule.RegisterTask(data.Task, data.Interval, 0,
			func(ctx context.Context, log *logging.Logger) (interface{}, error) {
				requestExec(ts, data)
				ts.logger.Metadata(logging.Metadata{"plugin": appname, "task": data.Task})
				ts.logger.Debug("task execution requested")
				return data, nil
			})
		if err != nil {
			ts.logger.Metadata(logging.Metadata{"plugin": appname, "task": data.Task})
			ts.logger.Debug("failed to register task execution")
		}
	}

	// register reaction items
	for _, item := range ts.conf.Reactions {
		data := item

		if _, ok := ts.tasks[data.Reaction]; !ok {
			return fmt.Errorf("task %s was not found in task list", data.Reaction)
		}

		if data.OfTask != "" {
			// register task based reaction
			if _, ok := ts.tasks[data.OfTask]; !ok {
				return fmt.Errorf("task %s was not found in task list", data.OfTask)
			}
			if taskList, ok := ts.taskReacts[data.OfTask]; !ok {
				ts.taskReacts[data.OfTask] = []lib.Reaction{data}
			} else {
				ts.taskReacts[data.OfTask] = append(taskList, data)
			}
		} else {
			// register metric based ReactionOnValue
			if taskList, ok := ts.metrReacts[data.OfMetric]; !ok {
				ts.metrReacts[data.OfMetric] = []lib.Reaction{data}
			} else {
				ts.metrReacts[data.OfMetric] = append(taskList, data)
			}
		}

	}

	return nil
}
