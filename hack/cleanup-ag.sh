#!/usr/bin/env bash
#
# Copyright 2023 The Kubernetes Authors.
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
#
# Requires: jq, kubectl, cmk (get https://github.com/apache/cloudstack-cloudmonkey/releases/tag/6.4.0-rc1 or later)
#
# About: this tool helps to remove CloudStack affinity groups from CAPC
# management cluster which are not assigned to any instances.
#
# Usage and help:
# chmod +x cleanup-ag.sh
# ./cleanup-ag.sh -h

set -o errexit
set -o nounset
set -o pipefail

export DRY_RUN=false
export VERBOSE=false
export KUBECONFIG=$HOME/.kube/config

debug() {
  if [[ "$VERBOSE" == "true" ]]; then
    echo $@
  fi
}

get_affinity_groups() {
  kubectl get cloudstackaffinitygroups -o json -A | jq -r '.items[].metadata.name'
}

get_cluster() {
  affinityGroup=$1
  kubectl get cloudstackaffinitygroup $affinityGroup -o json | jq -r '.metadata.labels."cluster.x-k8s.io/cluster-name"'
}

get_cluster_credentials() {
  cluster=$1
  kubectl get cloudstackcluster $cluster -o json | jq -r '.spec.failureDomains[].acsEndpoint.name' | uniq
}

setup_acs_credentials() {
  credential=$1
  export CS_URL=$(kubectl get secret $credential -o json | jq -r '.data."api-url"' | base64 -D)
  export CS_APIKEY=$(kubectl get secret $credential -o json | jq -r '.data."api-key"' | base64 -D)
  export CS_SECRETKEY=$(kubectl get secret $credential -o json | jq -r '.data."secret-key"' | base64 -D)
  debug "Using CloudStack Control Plane URL: $CS_URL and CloudStack Account: $(run_cmk list users | jq -r '.user[] | .account + " and User: " + .username')"
}

run_cmk() {
  cmk -u $CS_URL -k $CS_APIKEY -s $CS_SECRETKEY -o json $@
}

main() {
  for ag in $(get_affinity_groups); do
    echo "Checking CloudStack Affinity Group: $ag"
    cluster=$(get_cluster $ag)
    for credential in $(get_cluster_credentials $cluster); do
      setup_acs_credentials $credential
      CS_AG_ID=$(kubectl get cloudstackaffinitygroup $ag -o json | jq -r '.spec.id')
      CS_AG_VMS=$(run_cmk list affinitygroups id=$CS_AG_ID | jq -r '.affinitygroup[0].virtualmachineIds')
      if [[ "$CS_AG_VMS" == "null" ]]; then
        echo "Found Affinity Group ($CS_AG_ID) with no instances assigned:" $ag
        if [[ "$DRY_RUN" == "false" ]]; then
          kubectl delete cloudstackaffinitygroup $ag
          echo "Affinity Group ($CS_AG_ID) $ag has been removed"
        else
          echo "[DRY RUN] Affinity Group ($CS_AG_ID) $ag has been removed"
        fi
      fi
    done
  done
}

help() {
  echo "Usage: $0 [-d|k|h|v]"
  echo
  echo "This cleanup tool helps to remove CloudStack affinity groups from CAPC"
  echo "management cluster which are not assigned to any instances, which may"
  echo "have been created as a side effect of other operations. This tool checks"
  echo "all the cloudstackaffinitygroups using its CloudStack cluster specific"
  echo "credential(s) and uses cmk to check if the affinity group have no"
  echo "instances assigned. In dry-run, it outputs such affinity groups"
  echo "otherwise it deletes them."
  echo
  echo "Options:"
  echo "-d     Runs the tools in dry-run mode"
  echo "-k     Pass custom kube config, default: \$HOME/.kube/config"
  echo "-h     Print this help"
  echo "-v     Verbose mode"
  echo
}

while getopts ":dkvh" option; do
   case $option in
      d)
         export DRY_RUN=true;;
      k)
         export KUBECONFIG=$OPTARG;;
      v)
         export VERBOSE=true;;
      h)
         help
         exit;;
     \?)
         echo "Error: Invalid option provided, please see help docs"
         help
         exit;;
   esac
done

main
