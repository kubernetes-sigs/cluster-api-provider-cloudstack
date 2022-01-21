# Cluster API Provider for Cloudstack (CAPC) Release Notes

## Version 0.3.0

These Release Notes are for the customer downloading and deploying CAPC private Version 0.3.0 released on 01/21/2022.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

THE SOFTWARE SHOULD NOT BE USED FOR PRODUCTION WORKLOADS

* This software creates virtual machine instances in a target CloudStack environment and configures them into a Kubernetes cluster.
* This is a release of partial functionality for mid-development-process review by stakeholders.
    * Cluster creation, deletion and upgrade with CAPC via EKS Anywhere.
    * Support for CloudStack Affinity Groups
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

### Release Assets :

* cluster-api-provider-cloudstack.tar.gz: container image of the CAPC controller
* cluster-api.zip: configuration files for clusterctl
    * infrastructure-components.yaml
    * metadata.yaml
    * cluster-template.yaml
    * cluster-template-ssh.yaml
* EVALUATION_DEPLOYMENT.md: instructions for manual deployment of this interim release for evaluation via clusterctl.

### Known Issues :

* When using IP address for controlPlaneEndpoint cluster upgrade is not supported.
* VM Health Checking not implemented.

###  Future Scope/Features

* Dynamic Affinity Group creation