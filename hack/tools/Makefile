# Copyright 2021 The Kubernetes Authors.
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

UNAME := $(shell uname -s)

# Directories.
BIN_DIR := bin
SHARE_DIR := share

OS := $(shell go env GOOS)
RUST_TARGET := unknown-$(OS)-gnu

ifeq ($(OS), darwin)
	RUST_TARGET := apple-darwin
	GH_ARCH_SUFFIX := macOS_amd64
	GTAR := gtar
endif

ifeq ($(OS), linux)
	GH_ARCH_SUFFIX := linux_amd64
	GTAR := tar
endif

MDBOOK_EXTRACT_COMMAND := tar xfvz $(SHARE_DIR)/mdbook.tar.gz -C bin
MDBOOK_ARCHIVE_EXT := .tar.gz

ifeq ($(OS), windows)
	RUST_TARGET := pc-windows-msvc
	MDBOOK_ARCHIVE_EXT := .zip
	MDBOOK_EXTRACT_COMMAND := unzip -d /tmp
endif

GOLANGCI_LINT_VERSION := v1.60.2
GOLANGCI_LINT_BIN := golangci-lint
GOLANGCI_LINT := $(abspath $(BIN_DIR)/$(GOLANGCI_LINT_BIN))
GOLANGCI_LINT_PKG := github.com/golangci/golangci-lint/cmd/golangci-lint

## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(BIN_DIR):
	mkdir -p $@

$(SHARE_DIR):
	mkdir -p $@

.PHONY: $(GTAR)
$(GTAR):
	@$(GTAR) --version > /dev/null || (echo Install GNU Tar with brew install gnu-tar && exit -1)


CONTROLLER_GEN := $(BIN_DIR)/controller-gen
$(CONTROLLER_GEN): $(BIN_DIR) go.mod go.sum # Build controller-gen from tools folder.
	go build -tags=tools -o $@ sigs.k8s.io/controller-tools/cmd/controller-gen

CONVERSION_GEN := $(BIN_DIR)/conversion-gen
$(CONVERSION_GEN): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $@ k8s.io/code-generator/cmd/conversion-gen

# A target for compatibility with the root Makefile.
$(BIN_DIR)/$(GOLANGCI_LINT_BIN): $(GOLANGCI_LINT) 

$(GOLANGCI_LINT_BIN): $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(BIN_DIR) # Install golangci-lint in tools folder.
	GOBIN=$(abspath $(BIN_DIR)) go install $(GOLANGCI_LINT_PKG)@$(GOLANGCI_LINT_VERSION)

STATIC_CHECK := $(BIN_DIR)/staticcheck
$(STATIC_CHECK): $(BIN_DIR) go.mod go.sum # Build staticcheck from tools folder.
	go build -tags=tools -o $@ honnef.co/go/tools/cmd/staticcheck

GINKGO := $(BIN_DIR)/ginkgo
$(GINKGO): $(BIN_DIR) go.mod go.sum # Build staticcheck from tools folder.
	go build -tags=tools -o $@ github.com/onsi/ginkgo/v2/ginkgo

ENVSUBST := $(BIN_DIR)/envsubst
$(ENVSUBST): $(BIN_DIR) go.mod go.sum # Build envsubst from tools folder.
	go build -tags=tools -o $@ github.com/a8m/envsubst/cmd/envsubst

GH_SHARE := $(SHARE_DIR)/gh

$(GH_SHARE): $(SHARE_DIR)
	mkdir -p $@

$(GH_SHARE)/gh.tar.gz: $(GH_SHARE)
	curl -L "https://github.com/cli/cli/releases/download/v$(GH_VERSION)/gh_$(GH_VERSION)_$(GH_ARCH_SUFFIX).tar.gz" -o $@

GH := $(BIN_DIR)/gh
GH: $(GTAR) $(GH_SHARE)/gh.tar.gz
	$(GTAR) -xvf share/gh/gh.tar.gz gh_$(GH_VERSION)_$(GH_ARCH_SUFFIX)/bin/gh --strip-components 1 --directory $(BIN_DIR)
	chmod +x $@
	touch -m $@

KIND := $(BIN_DIR)/kind
$(KIND): $(BIN_DIR) go.mod go.sum
	go build -tags tools -o $@ sigs.k8s.io/kind

KUSTOMIZE := $(BIN_DIR)/kustomize
$(KUSTOMIZE): $(BIN_DIR) go.mod go.sum # Build kustomize from tools folder.
	CGO_ENABLED=0 go build -tags=tools -o $@ sigs.k8s.io/kustomize/kustomize/v5

MDBOOK_SHARE := $(SHARE_DIR)/mdbook$(MDBOOK_ARCHIVE_EXT)
$(MDBOOK_SHARE): ../../versions.mk $(SHARE_DIR)
	curl -sL -o $(MDBOOK_SHARE) "https://github.com/rust-lang/mdBook/releases/download/$(MDBOOK_VERSION)/mdBook-$(MDBOOK_VERSION)-x86_64-$(RUST_TARGET)$(MDBOOK_ARCHIVE_EXT)"

MDBOOK := $(BIN_DIR)/mdbook
$(MDBOOK): $(BIN_DIR) $(MDBOOK_SHARE)
	$(MDBOOK_EXTRACT_COMMAND)
	chmod +x $@
	touch -m $@

MDBOOK_EMBED := $(BIN_DIR)/mdbook-embed
$(MDBOOK_EMBED): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $(BIN_DIR)/mdbook-embed sigs.k8s.io/cluster-api/hack/tools/mdbook/embed

MDBOOK_RELEASELINK := $(BIN_DIR)/mdbook-releaselink
$(MDBOOK_RELEASELINK): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $(BIN_DIR)/mdbook-releaselink sigs.k8s.io/cluster-api/hack/tools/mdbook/releaselink

MDBOOK_TABULATE := $(BIN_DIR)/mdbook-tabulate
$(MDBOOK_TABULATE): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $(BIN_DIR)/mdbook-tabulate sigs.k8s.io/cluster-api/hack/tools/mdbook/tabulate

MOCKGEN := $(BIN_DIR)/mockgen
$(MOCKGEN): $(BIN_DIR) go.mod go.sum # Build mockgen from tools folder.
	go build -tags=tools -o $@ go.uber.org/mock/mockgen

RELEASE_NOTES := $(BIN_DIR)/release-notes
$(RELEASE_NOTES): $(BIN_DIR) go.mod go.sum
	go build -tags tools -o $@ sigs.k8s.io/cluster-api/hack/tools/release

PLANTUML := $(BIN_DIR)/plantuml-sentinal
$(PLANTUML): plantuml.Dockerfile ../../versions.mk
	docker build --build-arg PLANTUML_VERSION=$(PLANTUML_VERSION)  . -f plantuml.Dockerfile -t "plantuml-builder"
	touch $@

.PHONY: clean
clean: ## Remove all tools
	rm -rf bin
	rm -rf share
