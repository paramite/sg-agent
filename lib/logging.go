package lib

import (
	"github.com/infrawatch/apputils/scheduler"
	"github.com/infrawatch/sg-core/pkg/data"
)

// CreateLogEvent formats event of type data.LOG
func CreateLogEvent(indexPrefix string, publisherSuffix string, object interface{}) *data.Event {
	var msg string
	labels := map[string]interface{}{}

	switch obj := object.(type) {
	case TaskExecution:
	//	msg = "Reaction task execution request submitted for execution."
	//	labels["action"] = "reaction"
	case scheduler.Result:
		msg = "Scheduled task execution request submitted for execution."
		if task, ok := obj.Output.(TaskRequest); ok {
			labels["action"] = "scheduled"
			labels["name"] = task.Name
			labels["command"] = task.Command
			labels["type"] = task.Type
		} else {
			return nil
		}
	}

	return &data.Event{
		Index:     FormatIndex(indexPrefix),
		Time:      GetTimestamp(),
		Type:      data.LOG,
		Publisher: FormatPublisher(publisherSuffix),
		Severity:  data.INFO,
		Labels:    labels,
		Message:   msg,
	}
}
