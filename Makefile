# find or install mockgen
ifeq (, $(shell which mockgen))
 $(shell go get github.com/golang/mock/mockgen@v1.4.3)
endif
ifeq (, $(shell which ginkgo))
 $(shell go get github.com/onsi/ginkgo/ginkgo)
endif
ifeq (, $(shell which gopherbadger))
 $(shell go get github.com/jpoles1/gopherbadger)
endif
ifeq (, $(shell which helm))
 $(shell curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash)
endif

# generate mocks
mocks:
	mockgen -destination pkg/mocks/logr/mock.go github.com/go-logr/logr Logger
	mockgen -destination pkg/mocks/client/mock.go sigs.k8s.io/controller-runtime/pkg/client Client,Reader

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Run tests
test: mocks fmt vet
	gopherbadger -md="README.md" -png=false

# Run tests
helm-template:
	helm template helm/example-batch-job-controller/ --debug

# Build docker image
build-docker:
	docker build -t batch-job-controller .

# Build podman image
build-podman:
	podman build -t batch-job-controller .
