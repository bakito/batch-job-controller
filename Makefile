

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
	goreleaser --rm-dist

test-release:
	goreleaser --skip-publish --snapshot --rm-dist

tools: mockgen ginkgo helm

helm:
ifeq (, $(shell which helm))
 $(shell curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash)
endif



## toolbox - start
## Current working directory
LOCALDIR ?= $(shell which cygpath > /dev/null 2>&1 && cygpath -m $$(pwd) || pwd)
## Location to install dependencies to
LOCALBIN ?= $(LOCALDIR)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GINKGO ?= $(LOCALBIN)/ginkgo
HELM_DOCS ?= $(LOCALBIN)/helm-docs
MOCKGEN ?= $(LOCALBIN)/mockgen
SEMVER ?= $(LOCALBIN)/semver

## Tool Installer
.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download ginkgo locally if necessary.
$(GINKGO): $(LOCALBIN)
	test -s $(LOCALBIN)/ginkgo || GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo
.PHONY: helm-docs
helm-docs: $(HELM_DOCS) ## Download helm-docs locally if necessary.
$(HELM_DOCS): $(LOCALBIN)
	test -s $(LOCALBIN)/helm-docs || GOBIN=$(LOCALBIN) go install github.com/norwoodj/helm-docs/cmd/helm-docs
.PHONY: mockgen
mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
$(MOCKGEN): $(LOCALBIN)
	test -s $(LOCALBIN)/mockgen || GOBIN=$(LOCALBIN) go install go.uber.org/mock/mockgen
.PHONY: semver
semver: $(SEMVER) ## Download semver locally if necessary.
$(SEMVER): $(LOCALBIN)
	test -s $(LOCALBIN)/semver || GOBIN=$(LOCALBIN) go install github.com/bakito/semver

## Update Tools
.PHONY: update-toolbox-tools
update-toolbox-tools:
	@rm -f \
		$(LOCALBIN)/ginkgo \
		$(LOCALBIN)/helm-docs \
		$(LOCALBIN)/mockgen \
		$(LOCALBIN)/semver
	toolbox makefile -f $(LOCALDIR)/Makefile
## toolbox - end


