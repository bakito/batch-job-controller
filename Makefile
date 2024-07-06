

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

## toolbox - start
## Current working directory
LOCALDIR ?= $(shell which cygpath > /dev/null 2>&1 && cygpath -m $$(pwd) || pwd)
## Location to install dependencies to
LOCALBIN ?= $(LOCALDIR)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GINKGO ?= $(LOCALBIN)/ginkgo
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GORELEASER ?= $(LOCALBIN)/goreleaser
HELM_DOCS ?= $(LOCALBIN)/helm-docs
MOCKGEN ?= $(LOCALBIN)/mockgen
SEMVER ?= $(LOCALBIN)/semver

## Tool Versions
GOLANGCI_LINT_VERSION ?= v1.59.1
GORELEASER_VERSION ?= v2.0.1
HELM_DOCS_VERSION ?= v1.14.2
MOCKGEN_VERSION ?= v0.4.0
SEMVER_VERSION ?= v1.1.3

## Tool Installer
.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download ginkgo locally if necessary.
$(GINKGO): $(LOCALBIN)
	test -s $(LOCALBIN)/ginkgo || GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo
.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint || GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
.PHONY: goreleaser
goreleaser: $(GORELEASER) ## Download goreleaser locally if necessary.
$(GORELEASER): $(LOCALBIN)
	test -s $(LOCALBIN)/goreleaser || GOBIN=$(LOCALBIN) go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
.PHONY: helm-docs
helm-docs: $(HELM_DOCS) ## Download helm-docs locally if necessary.
$(HELM_DOCS): $(LOCALBIN)
	test -s $(LOCALBIN)/helm-docs || GOBIN=$(LOCALBIN) go install github.com/norwoodj/helm-docs/cmd/helm-docs@$(HELM_DOCS_VERSION)
.PHONY: mockgen
mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
$(MOCKGEN): $(LOCALBIN)
	test -s $(LOCALBIN)/mockgen || GOBIN=$(LOCALBIN) go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
.PHONY: semver
semver: $(SEMVER) ## Download semver locally if necessary.
$(SEMVER): $(LOCALBIN)
	test -s $(LOCALBIN)/semver || GOBIN=$(LOCALBIN) go install github.com/bakito/semver@$(SEMVER_VERSION)

## Update Tools
.PHONY: update-toolbox-tools
update-toolbox-tools:
	@rm -f \
		$(LOCALBIN)/ginkgo \
		$(LOCALBIN)/golangci-lint \
		$(LOCALBIN)/goreleaser \
		$(LOCALBIN)/helm-docs \
		$(LOCALBIN)/mockgen \
		$(LOCALBIN)/semver
	toolbox makefile -f $(LOCALDIR)/Makefile \
		github.com/golangci/golangci-lint/cmd/golangci-lint \
		github.com/goreleaser/goreleaser/v2 \
		github.com/norwoodj/helm-docs/cmd/helm-docs \
		go.uber.org/mock/mockgen@github.com/uber/mock \
		github.com/bakito/semver
## toolbox - end


