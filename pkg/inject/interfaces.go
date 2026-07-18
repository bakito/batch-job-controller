package inject

import (
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
)

// injects from "sigs.k8s.io/controller-runtime/pkg/runtime/inject" are set by the manager

// EventRecorder inject the event recorder.
type EventRecorder interface {
	InjectEventRecorder(e events.EventRecorder)
}

// Controller inject the cache.
type Controller interface {
	InjectController(c lifecycle.Controller)
}

// Reader inject the api reader.
type Reader interface {
	InjectReader(c client.Reader)
}

// Client inject the api client.
type Client interface {
	InjectClient(c client.Client)
}

// Config inject the config.
type Config interface {
	InjectConfig(c *config.Config)
}

type Healthz interface {
	Name() string
	ReadyzCheck() healthz.Checker
	HealthzCheck() healthz.Checker
}
