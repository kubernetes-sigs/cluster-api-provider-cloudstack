## Debugging the CAPC Controller Outside of the Cluster

The CAPC controller can be run outside of the cluster, including in a debugger.  To do so one must:
- Have all the CAPI resources established in your bootstrap cluster except the CAPC controller
- The CAPC controller webhook certs exported to the local filesystem
- A modified CAPC webhook service that points to the externally running CAPC controller

Script prepForDebugging.sh sets that up for a CAPC bootstrap cluster.  It assumes:
- The following programs are available for use: kubectl, jq, envsubst, pcregrep (pcre)
- That the bootstrap cluster has been created and initialized (clusterctl init --infrastructure cloudstack)
- That your local workstation's IP address can be obtained from network device en0 (or another device that can be specified as an argument on the prepForDebugging.sh command line)
- That CAPC is the v1alpha3 version.  (Things change for v1alpha4)

How to:
- Establish your bootstrap kind cluster
- Deploy CAPC to bootstrap cluster: clusterctl init --infrastructure cloudstack
- cd to your CAPC project directory
- Make sure you've got a correct cloud-config file in your CAPC project directory
- run hack/debugging/prepForDebugging.sh
- Verify that the displayed k8s service endpoint is for your local workstation address.
- Run CAPC bin/manager in your debugger with parameters --cert-dir=/tmp/k8s-webhook-server/serving-certs --cloud-config-file=cloud-config
