# Image URL to use all building/pushing image targets
IMG ?= public.ecr.aws/a4z9h2b1/cluster-api-provider-capc:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec
VERSION ?= v0.2.0

# Allow overriding release-manifest generation destination directory
RELEASE_DIR ?= out

# Quiet Ginkgo for now.
# The warnings are in regards to a future release.
export ACK_GINKGO_DEPRECATIONS := 1.16.5
export ACK_GINKGO_RC=true

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
export PATH := $(PROJECT_DIR)/bin:$(PATH)

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: manifests
# Using a flag file here as config output is too complicated to be a target.
manifests: config/.flag.mk ## Generates crd, webhook, rbac, and other configuration manifests from kubebuilder instructions in go comments.
config/.flag.mk: bin/controller-gen $(shell find ./controllers ./api -type f -name "*test*" -prune -o -print) # This flags that we've recently generated the configuration manifests directory.
	controller-gen crd:crdVersions=v1 rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	@touch config/.flag.mk

.PHONY: release-manifests
release-manifests: $(RELEASE_DIR)/* manifests ## Create kustomized release manifest in $RELEASE_DIR (defaults to out).
$(RELEASE_DIR)/%: $(shell find config)
	sed -i'' -e 's@image: .*@image: '"$(IMG)"'@' config/default/manager_image_patch.yaml
	@mkdir -p $(RELEASE_DIR)
	cp metadata.yaml $(RELEASE_DIR)/metadata.yaml
	kustomize build config/default > $(RELEASE_DIR)/infrastructure-components.yaml

.PHONY: generate-deepcopy
generate-deepcopy: bin/controller-gen $(shell find api -type f -name zz_generated.deepcopy.go) ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
api/%/zz_generated.deepcopy.go: $(shell find ./api -type f -name "*test*" -prune -o -name "*zz_generated*" -prune -o -print)
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

##@ Build

.PHONY: build
build: binaries generate-deepcopy manifests release-manifests bin/manager bin/mockgen ## Build manager binary.
bin/manager: $(shell find ./controllers ./api -type f -name "*test*" -prune -o -print)
	go fmt ./...
	go vet ./...
	go build -o bin/manager main.go

.PHONY: run
run: generate-deepcopy fmt vet ## Run a controller from your host.
	go fmt ./...
	go vet ./...
	go run ./main.go

# Using a flag file here as docker build doesn't produce a target file.
.PHONY: docker-build
docker-build: .dockerflag.mk ## Build docker image containing the controller manager.
.dockerflag.mk: Dockerfile $(shell find ./controllers ./api -type f -name "*test*" -prune -o -print)
	docker build -t ${IMG} .
	@touch .dockerflag.mk

.PHONY: docker-push
docker-push: .dockerflag.mk ## Push docker image with the manager.
	docker push ${IMG}

##@ Linting

.PHONY: fmt
fmt: ## Run go fmt on the whole project.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet on the whole project.
	go vet ./...

.PHONY: lint
lint: bin/golangci-lint ## Run linting for the project.
	go fmt ./...
	go vet ./...
	golangci-lint run ./...
	@ # The below string of commands checks that ginkgo isn't present in the controllers.
	@(grep ginkgo ${PROJECT_DIR}/controllers/cloudstack*_controller.go && \
		echo "Remove ginkgo from controllers. This is probably an artifact of testing." \
		 	 "See the hack/testing_ginkgo_recover_statements.sh file") && exit 1 || \
		echo "Gingko statements not found in controllers... (passed)"

##@ Deployment

.PHONY: deploy
deploy: generate-deepcopy manifests bin/kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

undeploy: bin/kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kustomize build config/default | kubectl delete -f -

##@ Binaries

.PHONY: binaries
binaries: bin/controller-gen bin/kustomize bin/ginkgo bin/golangci-lint bin/mockgen ## Locally install all needed bins.
bin/controller-gen: ## Install controller-gen to bin.
	GOBIN=$(PROJECT_DIR)/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.9
bin/golangci-lint: ## Install golangci-lint to bin.
	GOBIN=$(PROJECT_DIR)/bin go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0
bin/ginkgo: ## Install ginkgo to bin.
	GOBIN=$(PROJECT_DIR)/bin go install github.com/onsi/ginkgo/ginkgo@v1.16.5
bin/mockgen:
	GOBIN=$(PROJECT_DIR)/bin go install github.com/golang/mock/mockgen@v1.6.0
bin/kustomize: ## Install kustomize to bin.
	@mkdir -p bin
	cd bin && curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash

##@ Cleanup

.PHONY: clean
clean: ## Clean.
	rm -rf $(RELEASE_DIR)
	rm -rf bin

##@ Testing

.PHONY: test
test: lint generate-deepcopy generate-mocks bin/ginkgo ## Run tests. At the moment this is only unit tests.
	@./hack/testing_ginkgo_recover_statements.sh --add # Add ginkgo.GinkgoRecover() statements to controllers.
	@# The following is a slightly funky way to make sure the ginkgo statements are removed regardless the test results.
	@ginkgo -v ./api/... ./controllers/... ./pkg/... -coverprofile cover.out; EXIT_STATUS=$$?;\
		./hack/testing_ginkgo_recover_statements.sh --remove; exit $$EXIT_STATUS
	
.PHONY: generate-mocks
generate-mocks: $(shell find ./pkg/mocks -type f -name "mock*.go") ## Generate mocks needed for testing. Primarily mocks of the cloud package.
pkg/mocks/mock%.go: $(shell find ./pkg/cloud -type f -name "*test*" -prune -o -print)
	go generate ./...

