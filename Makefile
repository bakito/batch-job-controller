# Include toolbox tasks
include ./.toolbox.mk

# generate mocks
mocks: tb.mockgen
	$(TB_MOCKGEN) -destination pkg/mocks/lifecycle/mock.go github.com/bakito/batch-job-controller/pkg/lifecycle Controller
	$(TB_MOCKGEN) -destination pkg/mocks/logr/mock.go      github.com/go-logr/logr                              LogSink
	$(TB_MOCKGEN) -destination pkg/mocks/record/mock.go    k8s.io/client-go/tools/record                        EventRecorder
	$(TB_MOCKGEN) -destination pkg/mocks/client/mock.go    sigs.k8s.io/controller-runtime/pkg/client            Client,Reader
	$(TB_MOCKGEN) -destination pkg/mocks/manager/mock.go   sigs.k8s.io/controller-runtime/pkg/manager           Manager

# Run go mod tidy
tidy:
	go mod tidy

# Run tests
test: tb.ginkgo tidy mocks
	$(TB_GINKGO) --cover -r -output-dir=. -coverprofile=coverage.out

lint: tb.golangci-lint
	$(TB_GOLANGCI_LINT) run --fix

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

release: tb.semver tb.goreleaser
	@version=$$($(TB_SEMVER)); \
	git tag -s $$version -m"Release $$version"
	$(TB_GORELEASER) --clean

test-release: tb.goreleaser
	$(TB_GORELEASER) --skip=publish --snapshot --clean

tools: tb.mockgen tb.ginkgo helm

helm:
ifeq (, $(shell which helm))
 $(shell curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash)
endif

helm-docs: tb.helm-docs update-docs
	@$(TB_HELM_DOCS)

update-docs: tb.semver
	@version=$$($(TB_SEMVER) -next); \
	versionNum=$$($(TB_SEMVER) -next -numeric); \
	sed -i "s/^version:.*$$/version: $${versionNum}/"    ./helm/example-batch-job-controller/Chart.yaml; \
	sed -i "s/^appVersion:.*$$/appVersion: $${version}/" ./helm/example-batch-job-controller/Chart.yaml

