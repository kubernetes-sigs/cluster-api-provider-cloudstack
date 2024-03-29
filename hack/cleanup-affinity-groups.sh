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
# chmod +x cleanup-affinity-groups.sh
# ./cleanup-affinity-groups.sh -h

set -o errexit
set -o nounset
set -o pipefail

# script params
DRY_RUN=false
VERBOSE=false

# k8s params
NAMESPACE=default
KUBECONFIG=$HOME/.kube/config

# cmk params
CS_URL=
CS_APIKEY=
CS_SECRETKEY=

debug() {
  if [[ "$VERBOSE" == "true" ]]; then
    echo -e "[debug] $@"
  fi
}

_kubectl() {
  KUBECONFIG=$KUBECONFIG kubectl -n $NAMESPACE -o json $@
}

_cmk() {
  cmk -u $CS_URL -k $CS_APIKEY -s $CS_SECRETKEY -o json $@
}

get_affinity_groups() {
  _kubectl get cloudstackaffinitygroups | jq -r '.items[].metadata.name'
}

get_cluster() {
  local affinitygroup=$1
  _kubectl get cloudstackaffinitygroup $affinitygroup | jq -r '.metadata.labels."cluster.x-k8s.io/cluster-name"'
}

get_cluster_credentials() {
  local cluster=$1
  _kubectl get cloudstackcluster $cluster | jq -r '.spec.failureDomains[].acsEndpoint.name' | uniq
}

setup_acs_credentials() {
  local credential=$1
  CS_URL=$(_kubectl get secret $credential | jq -r '.data."api-url"' | base64 -D)
  CS_APIKEY=$(_kubectl get secret $credential | jq -r '.data."api-key"' | base64 -D)
  CS_SECRETKEY=$(_kubectl get secret $credential | jq -r '.data."secret-key"' | base64 -D)
  debug "Using CloudStack Control Plane URL: $CS_URL and CloudStack Account: $(_cmk list users | jq -r '.user[] | .account + " and User: " + .username')"
}

main() {
  local ags=$(get_affinity_groups)
  debug "Affinity groups in the namespace $NAMESPACE:\n$ags"
  for ag in $ags; do
    echo -e "\033[0;32m[info]\033[0m Checking CloudStack Affinity Group: $ag"
    local cluster=$(get_cluster $ag)
    for credential in $(get_cluster_credentials $cluster); do
      setup_acs_credentials $credential
      local ag_uuid=$(_kubectl get cloudstackaffinitygroup $ag | jq -r '.spec.id')
      local ag_instances=$(_cmk list affinitygroups id=$ag_uuid | jq -r '.affinitygroup[0].virtualmachineIds')
      if [[ "$ag_instances" == "null" ]]; then
        echo -e "\033[0;35m[info]\033[0m Found Affinity Group ($ag_uuid) with no instances assigned: $ag"
        if [[ "$DRY_RUN" == "false" ]]; then
          kubectl -n $NAMESPACE delete cloudstackaffinitygroup $ag
          echo -e "\033[0;31m[info]\033[0m Affinity Group ($ag_uuid) $ag has been removed"
        else
          echo -e "\033[0;35m[info]\033[0m [dryrun] Affinity Group ($ag_uuid) $ag has been removed"
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
  echo "-n     Kubernetes namespace, default: default"
  echo "-h     Print this help"
  echo "-v     Verbose mode"
  echo
}

while getopts ":dk:vn:h" option; do
   case $option in
      d)
         DRY_RUN=true;;
      k)
         KUBECONFIG=$OPTARG;;
      v)
         VERBOSE=true;;
      n)
         NAMESPACE=$OPTARG;;
      h)
         help
         exit;;
     \?)
         echo "Error: Invalid option provided, please see help docs"
         help
         exit;;
   esac
done

if ! command -v jq &> /dev/null
then
    echo "[error] jq could not be found, please install first"
    exit 1
fi

if ! command -v kubectl &> /dev/null
then
    echo "[error] kubectl could not be found, please install first"
    exit 1
fi

if ! command -v cmk &> /dev/null
then
    echo "[error] cmk could not be found, please install https://github.com/apache/cloudstack-cloudmonkey/releases/tag/6.4.0-rc1 or newer"
    exit 1
fi

debug "[options] DRY_RUN=$DRY_RUN"
debug "[options] VERBOSE=$VERBOSE"
debug "[options] NAMESPACE=$NAMESPACE"
debug "[options] KUBECONFIG=$KUBECONFIG"
main
