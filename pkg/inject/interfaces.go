package inject

import (
	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// injects from "sigs.k8s.io/controller-runtime/pkg/runtime/inject" are set by the manager

// EventRecorder inject the event recorder
type EventRecorder interface {
	InjectEventRecorder(record.EventRecorder)
}

// Cache inject the cache
type Cache interface {
	InjectCache(lifecycle.Cache)
}

// Reader inject the api reader
type Reader interface {
	InjectReader(client.Reader)
}

// Cache inject an event recorder
type Config interface {
	InjectConfig(*config.Config)
}
