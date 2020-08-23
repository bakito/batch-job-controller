module github.com/bakito/batch-job-controller

go 1.14

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.0
	github.com/golang/mock v1.4.4
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.8.0
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/prometheus/client_golang v1.7.1
	github.com/robfig/cron/v3 v3.0.1
	golang.org/x/net v0.0.0-20200625001655-4c5254603344 // indirect
	google.golang.org/appengine v1.6.1 // indirect
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	sigs.k8s.io/controller-runtime v0.6.2
)
