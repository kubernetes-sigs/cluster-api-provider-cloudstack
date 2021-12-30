#!/bin/bash

# Tested on Mac only!
# Pre-reqs: kubectl, jq, envsubst, pcregrep (pcre)

if [ ! -d "./hack/" ]; then
  echo "Run this from your CAPC project directory"
  exit 1
fi

MY_NETWORK_DEVICE="en0"
if [ -n "$1" ]; then
  MY_NETWORK_DEVICE=$1
fi

export MY_IP=$(ifconfig $MY_NETWORK_DEVICE | pcregrep -o1 'inet (\d+\.\d+\.\d+\.\d+) netmask')
if [ -z "$MY_IP" ]; then
  echo "Cannot determine your IP address from ifconfig of device $MY_NETWORK_DEVICE"
  exit 1
fi
echo "Using workstation IP $MY_IP (network device $MY_NETWORK_DEVICE)"

echo "Terminating capc controller deployment"
kubectl delete deployment capc-controller-manager -n capc-system

CERT_DIR="/tmp/k8s-webhook-server/serving-certs"
echo "Exporting certs to $CERT_DIR"
mkdir -p /tmp/k8s-webhook-server/serving-certs
kubectl -n capc-system get secret webhook-server-cert -o jsonpath='{.data}' | jq -r '."ca.crt"'  | base64 -d > "$CERT_DIR/ca.crt"
kubectl -n capc-system get secret webhook-server-cert -o jsonpath='{.data}' | jq -r '."tls.crt"' | base64 -d > "$CERT_DIR/tls.crt"
kubectl -n capc-system get secret webhook-server-cert -o jsonpath='{.data}' | jq -r '."tls.key"' | base64 -d > "$CERT_DIR/tls.key"
ls -l $CERT_DIR
cat $CERT_DIR/ca.crt
cat $CERT_DIR/tls.crt

echo ""
echo "Deleting current webhook service"
kubectl -n capc-system delete service capc-webhook-service

echo "Creating new webhook service pointed to local machine IP addr"
# Inject MY_IP into DebugWebhookServiceTemplate.yaml
envsubst < ./hack/debugging/debugWebhookServiceTemplate.yaml > /tmp/debugWebhookService.yaml
kubectl -n capc-system apply -f /tmp/debugWebhookService.yaml

echo ""
echo "***New service***"
kubectl -n capc-system describe service capc-webhook-service
echo ""
echo "***Endpoint***"
kubectl -n capc-system describe endpoints

echo "debug manager with --cert-dir=$CERT_DIR --cloud-config-file=cloud-config"
