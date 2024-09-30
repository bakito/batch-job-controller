# Include toolbox tasks
include ./.toolbox.mk



# generate mocks
mocks: mockgen
	$(MOCKGEN) -destination pkg/mocks/lifecycle/mock.go github.com/bakito/batch-job-controller/pkg/lifecycle Controller
	$(MOCKGEN) -destination pkg/mocks/logr/mock.go      github.com/go-logr/logr                              LogSink
	$(MOCKGEN) -destination pkg/mocks/record/mock.go    k8s.io/client-go/tools/record                        EventRecorder
	$(MOCKGEN) -destination pkg/mocks/client/mock.go    sigs.k8s.io/controller-runtime/pkg/client            Client,Reader
	$(MOCKGEN) -destination pkg/mocks/manager/mock.go   sigs.k8s.io/controller-runtime/pkg/manager           Manager

# Run go mod tidy
tidy:
	go mod tidy

# Run tests
test: ginkgo tidy mocks
	$(GINKGO) --cover -r -output-dir=. -coverprofile=coverage.out

lint: golangci-lint
	$(GOLANGCI_LINT) run --fix

# Run tests
helm-lint: helm
	helm lint helm/example-batch-job-controller/ --set routes.hostSuffix=test.com --strict

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
	goreleaser --clean

test-release:
	goreleaser --skip=publish --snapshot --clean

tools: mockgen ginkgo helm

helm:
ifeq (, $(shell which helm))
 $(shell curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash)
endif

docs: helm-docs update-docs
	@$(LOCALBIN)/helm-docs

update-docs: semver
	@version=$$($(LOCALBIN)/semver -next); \
	versionNum=$$($(LOCALBIN)/semver -next -numeric); \
	sed -i "s/^version:.*$$/version: $${versionNum}/"    ./helm/example-batch-job-controller/Chart.yaml; \
	sed -i "s/^appVersion:.*$$/appVersion: $${version}/" ./helm/example-batch-job-controller/Chart.yaml

