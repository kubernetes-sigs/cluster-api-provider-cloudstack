# Cluster API Provider for Cloudstack (CAPC) Release Notes

## Version 0.1.0

These Release Notes are for the customer downloading and deploying CAPC private Version 0.1.0  released on 12/3/2021.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

THE SOFTWARE SHOULD NOT BE USED FOR PRODUCTION WORKLOADS

* This software creates virtual machine instances in a target CloudStack environment and configures them into a Kubernetes cluster.
* This is a release of partial functionality for mid-development-process review by stakeholders.  Only cluster creation is supported.
    * Cluster creation with CAPI clusterctl _v0.4.4_ is supported.  EKS-A integration is not part of this release.
    * With CloudStack Shared Networks only *single* Control Plane clusters are supported.   High Availability with multiple control planes is not supported on CloudStack Shared Networks.  High Availability *is* supported on CloudStack Isolated Networks using CloudStack native Load Balancers over CloudStack Public IPs.
    * Baseline Components
        * Apache CloudStack 4.14
        * Cluster API clusterctl v0.4.4
        * Guest OS - RHEL 8
        * Host OS - CentOS 7
        * Cilium Container Network
    * Note: The default mode of operation for the deployed Kubernetes cluster components is to use self-signed certificates.  Options exist for use of an enterprise certificate authority via cert-manager (https://cert-manager.io/docs/configuration/).  Detailed configuration of this component is outside the scope of this release.
* The following pre-conditions must be met for CreateCluster feature to operate as designed.
    * A functional CloudStack 4.14 deployment
    * The CloudStack account used by CAPC must have domain administrator privileges or be otherwise appropriately privileged to execute the API calls specified in the below CAPC CloudStack API Calls document link.
    * Network and Zone must be pre-created and available to CAPC prior to CreateCluster API call.
    * A suitable template for the image must be available in CloudStack.
        * The software has been tested with RHEL-8 images created with CAPI Image-builder.
    * Machine offerings suitable for running Kubernetes nodes must be available in CloudStack
    * When using CloudStack Shared Networks, an unused IP address in the shared network’s address range must be available for the Kubernetes Control Plane for each cluster, upon which it will be exposed.
* Following enhancements to CreateCluster functionality are added to the roadmap and will be targeted in the future release of CAPC.
    * Additional custom environment specific items based on feedback from the pilot application.
    * Support for specifying *extra Subject Alternate Names* for generated API Server certificates.
    * Support for CloudStack-supported Hypervisors beyond KVM/QEMU
* This software has been evaluated by the Dependency Check (https://owasp.org/www-project-dependency-check/)  golang analyzer (https://jeremylong.github.io/DependencyCheck/analyzers/golang-mod.html).  Findings have been catalogued via security_findings.csv.

### Documentation :

CAPC CloudStack API Calls


### Release Assets :

* cluster-api-provider-cloudstack.tar.gz: container image of the CAPC controller
* cluster-api.zip: configuration files for clusterctl
    * infrastructure-components.yaml
    * metadata.yaml
    * cluster-template.yaml
    * cluster-template-ssh.yaml
* EVALUATION_DEPLOYMENT.md: instructions for manual deployment of this interim release for evaluation

## Version 0.2.0

These Release Notes are for the customer downloading and deploying CAPC private Version 0.2.0  released on 12/24/2021. Baseline components and pre-conditions from v0.1.0 also apply to this release.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

THE SOFTWARE SHOULD NOT BE USED FOR PRODUCTION WORKLOADS.

### **New Features:**

* v1alpha3 APIs implemented, for compatibility with EKS Anywhere
* Delete Cluster API is implemented: VMs will be deleted completely.
* CAPI Move operation supported
* EKS-A integration of CAPC enables orchestration of Kubernetes Cluster Provisioning on CloudStack similar to existing VSphere implementation
    * Cluster creation and deletion operations supported at this time

### Release Assets:

* cluster-api-provider-cloudstack-v0.2.0.tar.gz: container image of the CAPC controller
* cluster-api.zip: configuration files for clusterctl
    * infrastructure-components.yaml
    * metadata.yaml
    * cluster-template.yaml
    * cluster-template-managed-ssh.yaml
    * cluster-template-ssh-material.yaml
    * EVALUATION_DEPLOYMENT.md: instructions for manual deployment of this interim release for evaluation

###  Known Issues

* VM Health Checking not implemented.
* Deletion of CAPC Clusters with missing VMs can hang.
* Deletion of CAPC Clusters by deleting *all* its CAPC CRs fails.  Delete CAPC clusters deleting the Cluster CR only.
* CloudStackMachine CR may report ready for a VM that fails to get to the *running* state*.*

###  Future Scope/Features

* Cluster Upgrade.
* AntiAffinity groups.
* Support for multiple control plane nodes in an externally load balanced environment on a shared network.

