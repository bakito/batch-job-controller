

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
test: ginkgo tidy mocks fmt vet
	ginkgo ./... -coverprofile=coverage.out

# Run tests
helm-lint: helm
	helm lint helm/example-batch-job-controller/ --set routes.hostSuffix=test.com

helm-template: helm-lint
	helm template helm/example-batch-job-controller/ --debug --set routes.hostSuffix=test.com

# Build docker image
build-docker:
	docker build --build-arg upx_brute=" " -t batch-job-controller .

# Build podman image
build-podman:
	podman build --build-arg upx_brute=" " -t batch-job-controller .

release: semver
	@version=$$(semver); \
	git tag -s $$version -m"Release $$version"
	goreleaser --rm-dist

test-release:
	goreleaser --skip-publish --snapshot --rm-dist

tools: mockgen ginkgo helm

mockgen:
ifeq (, $(shell which mockgen))
 $(shell go install github.com/golang/mock/mockgen@v1.6.0)
endif
ginkgo:
ifeq (, $(shell which ginkgo))
 $(shell go install github.com/onsi/ginkgo/ginkgo@v1.16.5)
endif
helm:
ifeq (, $(shell which helm))
 $(shell curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash)
endif

semver:
ifeq (, $(shell which semver))
 $(shell go install github.com/bakito/semver@latest)
endif
