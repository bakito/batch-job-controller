

# generate mocks
mocks:
	mockgen -destination pkg/mocks/lifecycle/mock.go   github.com/bakito/batch-job-controller/pkg/lifecycle Controller
	mockgen -destination pkg/mocks/logr/mock.go        github.com/go-logr/logr                              Logger
	mockgen -destination pkg/mocks/record/mock.go      k8s.io/client-go/tools/record                        EventRecorder
	mockgen -destination pkg/mocks/client/mock.go      sigs.k8s.io/controller-runtime/pkg/client            Client,Reader
	mockgen -destination pkg/mocks/manager/mock.go     sigs.k8s.io/controller-runtime/pkg/manager           Manager

# Run go fmt against code
fmt:
	go fmt ./...
	gofmt -s -w .

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

# Run ci tests
test-ci: test
	goveralls -service=travis-ci -v -coverprofile=coverage.out

# Run tests
helm-template: helm
	helm template helm/example-batch-job-controller/ --debug --set routes.hostSuffix=test.com

# Build docker image
build-docker:
	docker build --build-arg upx_brute=" " -t batch-job-controller .

# Build podman image
build-podman:
	podman build --build-arg upx_brute=" " -t batch-job-controller .

release: goreleaser
	goreleaser --rm-dist

test-release: goreleaser
	goreleaser --skip-publish --snapshot --rm-dist

licenses: go-licenses
	go-licenses csv "github.com/bakito/batch-job-controller/cmd/generic"  2>/dev/null | sort > ./dependency-licenses.csv

tools: mockgen ginkgo helm goveralls goreleaser go-licenses

mockgen:
ifeq (, $(shell which mockgen))
 $(shell go get github.com/golang/mock/mockgen@v1.4.3)
endif
ginkgo:
ifeq (, $(shell which ginkgo))
 $(shell go get github.com/onsi/ginkgo/ginkgo)
endif
goveralls:
ifeq (, $(shell which goveralls))
 $(shell go get github.com/mattn/goveralls)
endif
go-licenses:
ifeq (, $(shell which go-licenses))
 $(shell go get github.com/google/go-licenses)
endif
goreleaser:
ifeq (, $(shell which goreleaser))
 $(shell go get github.com/goreleaser/goreleaser)
endif
helm:
ifeq (, $(shell which helm))
 $(shell curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash)
endif