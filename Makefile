# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

export REPO_ROOT := $(shell git rev-parse --show-toplevel)

include $(REPO_ROOT)/common.mk

# Directories
TOOLS_DIR := $(REPO_ROOT)/hack/tools
TOOLS_DIR_DEPS := $(TOOLS_DIR)/go.sum $(TOOLS_DIR)/go.mod $(TOOLS_DIR)/Makefile
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
BIN_DIR ?= bin
RELEASE_DIR ?= out

GH_REPO ?= kubernetes-sigs/cluster-api-provider-cloudstack

# Binaries
CONTROLLER_GEN := $(TOOLS_BIN_DIR)/controller-gen
CONVERSION_GEN := $(TOOLS_BIN_DIR)/conversion-gen
GINKGO := $(TOOLS_BIN_DIR)/ginkgo
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
KUSTOMIZE := $(TOOLS_BIN_DIR)/kustomize
MOCKGEN := $(TOOLS_BIN_DIR)/mockgen
STATIC_CHECK := $(TOOLS_BIN_DIR)/staticcheck
KUBECTL := $(TOOLS_BIN_DIR)/kubectl
API_SERVER := $(TOOLS_BIN_DIR)/kube-apiserver
ETCD := $(TOOLS_BIN_DIR)/etcd

# Release
STAGING_REGISTRY := gcr.io/k8s-staging-capi-cloudstack
STAGING_BUCKET ?= artifacts.k8s-staging-capi-cloudstack.appspot.com
BUCKET ?= $(STAGING_BUCKET)
PROD_REGISTRY ?= registry.k8s.io/capi-cloudstack
REGISTRY ?= $(STAGING_REGISTRY)
RELEASE_TAG ?= $(shell git describe --abbrev=0 2>/dev/null)
PULL_BASE_REF ?= $(RELEASE_TAG)
RELEASE_ALIAS_TAG ?= $(PULL_BASE_REF)

# Image URL to use all building/pushing image targets
REGISTRY ?= $(STAGING_REGISTRY)
IMAGE_NAME ?= capi-cloudstack-controller
TAG ?= dev
CONTROLLER_IMG ?= $(REGISTRY)/$(IMAGE_NAME)
IMG ?= $(CONTROLLER_IMG):$(TAG)
IMG_LOCAL ?= localhost:5000/$(IMAGE_NAME):$(TAG)
MANIFEST_FILE := infrastructure-components
CONFIG_DIR := config
NAMESPACE := capc-system

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
# SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

export PATH := $(TOOLS_BIN_DIR):$(PATH)

all: build

##@ Binaries
## --------------------------------------
## Binaries
## --------------------------------------

.PHONY: binaries
binaries: $(CONTROLLER_GEN) $(CONVERSION_GEN) $(GOLANGCI_LINT) $(STATIC_CHECK) $(GINKGO) $(MOCKGEN) $(KUSTOMIZE) managers # Builds and installs all binaries

.PHONY: managers
managers:
	$(MAKE) manager-cloudstack-infrastructure

.PHONY: manager-cloudstack-infrastructure
manager-cloudstack-infrastructure: ## Build manager binary.
	CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -ldflags "${LDFLAGS} -extldflags '-static'" -o $(BIN_DIR)/manager .

export K8S_VERSION=1.28.3
$(KUBECTL) $(API_SERVER) $(ETCD) &:
	cd $(TOOLS_DIR) && curl --silent -L "https://go.kubebuilder.io/test-tools/${K8S_VERSION}/$(shell go env GOOS)/$(shell go env GOARCH)" --output - | \
		tar -C ./ --strip-components=1 -zvxf -

##@ Linting
## --------------------------------------
## Linting
## --------------------------------------

.PHONY: fmt
fmt: ## Run go fmt on the whole project.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet on the whole project.
	go vet ./...

.PHONY: lint
lint: $(GOLANGCI_LINT) $(STATIC_CHECK) generate-mocks ## Run linting for the project.
	$(MAKE) fmt
	$(MAKE) vet
	$(GOLANGCI_LINT) run -v --timeout 360s ./...
	$(STATIC_CHECK) ./...
	@ # The below string of commands checks that ginkgo isn't present in the controllers.
	@(grep ginkgo ${REPO_ROOT}/controllers/cloudstack*_controller.go | grep -v import && \
		echo "Remove ginkgo from controllers. This is probably an artifact of testing." \
		 	 "See the hack/testing_ginkgo_recover_statements.sh file") && exit 1 || \
		echo "Gingko statements not found in controllers... (passed)"


##@ Generate
## --------------------------------------
## Generate
## --------------------------------------

.PHONY: modules
modules: ## Runs go mod to ensure proper vendoring.
	go mod tidy -compat=1.21
	cd $(TOOLS_DIR); go mod tidy -compat=1.21

.PHONY: generate-all
generate-all: generate-mocks generate-deepcopy generate-manifests

.PHONY: generate-mocks
generate-mocks: $(MOCKGEN) generate-deepcopy pkg/mocks/mock_client.go $(shell find ./pkg/mocks -type f -name "mock*.go") ## Generate mocks needed for testing. Primarily mocks of the cloud package.
pkg/mocks/mock%.go: $(shell find ./pkg/cloud -type f -name "*test*" -prune -o -print)
	go generate ./...

DEEPCOPY_GEN_TARGETS=$(shell find api -type d -name "v*" -exec echo {}\/zz_generated.deepcopy.go \;)
DEEPCOPY_GEN_INPUTS=$(shell find ./api -name "*test*" -prune -o -name "*zz_generated*" -prune -o -type f -print)
.PHONY: generate-deepcopy
generate-deepcopy: $(DEEPCOPY_GEN_TARGETS) ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
api/%/zz_generated.deepcopy.go: $(CONTROLLER_GEN) $(DEEPCOPY_GEN_INPUTS)
	CGO_ENABLED=0 $(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

MANIFEST_GEN_INPUTS=$(shell find ./api ./controllers -type f -name "*test*" -prune -o -name "*zz_generated*" -prune -o -print)
# Using a flag file here as config output is too complicated to be a target.
# The following triggers manifest building if $(IMG) differs from that found in config/default/manager_image_patch.yaml.
$(shell	grep -qs "$(IMG)" config/default/manager_image_patch_edited.yaml || rm -f config/.flag.mk)
.PHONY: generate-manifests
generate-manifests: config/.flag.mk ## Generates crd, webhook, rbac, and other configuration manifests from kubebuilder instructions in go comments.
config/.flag.mk: $(CONTROLLER_GEN) $(MANIFEST_GEN_INPUTS)
	sed -e 's@image: .*@image: '"$(IMG)"'@' config/default/manager_image_patch.yaml > config/default/manager_image_patch_edited.yaml
	$(CONTROLLER_GEN) crd:crdVersions=v1 rbac:roleName=manager-role webhook paths="{./api/...,./controllers/...}" output:crd:artifacts:config=config/crd/bases
	@touch config/.flag.mk

.PHONY: generate-conversion
generate-conversion: $(CONVERSION_GEN) ## Generate code to convert api/v1beta1 and api/v1beta2 to api/v1beta3
	$(CONVERSION_GEN) \
		--input-dirs=./api/v1beta1 \
		--go-header-file=./hack/boilerplate.go.txt \
		--output-base=. --output-file-base=zz_generated.conversion
	$(CONVERSION_GEN) \
		--input-dirs=./api/v1beta2 \
		--go-header-file=./hack/boilerplate.go.txt \
		--output-base=. --output-file-base=zz_generated.conversion

##@ Build
## --------------------------------------
## Build
## --------------------------------------

MANAGER_BIN_INPUTS=$(shell find ./controllers ./api ./pkg -name "*mock*" -prune -o -name "*test*" -prune -o -type f -print) main.go go.mod go.sum
.PHONY: build
build: binaries generate-deepcopy lint generate-manifests release-manifests ## Build manager binary.
$(BIN_DIR)/manager: $(MANAGER_BIN_INPUTS)
	go build -o $(BIN_DIR)/manager main.go

.PHONY: build-for-docker
build-for-docker: $(BIN_DIR)/manager-linux-amd64 ## Build manager binary for docker image building.
$(BIN_DIR)/manager-linux-amd64: $(MANAGER_BIN_INPUTS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    	go build -a -ldflags "${ldflags} -extldflags '-static'" \
    	-o $(BIN_DIR)/manager-linux-amd64 main.go

.PHONY: run
run: generate-deepcopy generate-conversion ## Run a controller from your host.
	go run ./main.go

##@ Deploy
## --------------------------------------
## Deploy
## --------------------------------------

.PHONY: deploy
deploy: generate-deepcopy generate-manifests $(KUSTOMIZE) ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	cd $(REPO_ROOT)
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: $(KUSTOMIZE) ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

##@ Docker
## --------------------------------------
## Docker
## --------------------------------------

# Using a flag file here as docker build doesn't produce a target file.
DOCKER_BUILD_INPUTS=$(MANAGER_BIN_INPUTS) Dockerfile
.PHONY: docker-build
docker-build: generate-deepcopy generate-conversion build-for-docker .dockerflag.mk ## Build docker image containing the controller manager.
.dockerflag.mk: $(DOCKER_BUILD_INPUTS)
	docker build -t ${IMG} .
	@touch .dockerflag.mk

.PHONY: docker-push
docker-push: .dockerflag.mk ## Push docker image with the manager.
	docker push ${IMG}

##@ Tilt
## --------------------------------------
## Tilt Development
## --------------------------------------

.PHONY: tilt-up
tilt-up: cluster-api create-kind-cluster cluster-api/tilt-settings.json generate-manifests ## Setup and run tilt for development.
	cd cluster-api && tilt up

KIND_CLUSTER_NAME := $(shell cat ./hack/tilt-settings.json | grep kind_cluster_name | cut -d: -f2 | xargs)

.PHONY: create-kind-cluster
create-kind-cluster: cluster-api cluster-api/tilt-settings.json ## Create a kind cluster with a local Docker repository.
	@if [ -z "$$(kind get clusters | grep $(KIND_CLUSTER_NAME))" ]; then \
		CAPI_KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) ./cluster-api/hack/kind-install-for-capd.sh; \
	fi;

.PHONY: delete-kind-cluster
delete-kind-cluster:
	kind delete cluster --name $(KIND_CLUSTER_NAME)

cluster-api: ## Clone cluster-api repository for tilt use.
	git clone --branch v1.6.8 --depth 1 https://github.com/kubernetes-sigs/cluster-api.git

cluster-api/tilt-settings.json: hack/tilt-settings.json cluster-api
	cp ./hack/tilt-settings.json cluster-api

##@ Tests
## --------------------------------------
## Tests
## --------------------------------------

export KUBEBUILDER_ASSETS=$(TOOLS_BIN_DIR)
DEEPCOPY_GEN_TARGETS_TEST=$(shell find test/fakes -type d -name "fakes" -exec echo {}\/zz_generated.deepcopy.go \;)
DEEPCOPY_GEN_INPUTS_TEST=$(shell find test/fakes/* -name "*zz_generated*" -prune -o -type f -print)
.PHONY: generate-deepcopy-test
generate-deepcopy-test: $(DEEPCOPY_GEN_TARGETS_TEST) ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
test/fakes/zz_generated.deepcopy.go: $(CONTROLLER_GEN) $(DEEPCOPY_GEN_INPUTS_TEST)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

MANIFEST_GEN_INPUTS_TEST=$(shell find test/fakes/* -name "*zz_generated*" -prune -o -type f -print)
.PHONY: generate-manifest-test
generate-manifest-test: config/.flag-test.mk ## Generates crd, webhook, rbac, and other configuration manifests from kubebuilder instructions in go comments.
config/.flag-test.mk: $(CONTROLLER_GEN) $(MANIFEST_GEN_INPUTS_TEST)
	$(CONTROLLER_GEN) crd:crdVersions=v1 rbac:roleName=manager-role webhook paths="./test/fakes" output:crd:artifacts:config=test/fakes
	@touch config/.flag-test.mk

.PHONY: test
test: ## Run tests.
test: generate-deepcopy-test generate-manifest-test generate-mocks lint $(GINKGO) $(KUBECTL) $(API_SERVER) $(ETCD)
	@./hack/testing_ginkgo_recover_statements.sh --add # Add ginkgo.GinkgoRecover() statements to controllers.
	@# The following is a slightly funky way to make sure the ginkgo statements are removed regardless the test results.
	@$(GINKGO) --label-filter="!integ" --cover -coverprofile cover.out --covermode=atomic -v ./api/... ./controllers/... ./pkg/...; EXIT_STATUS=$$?;\
		./hack/testing_ginkgo_recover_statements.sh --remove; exit $$EXIT_STATUS

CLUSTER_TEMPLATES_INPUT_FILES=$(shell find test/e2e/data/infrastructure-cloudstack/v1beta*/cluster-template* test/e2e/data/infrastructure-cloudstack/*/bases/* -type f)
CLUSTER_TEMPLATES_OUTPUT_FILES=$(shell find test/e2e/data/infrastructure-cloudstack -type d -name "cluster-template*" -exec echo {}.yaml \;)
.PHONY: e2e-cluster-templates
e2e-cluster-templates: $(CLUSTER_TEMPLATES_OUTPUT_FILES) ## Generate cluster template files for e2e testing.
cluster-template%yaml: $(KUSTOMIZE) $(CLUSTER_TEMPLATES_INPUT_FILES)
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(basename $@) > $@

e2e-essentials: $(GINKGO) $(KUBECTL) e2e-cluster-templates create-kind-cluster ## Fulfill essential tasks for e2e testing.
	IMG=$(IMG_LOCAL) make generate-manifests docker-build docker-push

JOB ?= .*
E2E_CONFIG ?= ${REPO_ROOT}/test/e2e/config/cloudstack.yaml
E2E_TIMEOUT ?= 3h
run-e2e: e2e-essentials ## Run e2e testing. JOB is an optional REGEXP to select certainn test cases to run. e.g. JOB=PR-Blocking, JOB=Conformance
	$(KUBECTL) apply -f cloud-config.yaml && \
	cd test/e2e && \
	$(GINKGO) -v --trace --tags=e2e --focus=$(JOB) --timeout=$(E2E_TIMEOUT) --skip=Conformance --skip-package=kubeconfig_helper --nodes=1 --no-color=false ./... -- \
	    -e2e.artifacts-folder=${REPO_ROOT}/_artifacts \
	    -e2e.config=${E2E_CONFIG} \
	    -e2e.skip-resource-cleanup=false -e2e.use-existing-cluster=true
	EXIT_STATUS=$$?
	kind delete clusters capi-test
	exit $$EXIT_STATUS

run-e2e-smoke:
	./hack/ensure-kind.sh
	./hack/ensure-cloud-config-yaml.sh
	JOB="\"CAPC E2E SMOKE TEST\"" $(MAKE) run-e2e

##@ Cleanup
## --------------------------------------
## Cleanup
## --------------------------------------

.PHONY: clean
clean: ## Cleans up everything.
	rm -rf $(RELEASE_DIR)
	rm -rf bin
	rm -rf $(TOOLS_BIN_DIR)
	rm -rf cluster-api
	rm -rf test/e2e/data/infrastructure-cloudstack/*/*yaml
	rm -rf config/.flag.mk config/.flag-test.mk .dockerflag.mk

##@ Release
## --------------------------------------
## Release
## --------------------------------------

.PHONY: release-manifests
RELEASE_MANIFEST_TARGETS=$(RELEASE_DIR)/infrastructure-components.yaml $(RELEASE_DIR)/metadata.yaml
RELEASE_MANIFEST_INPUTS=$(KUSTOMIZE) config/.flag.mk $(shell find config)
RELEASE_MANIFEST_SOURCE_BASE ?= config/default
release-manifests: $(RELEASE_MANIFEST_TARGETS) ## Create kustomized release manifest in $RELEASE_DIR (defaults to out).
$(RELEASE_DIR)/%: $(RELEASE_MANIFEST_INPUTS)
	@mkdir -p $(RELEASE_DIR)
	cp metadata.yaml $(RELEASE_DIR)/metadata.yaml
	$(KUSTOMIZE) build $(RELEASE_MANIFEST_SOURCE_BASE) > $(RELEASE_DIR)/infrastructure-components.yaml

.PHONY: release-manifests-metrics-port
release-manifests-metrics-port:
	make release-manifests RELEASE_MANIFEST_SOURCE_BASE=config/default-with-metrics-port

.PHONY: release-staging
release-staging: ## Builds and uploads manifests to the staging bucket and creates new tag
	$(MAKE) release-templates
	$(MAKE) release-manifests TAG=$(RELEASE_ALIAS_TAG)
	$(MAKE) upload-staging-artifacts
	git tag $(RELEASE_ALIAS_TAG)
	git push upstream $(RELEASE_ALIAS_TAG)

.PHONY: release-alias-tag
release-alias-tag: # Adds the tag to the last build tag.
	gcloud container images add-tag -q $(CONTROLLER_IMG):$(TAG) $(CONTROLLER_IMG):$(RELEASE_ALIAS_TAG)

.PHONY: release-templates
release-templates: ## Generate release templates
	@mkdir -p $(RELEASE_DIR)
	cp templates/cluster-template*.yaml $(RELEASE_DIR)/

.PHONY: upload-staging-artifacts
upload-staging-artifacts: ## Upload release artifacts to the staging bucket
	gsutil cp $(RELEASE_DIR)/* gs://$(STAGING_BUCKET)/components/$(RELEASE_ALIAS_TAG)/
