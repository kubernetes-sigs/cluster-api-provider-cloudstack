#!/bin/bash
set -eu

zone_name=
profile="default"

export AWS_DEFAULT_OUTPUT="json"

help() {
  echo "Continually queries Kubernetes for control plane machines and adds their IP address to an Amazon Route 53"
  echo "recordset.  The recordset name will be cp, and it will be created in the specified zone.  If a recordset"
  echo "already exists with that name, it will first be deleted."
  echo
  echo "The Route 53 zone has to already exist.  You can create one in the AWS console."
  echo
  echo "Before running this script, configure kubectl with the proper kubeconfig and namespace so it can get the"
  echo "cluster machines."
  echo
  echo "This script is not intended for production use."
  echo
  echo "USAGE: $0 -z <zone name> [-p <AWS profile name>]"
}

if [[ $# -eq 0 ]]
then
  help
  exit 2
fi

short_opts='z:p:h'
long_opts='zone:,profile:,help'
parsed_opts=$(getopt 'z:p:h' $*)
eval set -- $parsed_opts

while true
do
  case "$1" in
    -z)
      zone_name="$2"
      shift 2
      ;;
    -p)
      profile="$2"
      shift 2
      ;;
    -h)
      shift
      help
      exit 0
      ;;
    --)
      shift
      break
      ;;
    *)
      echo "Impossible value found.  This is a bug."
      exit 1
      ;;
  esac
done

if [[ -z $zone_name ]]
then
  echo "Missing zone name"
  exit 1
fi

# Zone name must end with a period, but the user doesn't need to know that.  Add one if it's missing.
if [[ ! $zone_name =~ [.]$ ]]
then
  zone_name=$zone_name.
fi

recordset_name="cp.$zone_name"

echo "Getting the zone ID from AWS"
zone_id=$(aws route53 list-hosted-zones --profile "$profile" | jq -r '.HostedZones[] | select(.Name == "'"$zone_name"'").Id | split("/")[2]')
if [[ -n $zone_id ]]
then
  echo "Found zone $zone_name"
else
  echo "Zone $zone_name not found.  Please create it first."
  exit 1
fi

get_recordset() {
  aws route53 list-resource-record-sets --profile "$profile" --hosted-zone-id "$zone_id" | jq -r '.ResourceRecordSets[] | select(.Name == "'"$recordset_name"'")'
}

upsert_addresses() {
  local addresses=$1
  echo "Replacing old records"
  local recordset='{"Name":"'"$recordset_name"'","Type":"A","TTL":10,"ResourceRecords":[]}'
  for address in $addresses
  do
    echo "Adding $address"
    recordset=$(echo "$recordset" | jq -r --arg a "$address" '.ResourceRecords += [{"Value":$a}]')
  done
  local batch=$(jq -r -n --argjson rs "$recordset" '{"Changes":[{"Action":"UPSERT","ResourceRecordSet":$rs}]}')
  aws route53 change-resource-record-sets --profile "$profile" --hosted-zone-id "$zone_id" --change-batch "$batch" > /dev/null
}

# If the recordset exists from a previous run, delete it.
old_recordset=$(get_recordset)
if [[ -n $old_recordset ]]
then
  echo "Deleting recordset $recordset_name"
  aws route53 change-resource-record-sets --profile "$profile" --hosted-zone-id "$zone_id" --change-batch '{"Changes":[{"Action":"DELETE","ResourceRecordSet":'"$old_recordset"'}]}' > /dev/null
fi

echo "Watching for control plane machines..."
old_addresses=
while true
do
  addresses=$(kubectl get machines -A -o json | jq -r '.items[] | select(.metadata.labels."cluster.x-k8s.io/control-plane" != null) | .status | select(.addresses!=null) | .addresses[].address')
  if [[ $addresses != "$old_addresses" ]]
  then
    upsert_addresses "$addresses"
  fi
  old_addresses=$addresses
  sleep 5
done
