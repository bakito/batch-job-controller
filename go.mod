module github.com/bakito/batch-job-controller

go 1.14

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0
	github.com/go-playground/validator/v10 v10.4.1
	github.com/golang/mock v1.4.4
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	github.com/prometheus/client_golang v1.8.0
	github.com/prometheus/common v0.14.0
	github.com/robfig/cron/v3 v3.0.1
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.6.3
)
