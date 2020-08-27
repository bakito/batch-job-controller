module github.com/bakito/batch-job-controller

go 1.14

require (
	github.com/ghodss/yaml v1.0.0
	// fix untli 0.2.1 is released https://github.com/go-logr/logr/issues/22
	github.com/go-logr/logr v0.2.1-0.20200730175230-ee2de8da5be6
	github.com/go-logr/zapr v0.2.0
	github.com/go-playground/validator/v10 v10.3.0
	github.com/golang/mock v1.4.4
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.8.0
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.13.0
	github.com/robfig/cron/v3 v3.0.1
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	sigs.k8s.io/controller-runtime v0.6.2
)
