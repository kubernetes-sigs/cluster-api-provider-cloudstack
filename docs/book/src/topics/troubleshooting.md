# Troubleshooting

This guide (based on kind but others should be similar) explains general info on how to debug issues if a cluster creation fails.

## Get logs of Cluster API controller containers

```bash
kubectl -n capc-system logs -l control-plane=capc-controller-manager -c manager
```

Similarly, the logs of the other controllers in the namespaces `capi-system` and `cabpk-system` can be retrieved.

## Authenticaton Error

This is caused when the API Key and / or the Signature is invalid.
Please check them in the Accounts > User > API Key section of the UI or via the getUserKeys API
```
E0325 04:30:51.030540       1 controller.go:317] controller/cloudstackcluster "msg"="Reconciler error" "error"="CloudStack API error 401 (CSExceptionErrorCode: 0): unable to verify user credentials and/or request signature" "name"="kvm-capi" "namespace"="default" "reconciler group"="infrastructure.cluster.x-k8s.io" "reconciler kind"="CloudStackCluster"

```

## Cluster reconciliation failed with error: No match found for xxxx: {Count:0 yyyy:[]}

This is caused when resource 'yyyy' with the name 'xxxx' does not exist on the CloudStack instance
```
E0325 04:12:44.047381       1 controller.go:317] controller/cloudstackcluster "msg"="Reconciler error" "error"="No match found for zone-1: \u0026{Count:0 Zones:[]}" "name"="kvm-capi" "namespace"="default" "reconciler group"="infrastructure.cluster.x-k8s.io" "reconciler kind"="CloudStackCluster"
```
In such a case, check the spelling of the resource name or create it in CloudStack. Following which, update it in the workload cluster yaml
(in cases where the resource name is not immutable) or delete the capi resource and re-create it with the updated name

