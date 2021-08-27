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
	//case lib.TaskExecution:
	//	msg = "Reaction task execution request submitted for execution."
	//  labels["action"] = "reaction"
	//  laels["task"] = TBD
	case scheduler.Result:
		msg = "Scheduled task execution request submitted for execution."
		labels["action"] = "scheduled"
		labels["task"] = obj.Output
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
