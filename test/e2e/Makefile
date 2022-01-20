.DEFAULT_GOAL:=help

IMG ?= public.ecr.aws/a4z9h2b1/cluster-api-provider-capc:latest

GOPROXY := $(shell go env GOPROXY)
ifeq ($(GOPROXY),)
GOPROXY := https://proxy.golang.org
endif
export GOPROXY

REPO_ROOT := $(shell git rev-parse --show-toplevel)

help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Binaries
## --------------------------------------

BIN_DIR := $(REPO_ROOT)/bin
GINKGO := $(BIN_DIR)/ginkgo
KUSTOMIZE := $(BIN_DIR)/kustomize
CONTROLLER_GEN := $(BIN_DIR)/controller-gen

$(GINKGO): # Build ginkgo from bin folder.
	@mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install github.com/onsi/ginkgo/ginkgo@v1.16.5

$(KUSTOMIZE): # Build kustomize from bin folder.
	@mkdir -p $(BIN_DIR)
	cd $(BIN_DIR) && curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash

$(CONTROLLER_GEN): ## Install controller-gen to bin.
	@mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.9

## --------------------------------------
## Kind Cluster
## --------------------------------------

.PHONY: kind-cluster
kind-cluster: $(REPO_ROOT)/cluster-api $(REPO_ROOT)/cluster-api/hack/kind-install-for-capd.sh # Create a kind cluster with a local Docker repository.
	-$(REPO_ROOT)/cluster-api/hack/kind-install-for-capd.sh

$(REPO_ROOT)/cluster-api: # Clone cluster-api repository for tilt use.
	git clone --branch v0.3.24 https://github.com/kubernetes-sigs/cluster-api.git $(REPO_ROOT)/cluster-api

# Need script from CAPI v1.0+
# Can delete this target after upgrading to newer CAPI.
$(REPO_ROOT)/cluster-api/hack/kind-install-for-capd.sh: $(REPO_ROOT)/cluster-api
	cd $(REPO_ROOT)/cluster-api && git checkout v1.0.0 -- hack/kind-install-for-capd.sh

## --------------------------------------
## Manifests
## --------------------------------------

MANIFEST_GEN_INPUTS=$(shell find $(REPO_ROOT)/api -type f -name "*test*" -prune -o -name "*zz_generated*" -prune -o -print)
.PHONY: manifests
manifests: $(REPO_ROOT)/config/.flag.mk
$(REPO_ROOT)/config/.flag.mk: $(CONTROLLER_GEN) $(MANIFEST_GEN_INPUTS)
	cd $(REPO_ROOT)
	sed -e 's@image: .*@image: '"$(IMG)"'@' $(REPO_ROOT)/config/default/manager_image_patch.yaml > $(REPO_ROOT)/config/default/manager_image_patch_edited.yaml
	$(CONTROLLER_GEN) crd:crdVersions=v1 rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=$(REPO_ROOT)/config/crd/bases
	@touch $(REPO_ROOT)/config/.flag.mk

## --------------------------------------
## Templates
## --------------------------------------

CLOUDSTACK_TEMPLATES := $(REPO_ROOT)/test/e2e/data/infrastructure-cloudstack

.PHONY: cluster-templates
cluster-templates: cluster-templates-v1alpha3

.PHONY: cluster-templates-v1alpha3
cluster-templates-v1alpha3: $(KUSTOMIZE)
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template > $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-kcp-remediation > $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-kcp-remediation.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-md-remediation > $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-md-remediation.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-invalid-zone > $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-invalid-zone.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-invalid-template > $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-invalid-template.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-invalid-cp-offering > $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-invalid-cp-offering.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-invalid-worker-offering > $(CLOUDSTACK_TEMPLATES)/v1alpha3/cluster-template-invalid-worker-offering.yaml

## --------------------------------------
## End-to-End Tests
## --------------------------------------

.PHONY: run-e2e
run-e2e: $(GINKGO) kind-cluster manifests cluster-templates ## Run End-to-End Tests
	$(GINKGO) -v -trace -tags=e2e -skip=Conformance -nodes=1 --noColor=false ./... -- \
	    -e2e.artifacts-folder=${REPO_ROOT}/_artifacts \
	    -e2e.config=${REPO_ROOT}/test/e2e/config/cloudstack.yaml \
	    -e2e.skip-resource-cleanup=false -e2e.use-existing-cluster=true
	kind delete clusters capi-test

.PHONY: run-e2e-pr-blocking
run-e2e-pr-blocking: $(GINKGO) kind-cluster manifests cluster-templates ## Run Only End-to-End Tests containing PR-Blocking tag
	echo $(CLOUDSTACK_B64ENCODED_SECRET)
	export CLOUDSTACK_B64ENCODED_SECRET=$(echo ${CLOUDSTACK_B64ENCODED_SECRET} | base64 --)
	echo $(CLOUDSTACK_B64ENCODED_SECRET)
	$(GINKGO) -v -trace -tags=e2e -focus=PR-Blocking -skip=Conformance -nodes=1 --noColor=false ./... -- \
	    -e2e.artifacts-folder=${REPO_ROOT}/_artifacts \
	    -e2e.config=${REPO_ROOT}/test/e2e/config/cloudstack.yaml \
	    -e2e.skip-resource-cleanup=false -e2e.use-existing-cluster=true
	kind delete clusters capi-test

.PHONY: run-conformance
run-conformance: $(GINKGO) kind-cluster manifests cluster-templates ## Run K8S Conformance Tests
	$(GINKGO) -v -trace -tags=e2e -focus=Conformance -nodes=1 --noColor=false ./... -- \
	    -e2e.artifacts-folder=${REPO_ROOT}/_artifacts \
	    -e2e.config=${REPO_ROOT}/test/e2e/config/cloudstack.yaml \
	    -e2e.skip-resource-cleanup=false -e2e.use-existing-cluster=true
	kind delete clusters capi-test
