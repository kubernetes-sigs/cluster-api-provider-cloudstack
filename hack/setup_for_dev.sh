#!/bin/sh

# Source this file for local dev.

export IMG=localhost:5000/cluster-api-provider-cloudstack:latest
export REPO_ROOT=`pwd`
export KUBEBUILDER_ASSETS=$REPO_ROOT/bin
export PATH=$REPO_ROOT/bin:$PATH
export ACK_GINKGO_DEPRECATIONS=1.16.4
export CLOUDSTACK_B64ENCODED_SECRET=$(base64 -i ./cloud-config)
