
# Image URL to use all building/pushing image targets
IMG ?= public.ecr.aws/a4z9h2b1/cluster-api-provider-capc:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:crdVersions=v1,preserveUnknownFields=false"

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
VERSION ?= v0.1.0

# Allow overriding manifest generation destination directory
MANIFEST_ROOT ?= ./config
CRD_ROOT ?= $(MANIFEST_ROOT)/crd/bases
WEBHOOK_ROOT ?= $(MANIFEST_ROOT)/webhook
RBAC_ROOT ?= $(MANIFEST_ROOT)/rbac
RELEASE_DIR := out
BUILD_DIR := .build
OVERRIDES_DIR := $(HOME)/.cluster-api/overrides/infrastructure-cloudstack/$(VERSION)

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

##@ Development
$(RELEASE_DIR):
	@mkdir -p $(RELEASE_DIR)


$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

$(OVERRIDES_DIR):
	@mkdir -p $(OVERRIDES_DIR)

.PHONY: release-manifests
release-manifests:
	$(MAKE) manifests STAGE=release MANIFEST_DIR=$(RELEASE_DIR)
	cp metadata.yaml $(RELEASE_DIR)/metadata.yaml

.PHONY: dev-manifests
dev-manifests:
	$(MAKE) manifests STAGE=dev MANIFEST_DIR=$(OVERRIDES_DIR)
	cp metadata.yaml $(OVERRIDES_DIR)/metadata.yaml

.PHONY: manifests
manifests: kustomize $(MANIFEST_DIR) $(BUILD_DIR) $(KUSTOMIZE) $(STAGE)-cluster-templates
	rm -rf $(BUILD_DIR)/config
	cp -R config $(BUILD_DIR)
	"$(KUSTOMIZE)" build $(BUILD_DIR)/config/default > $(MANIFEST_DIR)/infrastructure-components.yaml

.PHONY: dev-cluster-templates
dev-cluster-templates:
	cp templates/cluster-template.yaml $(OVERRIDES_DIR)/cluster-template.yaml

.PHONY: release-cluster-templates
release-cluster-templates:
	cp templates/cluster-template.yaml $(RELEASE_DIR)/cluster-template.yaml

.PHONY: generate
generate: ## Generate code and manifests
	$(MAKE) generate-go
	$(MAKE) generate-manifests

.PHONY: generate-manifests
generate-manifests: $(CONTROLLER_GEN) ## Generate manifests e.g. CRD, RBAC etc.
	$(CONTROLLER_GEN) \
		paths=./api/... \
		crd:crdVersions=v1 \
		output:crd:dir=$(CRD_ROOT) \
		output:webhook:dir=$(WEBHOOK_ROOT) \
		webhook
	$(CONTROLLER_GEN) \
		paths=./controllers/... \
		output:rbac:dir=$(RBAC_ROOT) \
		rbac:roleName=manager-role

.PHONY: generate-go
generate-go: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin

.PHONY: test
test: generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

##@ Build
.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Linting
GOLANGCI_LINT = $(HOME)/go/bin/golangci-lint
golangci-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0

.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run ./...

##@ Deployment

.PHONY: install
install: generate-manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: generate-manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

.PHONY: deploy
deploy: generate-manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

##@ Cleanup
.PHONY: clean
clean: ## Run all the clean targets
	$(MAKE) clean-temporary
	$(MAKE) clean-release
	$(MAKE) clean-examples
	$(MAKE) clean-build

.PHONY: clean-build
clean-build:
	rm -rf $(BUILD_DIR)

.PHONY: clean-temporary
clean-temporary: ## Remove all temporary files and folders
	rm -f minikube.kubeconfig
	rm -f kubeconfig

.PHONY: clean-release
clean-release: ## Remove the release folder
	rm -rf $(RELEASE_DIR)

.PHONY: clean-examples
clean-examples: ## Remove all the temporary files generated in the examples folder
	rm -rf examples/_out/
	rm -f examples/provider-components/provider-components-*.yaml

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

## --------------------------------------
## Testing
## --------------------------------------

CLOUDSTACK_TEMPLATES := $(PROJECT_DIR)/test/e2e/data/infrastructure-cloudstack
GINKGO_FOCUS  ?=
GINKGO_FOCUS_CONFORMANCE ?= "\\[Conformance\\]"
GINKGO_SKIP ?= "\\[Conformance\\]"
GINKGO_NODES  ?= 1
E2E_CONF_FILE  ?= ${PROJECT_DIR}/test/e2e/config/cloudstack.yaml
ARTIFACTS ?= ${PROJECT_DIR}/_artifacts
SKIP_RESOURCE_CLEANUP ?= false
USE_EXISTING_CLUSTER ?= false
GINKGO_NOCOLOR ?= false

GINKGO = $(shell pwd)/bin/ginkgo
ginkgo: ## Download ginkgo locally if necessary.
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/ginkgo)

# to set multiple ginkgo skip flags, if any
ifneq ($(strip $(GINKGO_SKIP)),)
_SKIP_ARGS := $(foreach arg,$(strip $(GINKGO_SKIP)),-skip="$(arg)")
endif

.PHONY: cluster-templates
cluster-templates: kustomize cluster-templates-v1alpha4 ## Generate cluster templates for all versions

.PHONY: cluster-templates-v1alpha4
cluster-templates-v1alpha4: kustomize ## Generate cluster templates for v1alpha4
	$(KUSTOMIZE) build $(CLOUDSTACK_TEMPLATES)/v1alpha4/cluster-template --load_restrictor none > $(CLOUDSTACK_TEMPLATES)/v1alpha4/cluster-template.yaml
	$(KUSTOMIZE) build $(CLOUDSTACK_TEMPLATES)/v1alpha4/cluster-template-kcp-remediation --load_restrictor none > $(CLOUDSTACK_TEMPLATES)/v1alpha4/cluster-template-kcp-remediation.yaml
	$(KUSTOMIZE) build $(CLOUDSTACK_TEMPLATES)/v1alpha4/cluster-template-md-remediation --load_restrictor none > $(CLOUDSTACK_TEMPLATES)/v1alpha4/cluster-template-md-remediation.yaml

.PHONY: run-e2e
run-e2e: ginkgo cluster-templates test-e2e-image-prerequisites ## Run the end-to-end tests
	time $(GINKGO) -v -trace -tags=e2e -focus="$(GINKGO_FOCUS)" $(_SKIP_ARGS) -nodes=$(GINKGO_NODES) --noColor=$(GINKGO_NOCOLOR) $(GINKGO_ARGS) ./test/e2e/... -- \
	    -e2e.artifacts-folder="$(ARTIFACTS)" \
	    -e2e.config="$(E2E_CONF_FILE)" \
	    -e2e.skip-resource-cleanup=$(SKIP_RESOURCE_CLEANUP) -e2e.use-existing-cluster=$(USE_EXISTING_CLUSTER)

.PHONY: run-conformance
run-conformance: ginkgo cluster-templates test-e2e-image-prerequisites ## Run the k8s conformance tests
	time $(GINKGO) -v -trace -tags=e2e -focus="$(GINKGO_FOCUS_CONFORMANCE)" -nodes=$(GINKGO_NODES) --noColor=$(GINKGO_NOCOLOR) $(GINKGO_ARGS) ./test/e2e/... -- \
	    -e2e.artifacts-folder="$(ARTIFACTS)" \
	    -e2e.config="$(E2E_CONF_FILE)" \
	    -e2e.skip-resource-cleanup=$(SKIP_RESOURCE_CLEANUP) -e2e.use-existing-cluster=$(USE_EXISTING_CLUSTER)

test-e2e-image-prerequisites:
	docker pull quay.io/jetstack/cert-manager-cainjector:v1.5.3
	docker pull quay.io/jetstack/cert-manager-webhook:v1.5.3
	docker pull quay.io/jetstack/cert-manager-controller:v1.5.3
	docker pull gcr.io/k8s-staging-cluster-api/cluster-api-controller:v0.4.3
	docker pull gcr.io/k8s-staging-cluster-api/kubeadm-bootstrap-controller:v0.4.3
	docker pull gcr.io/k8s-staging-cluster-api/kubeadm-control-plane-controller:v0.4.3
