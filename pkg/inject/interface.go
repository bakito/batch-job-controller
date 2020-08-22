package inject

import "k8s.io/client-go/tools/record"

// EventRecorder inject an event recorder
type EventRecorder interface {
	InjectEventRecorder(record.EventRecorder)
}
