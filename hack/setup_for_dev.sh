#!/bin/sh

# Source this file for local dev.

export IMG=localhost:5000/cluster-api-provider-cloudstack:latest
export PROJECT_DIR=`pwd`
export KUBEBUILDER_ASSETS=$PROJECT_DIR/bin
export PATH=$PROJECT_DIR/bin:$PATH
export ACK_GINKGO_DEPRECATIONS=1.16.4
export CLOUDSTACK_B64ENCODED_SECRET=$(base64 -i ./cloud-config)
