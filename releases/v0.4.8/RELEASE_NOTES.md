# Cluster API Provider for Cloudstack (CAPC) Release Notes

## Version v0.4.8

These Release Notes are for the customer downloading and deploying CAPC private Version 0.4.8 released on 10/20/2022.

### This release extends the v0.4.4 release of CAPC with:
  * Support for distributing VMs across multiple CloudStack management endpoints via Failure Domains (in addition to pre-existing Zone, Domain and Account-based Failure Domains)
  * v1beta2 declared, as the above is a breaking change.
  * Switch to user-provisioned-secret-based CloudStack credentials (from the previous env-var-based method).
  * Support for Customized Disk Offerings (i.e., with parameters)
  * Custom metrics that count CloudStack API errors returned, grouped by error code.
  * new *make* target and config files for creating an alternative infrastructure-components.yaml that exposes the manager metrics port from the pod via kube-rbac-proxy.
  * Discontinued MachineStateChecker, as the remediation technique of deleting CAPI machines from with the manager is proving unreliable.
  * Use of CAPI Machine name as hostname and k8s node name
  * Various bug fixes, doc improvements and build/test enhancements.

### TLS Certificates
The default mode of operation for the deployed Kubernetes cluster components is to use self-signed certificates.  Options exist for use of an enterprise certificate authority via cert-manager (https://cert-manager.io/docs/configuration/).  Detailed configuration of this component is outside the scope of this release.

### Pre-conditions

* The following pre-conditions must be met for CAPC to operate as designed.
    * A functional CloudStack 4.14 or 4.16 deployment
    * The CloudStack account used by CAPC must have domain administrator privileges or be otherwise appropriately privileged to execute the API calls specified in the below CAPC CloudStack API Calls document link.  A least-privilege CloudStack Role is now documents in the CAPC book. 
    * Zone(s) and Network(s) must be pre-created and available to CAPC prior to CreateCluster API call.
    * A VM template suitable for implementing a Kubernetes node with kubeadm must be available in CloudStack.
        * The software has been tested with RHEL-8 images created with CAPI Image-builder.
        * Links to pre-built images are available in the CAPC Book.
    * Machine offerings suitable for running Kubernetes nodes must be available in CloudStack
    * When using CloudStack Shared Networks, an unused IP address in the shared network’s address range must be available for the Kubernetes Control Plane for each cluster, upon which it will be exposed.

### Release Assets :
* CAPI Standard deployment manifests: infrastructure-components.yaml, metadata.yaml, cluster-template.yaml and its flavor variations.
* capi-cloudstack-controller image, at gcr.io/k8s-staging-capi-cloudstack
* security_findings.csv: results of package security scan

### Known Issues :
* Cluster upgrade is not supported when the controlPlaneEndpoint is defined to be an IP address in a shared network when not using kube-vip for the control plane. 
