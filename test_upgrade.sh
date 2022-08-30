#!/bin/bash

set -e # Exit on failed commands.
# set -x # Print each command as ran.
set -u # Fail on undeclared or unset variable use.
set -o pipefail # Fail if any command in pipe chain fails.

# Get the next dot release version based on current git tags and repo status.
export RELEASE_VERSION=$(./hack/release_ver.sh)
export RELEASE_DIR=infrastructure-cloudstack/$RELEASE_VERSION
(envsubst < ./hack/clusterctl-setup/clusterctl-config-template.yaml) > clusterctl-config.yaml
cat clusterctl-config.yaml
make release-manifests 
#clusterctl generate provider --infrastructure cloudstack:$RELEASE_VERSION --config clusterctl-config.yaml




#  git tag
make kind-cluster
export CLOUDSTACK_B64ENCODED_SECRET=`base64 -i cloud-config`
clusterctl init --infrastructure cloudstack:v0.4.4

# Wait for CAPC manager to be ready.
while [[ $(kubectl -n capc-system get pods -o json | jq '.items[].status.containerStatuses[].ready' | sort | uniq) != 'true' ]]; do
    echo waiting for machines to go ready
    sleep 1
done

kubectl apply -f ./acs_v1beta1.yaml

# Wait for all machines to be ready.
while [[ $(kubectl get cloudstackmachine -o json | jq '.items[].status.ready' | uniq) != 'true' ]]; do
    echo waiting for machines to go ready
    sleep 1
done

# finally, upgrade capc.
clusterctl upgrade apply --infrastructure capc-system/cloudstack:$RELEASE_VERSION --config clusterctl-config.yaml
