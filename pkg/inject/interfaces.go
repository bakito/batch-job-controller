package inject

import (
	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// injects from "sigs.k8s.io/controller-runtime/pkg/runtime/inject" are set by the manager

// EventRecorder inject the event recorder
type EventRecorder interface {
	InjectEventRecorder(record.EventRecorder)
}

// Controller inject the cache
type Controller interface {
	InjectController(lifecycle.Controller)
}

// Reader inject the api reader
type Reader interface {
	InjectReader(client.Reader)
}

// Config inject the config
type Config interface {
	InjectConfig(*config.Config)
}

type Healthz interface {
	Name() string
	ReadyzCheck() healthz.Checker
	HealthzCheck() healthz.Checker
}
