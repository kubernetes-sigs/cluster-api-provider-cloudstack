# Cluster API Provider for Cloudstack (CAPC) Release Notes

## Version 0.4.0

These Release Notes are for the customer downloading and deploying CAPC private Version 0.4.0 released on 03/11/2022.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

THE SOFTWARE SHOULD NOT BE USED FOR PRODUCTION WORKLOADS

### This release extends the v0.3.0 release of CAPC with:

  * v1beta1 API declared
  * Support for distributing cluster virtual machines across multiple CloudStack Zones/Networks
  * Enablement of the CAPC controller metrics port
  * Improved cleanup of CloudStack network components allocated as part of a cluster upon cluster deletion.
  * Accelerated cluster deletion through concurrent VM deletion
  * Node names match machine names
  * Support for disabling TLS certificate validation for CloudStack connections (cloud_config verify-ssl boolean parameter, true by default).
  * Example templates for implementing CAPI Machine Health Checking / Remediation of CAPC clusters

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

* cluster-api-provider-cloudstack-v0.4.0.tar.gz: container image of the CAPC controller
* shasum.txt containing checksum for the released cluster-api-provider-cloudstack-v0.4.0.tar.gz
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
