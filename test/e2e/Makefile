.DEFAULT_GOAL:=help

IMG ?= localhost:5000/cluster-api-provider-cloudstack:latest

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
kind-cluster: 
	rm -f kind-install-for-capd.sh
	wget https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/v1.0.0/hack/kind-install-for-capd.sh
	chmod +x kind-install-for-capd.sh
	-./kind-install-for-capd.sh

## --------------------------------------
## Manifests
## --------------------------------------

.PHONY: manifests
manifests: 
	d=$$(date +%s) \
	&& IMG=$(IMG) make -C $(REPO_ROOT) manifests \
	&& echo "This step took $$(($$(date +%s)-d)) seconds"

## --------------------------------------
## Templates
## --------------------------------------

CLOUDSTACK_TEMPLATES := $(REPO_ROOT)/test/e2e/data/infrastructure-cloudstack

.PHONY: cluster-templates
cluster-templates: cluster-templates-v1beta1

.PHONY: cluster-templates-v1beta1
cluster-templates-v1beta1: $(KUSTOMIZE)
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template > $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-kcp-remediation > $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-kcp-remediation.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-md-remediation > $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-md-remediation.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-zone > $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-zone.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-account > $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-account.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-domain > $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-domain.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-template > $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-template.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-cp-offering > $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-cp-offering.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-worker-offering > $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-invalid-worker-offering.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-insufficient-compute-resources > $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-insufficient-compute-resources.yaml
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-node-drain > $(CLOUDSTACK_TEMPLATES)/v1beta1/cluster-template-node-drain.yaml

## --------------------------------------
## Docker build and push to local registry
## --------------------------------------

.PHONY: docker-build-and-push
docker-build-and-push:
	d=$$(date +%s) \
	&& IMG=$(IMG) $(MAKE) -C $(REPO_ROOT) docker-build \
	&& IMG=$(IMG) $(MAKE) -C $(REPO_ROOT) docker-push \
	&& echo "This step took $$(($$(date +%s)-d)) seconds"


## --------------------------------------
## End-to-End Tests
## --------------------------------------

.PHONY: e2e-essentials
e2e-essentials: $(GINKGO) kind-cluster manifests cluster-templates docker-build-and-push

.PHONY: run-e2e
run-e2e: e2e-essentials ## Run End-to-End Tests
	d=$$(date +%s) \
	&& PATH=$(BIN_DIR):${PATH} $(GINKGO) -v -trace -tags=e2e -skip=Conformance -nodes=1 --noColor=false ./... -- \
	    -e2e.artifacts-folder=${REPO_ROOT}/_artifacts \
	    -e2e.config=${REPO_ROOT}/test/e2e/config/cloudstack.yaml \
	    -e2e.skip-resource-cleanup=false -e2e.use-existing-cluster=true \
	&& echo "This step took $$(($$(date +%s)-d)) seconds"
	kind delete clusters capi-test

.PHONY: run-e2e-pr-blocking
run-e2e-pr-blocking: e2e-essentials ## Run Only End-to-End Tests containing PR-Blocking tag
	d=$$(date +%s) \
	&& PATH=$(BIN_DIR):${PATH} $(GINKGO) -v -trace -tags=e2e -focus=PR-Blocking -skip=Conformance -nodes=1 --noColor=false ./... -- \
	    -e2e.artifacts-folder=${REPO_ROOT}/_artifacts \
	    -e2e.config=${REPO_ROOT}/test/e2e/config/cloudstack.yaml \
	    -e2e.skip-resource-cleanup=false -e2e.use-existing-cluster=true \
	&& echo "This step took $$(($$(date +%s)-d)) seconds"
	kind delete clusters capi-test

.PHONY: run-conformance
run-conformance: e2e-essentials ## Run K8S Conformance Tests
	d=$$(date +%s) \
	&& PATH=$(BIN_DIR):${PATH} $(GINKGO) -v -trace -tags=e2e -focus=Conformance -nodes=1 --noColor=false ./... -- \
	    -e2e.artifacts-folder=${REPO_ROOT}/_artifacts \
	    -e2e.config=${REPO_ROOT}/test/e2e/config/cloudstack.yaml \
	    -e2e.skip-resource-cleanup=false -e2e.use-existing-cluster=true \
	&& echo "This step took $$(($$(date +%s)-d)) seconds"
	kind delete clusters capi-test

.PHONY: run-all
run-all: e2e-essentials ## Run K8S Conformance Tests
	d=$$(date +%s) \
	&& PATH=$(BIN_DIR):${PATH} $(GINKGO) -v -trace -tags=e2e -nodes=1 --noColor=false ./... -- \
	    -e2e.artifacts-folder=${REPO_ROOT}/_artifacts \
	    -e2e.config=${REPO_ROOT}/test/e2e/config/cloudstack.yaml \
	    -e2e.skip-resource-cleanup=false -e2e.use-existing-cluster=true \
	&& echo "This step took $$(($$(date +%s)-d)) seconds"
	kind delete clusters capi-test
