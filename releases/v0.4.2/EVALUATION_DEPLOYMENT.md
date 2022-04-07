# CloudStack Cluster API Provider (CAPC) Release v.0.4.2 Evaluation Deployment Guide

This document defines a manual deployment process suitable for evaluating this CAPC release.

## Evaluation Environment Pre-Requisites:

### - A running Kubernetes cluster for hosting CAPC

This should be an easily disposable/re-creatable cluster, such as a locally-running kind (Kuberetes in Docker) cluster.

Your KUBECONFIG file's *current-context* must be set to the cluster you want to use.

### - CAPI clusterctl v1.0.1 (https://github.com/kubernetes-sigs/cluster-api/releases/tag/v1.0.1)

This process has been tested with this version of clusterctl.  Subsequent 1.0.x versions should work as well.

### - A CloudStack Environment with the following resources defined
- Zone
- Network
- CAPI-compatible QEMU template (i.e., created with https://github.com/kubernetes-sigs/image-builder)
- Machine Offerings (suitable for running Kubernetes nodes)
- apikey and secretkey for a CloudStack user having domain administrative privileges
- Available ACS IP Address for the k8s Control Plane endpoint (Shared network: available IP address in the network range; isolated network: public IP address)

## Deployment Steps
### Define Identity Environment Variable

An environment variable named CLOUDSTACK_B64ENCODED_SECRET must be defined, containing the base64 encoding of a 
cloud-config properties file.  This file is of the form:

```
[Global]
api-url = <urlOfCloudStackAPI>
api-key = <cloudstackUserApiKey>
secret-key = <cloudstackUserSecretKey>
```
After defining this in a file named cloud-config, create the environment variable with:

```
export CLOUDSTACK_B64ENCODED_SECRET=$(base64 -w0 -i cloud-config 2>/dev/null || base64 -b 0 -i cloud-config)
```

For security, delete this cloud-config file after creating this environment variable.

### Deploy the supplied container image archive (.tar.gz) to a suitable image registry.  

*We use https://github.com/kubernetes-sigs/cluster-api/blob/main/hack/kind-install-for-capd.sh to launch a local
docker registry integrated into a kind cluster for lightweight development and testing.*

- On a computer with docker, load the provided cluster-api-provider-capc.tar.gz to docker: 
```
docker load --input cluster-api-provider-capc_v0.4.2.tar.gz
```

This will create image *localhost:5000/cluster-api-provider-cloudstack:v0.4.2* in your local docker.  This is suitable
for pushing to a local registry.

- (Optional) Tag this image for your registry.
```
docker tag localhost:5000/cluster-api-provider-cloudstack:v0.4.2 <yourRepoFqdn>/cluster-api-provider-cloudstack:v0.4.2
```

Push it to your registry (localhost:5000 if using local registry)
```
docker push <yourRepoFqdn>/cluster-api-provider-cloudstack:v0.4.2
```

### Create clusterctl configuration files
A cluster-api.zip file has been provided, containing the files and directory structure suitable for configuring 
clusterctl to work with this interim release of CAPC.  It should be restored under $HOME/.cluster-api.  It contains:

```
Archive:  /Users/jweite/Dev/cluster-api-cloudstack-v0.4.2-assets/cluster-api.zip
* clusterctl.yaml
* dev-repository/
* dev-repository/infrastructure-cloudstack/
* dev-repository/infrastructure-cloudstack/v0.4.2/
* dev-repository/infrastructure-cloudstack/v0.4.2/cluster-template.yaml
* dev-repository/infrastructure-cloudstack/v0.4.2/cluster-template-managed-ssh.yaml
* dev-repository/infrastructure-cloudstack/v0.4.2/cluster-template-ssh-material.yaml
* dev-repository/infrastructure-cloudstack/v0.4.2/infrastructure-components.yaml
* dev-repository/infrastructure-cloudstack/v0.4.2/metadata.yaml
```

*Note: If you already have a $HOME/.cluster-api we strongly suggest you delete or stash it.*

```
cd ~
mkdir .cluster-api
cd .cluster-api
unzip cluster-api.zip 
```

### Edit the clusterctl configuration files
- **clusterctl.yaml:** in the *url* attribute replace \<USERID\> with your OS user id to form a valid absolute path to infrastructure-components.yaml.

- **dev-repository/infrastructure-cloudstack/v0.4.2/infrastructure-components.yaml:** if you're not using a local registry modify the capc-controller-manager deployment, changing the spec.template.spec.containers[0].image (line 617) to correctly reflect your container registry. 

### Deploy CAPI and CAPC to your bootstrap Kubernetes cluster
```
clusterctl init --infrastructure cloudstack
```

### Generate a manifest for the CAPI custom resources needed to allocate a workload cluster.

*Set the below environment variables as appropriate for your CloudStack environment.*

```
CLOUDSTACK_ZONE_NAME=<MyZoneName> \
CLOUDSTACK_NETWORK_NAME=<MyNetworkName> \
CLOUDSTACK_TEMPLATE_NAME=<MyTemplateName> \
CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING=<MyServiceOfferingName> \
CONTROL_PLANE_MACHINE_COUNT=1 \
CLOUDSTACK_WORKER_MACHINE_OFFERING=<MyServiceOfferingName> \
WORKER_MACHINE_COUNT=1 \
CLUSTER_ENDPOINT_IP=<AvailableSharedOrPublicIP> \
CLUSTER_ENDPOINT_PORT=6443 \
KUBERNETES_VERSION=<KubernetesVersionOnTheImage> \
CLUSTER_NAME=<MyClusterName> \
clusterctl generate cluster $CLUSTER_NAME --from ~/.cluster-api/dev-repository/infrastructure-cloudstack/v0.4.2/cluster-template.yaml > clusterTemplate.yaml
```

### Review the generated clusterTemplate.yaml and adjust as necessary


### Provision your workload cluster

```
kubectl apply -f clusterTemplate.yaml
```

Provisioning can take several minutes to complete.  You will see a control plane VM created in CloudStack pretty quickly, 
but it takes a while for it to complete its cloud-init to install Kubernetes and become a functioning control plane.  
Allocation of the worker node(s) (with *md* in their VM names) won't occur until the control plane is operational.

You can monitor the CAPC controller as it conducts the provisioning process with:
```
# Get the full name of the CAPC controller pod
kubectl -n capc-system get pods

# Tail its logs
kubectl -n capc-system log -f <CAPCcontrollerPodFullName>
```

### Fetch a kubeconfig to access your cluster
```
clusterctl get kubeconfig <clusterName> > <clusterName>_kubeconfig
```

You can then either export a KUBECONFIG environment variable pointing to this file, or use kubectl's --kubeconfig=<filePath>
flag.
```
export KUBECONFIG=<clusterName>_kubeconfig
```

### Examine the provisioned Kubernetes Cluster's nodes
```
kubectl get nodes
```
Expect to see a control plane and a worker node reported by Kubernetes.  Neither will report that they are ready
because no CNI is installed yet.

### Install Cilium CNI
```
cilium install
```
The above command presumes that the cilium installer is present on the local workstation.

It will take a minute while it waits for cilium to become active.

### Confirm that Cluster is Ready for Work
```
kubectl get nodes
```
Expect now to see both nodes list as ready.

### Conclusion
At this point the workload cluster is ready to accept workloads.  Use it in the usual way via the kubeconfig generated
earlier

### Cluster Deletion
As mentioned in the preface, CAPC is not yet able to delete workload cluster.  To do so manually we recommend
simply tearing-down the kind bootstrap cluster, and then manually deleting the CloudStack VMs created for it
using the CloudStack UI, API or similar facilities.
