package http

import (
	"github.com/bakito/batch-job-controller/pkg/inject"
)

var _ inject.Healthz = &Server{}
