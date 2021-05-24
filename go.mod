module github.com/bakito/batch-job-controller

go 1.16

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/go-playground/validator/v10 v10.6.1
	github.com/golang/mock v1.5.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/common v0.25.0
	github.com/robfig/cron/v3 v3.0.1
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	sigs.k8s.io/controller-runtime v0.8.3
)
