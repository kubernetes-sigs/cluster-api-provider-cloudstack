# Cluster API Provider for Cloudstack (CAPC) Release Notes

## Version v0.4.3

These Release Notes are for the customer downloading and deploying CAPC private Version 0.4.3 released on 05/9/2022.

### This release extends the v0.4.2 release of CAPC with:

  * ACS Disk Offering support
  * Increase control plane VM memory request to 50/100MiB.
  * Consider VMs in *expunging* state as deleted, eliminating long expunge delays from the deprovisioning critical path, most signficantly for failovers.
  * Faster detection of stopped VMs and VMs in error state than provided by CAPI MHC
  * Operations now performed as actual domain/account user specified, instead of as the overarching domain/account of the user running CAPC.
  * Bug-fix: dynamic affinity groups now work with sub-domains.
  * Various improvements to e2e tests 

### TLS Certificates
The default mode of operation for the deployed Kubernetes cluster components is to use self-signed certificates.  Options exist for use of an enterprise certificate authority via cert-manager (https://cert-manager.io/docs/configuration/).  Detailed configuration of this component is outside the scope of this release.

### Pre-conditions

* The following pre-conditions must be met for CAPC to operate as designed.
    * A functional CloudStack 4.14 or 4.16 deployment
    * The CloudStack account used by CAPC must have domain administrator privileges or be otherwise appropriately privileged to execute the API calls specified in the below CAPC CloudStack API Calls document link.
    * Zone(s) and Network(s) must be pre-created and available to CAPC prior to CreateCluster API call.
    * A VM template suitable for implementing a Kubernetes node with kubeadm must be available in CloudStack.
        * The software has been tested with RHEL-8 images created with CAPI Image-builder.
    * Machine offerings suitable for running Kubernetes nodes must be available in CloudStack
    * When using CloudStack Shared Networks, an unused IP address in the shared network’s address range must be available for the Kubernetes Control Plane for each cluster, upon which it will be exposed.

### Release Assets :

* cluster-api-provider-cloudstack-v0.4.3.tar.gz: container image of the CAPC controller
* shasum.txt containing checksum for the released cluster-api-provider-cloudstack-v0.4.3.tar.gz
* cluster-api.zip: configuration files for clusterctl
    * infrastructure-components.yaml
    * metadata.yaml
    * cluster-template.yaml
    * cluster-template-ssh.yaml
* EVALUATION_DEPLOYMENT.md: instructions for manual deployment of this interim release for evaluation via clusterctl.
* security_findings.csv: results of package security scan


### Known Issues :

* Cluster upgrade is not supported when the controlPlaneEndpoint is defined to be an IP address in a shared network.

###  Future Scope/Features

* Accelerated remediation of VM state drift
