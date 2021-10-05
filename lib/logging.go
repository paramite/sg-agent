package lib

import (
	"fmt"

	"github.com/infrawatch/sg-core/pkg/data"
)

// CreateLogEvent formats event of type data.LOG
func CreateLogEvent(indexPrefix string, publisherSuffix string, object interface{}) *data.Event {
	var msg string
	labels := map[string]interface{}{}

	switch obj := object.(type) {
	case *Job:
		msg = "Task execution request fulfilled."
		if len(obj.Execution.Attempts) > 0 {
			result := obj.Execution.Attempts[len(obj.Execution.Attempts)-1]
			msg = fmt.Sprintf("%s %d. attempt -> RC: %d", msg, len(obj.Execution.Attempts), result.ReturnCode)
		}
		labels["name"] = obj.Execution.Task.Name
		labels["command"] = obj.Execution.Task.Command
	case Task:
		msg = "Scheduled task execution request submitted for execution."
		labels["name"] = obj.Name
		labels["command"] = obj.Command
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
