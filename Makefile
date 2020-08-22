

# generate mocks
mocks:
	mockgen -destination pkg/mocks/logr/mock.go   github.com/go-logr/logr                              Logger
	mockgen -destination pkg/mocks/client/mock.go sigs.k8s.io/controller-runtime/pkg/client            Client,Reader
	mockgen -destination pkg/mocks/cache/mock.go  github.com/bakito/batch-job-controller/pkg/lifecycle Cache
	mockgen -destination pkg/mocks/record/mock.go  k8s.io/client-go/tools/record                        EventRecorder

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Run go mod tidy
tidy:
	go mod tidy

# Run tests
test: mocks tidy fmt vet
	go test ./...  -coverprofile=coverage.out
	go tool cover -func=coverage.out
	goverage-badge generate

# Run ci tests
test-ci: test
	goveralls -service=travis-ci -v -coverprofile=coverage.out

# Run tests
helm-template:
	helm template helm/example-batch-job-controller/ --debug --set routes.hostSuffix=test.com

# Build docker image
build-docker:
	docker build --build-arg upx_brute=" " -t batch-job-controller .

# Build podman image
build-podman:
	podman build --build-arg upx_brute=" " -t batch-job-controller .

tools: mockgen ginkgo goverage-badge helm goveralls

mockgen:
ifeq (, $(shell which mockgen))
 $(shell go get github.com/golang/mock/mockgen@v1.4.3)
endif
ginkgo:
ifeq (, $(shell which ginkgo))
 $(shell go get github.com/onsi/ginkgo/ginkgo)
endif
goverage-badge:
ifeq (, $(shell which goverage-badge))
 $(shell go get github.com/bakito/goverage-badge)
endif
goveralls:
ifeq (, $(shell which goveralls))
 $(shell go get github.com/mattn/goveralls)
endif
helm:
ifeq (, $(shell which helm))
 $(shell curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash)
endif