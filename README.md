## CloudStack Cluster API Provider (CAPC)

A Kubernetes Cluster API Provider implementation for Apache CloudStack, suitable for integration and use by AWS EKS-Anywhere.

## Security

See [CONTRIBUTING](CONTRIBUTING.md#security-issue-notifications) for more information.

## License

This project is licensed under the Apache-2.0 License.

## Testing

To run a particular test. In this case TestCreateInstance2.

Integration tests use Controller Runtime EnvTest.  Your testing environment must be pre-configured with several EnvTest
git dependencies.
See [configuring envtest for integration tests](https://book.kubebuilder.io/reference/envtest.html?highlight=etcd#configuring-envtest-for-integration-tests)

- export PROJECT_DIR=`pwd`

- copy cloud-config to project dir.

- `$ go test -v -run TestCreateInstance2 ./pkg/cloud`

## Dev w/Tilt

Install [tilt prerequisites](https://cluster-api.sigs.k8s.io/developer/tilt.html).

Clone the capi repository at v1.0.0.

`git clone https://github.com/kubernetes-sigs/cluster-api.git`
`cd cluster-api`
`git checkout v1.0.0`

Setup a kind cluster using
[capi repository]/hack/kind-install-for-capd.sh

There is a tiltfile in the hack directory. Edit the relative repository to match the location of the capi repository.

Copy the tiltfile into the capi repo.

Run `tilt up` from the capi repository.

## Running CAPC without Tilt - detailed instructions

Generally speaking, this cloudstack infrastructure provider will generate a docker image and 3 yaml files. `clusterctl` (a binary tool) will use the above docker image and 3 yaml files to provision a cluster from your local machine using cloudstack as a provider.

### Prerequisites:
Assuming your running environment is MacOS:

1. Follow the instructions [here](https://cluster-api.sigs.k8s.io/user/quick-start.html) to install the following tools:
    1. Docker
    2. KinD
    3. KubeCtl
    4. ClusterCtl - note that depending on the version of CAPC used, you may need different clusterctl versions. Download the binary by finding the necessary release [here](https://github.com/kubernetes-sigs/cluster-api/releases) ([v0.3.23](https://github.com/kubernetes-sigs/cluster-api/releases/tag/v0.3.23) for v1alpha3, [v1.0.2](https://github.com/kubernetes-sigs/cluster-api/releases/tag/v1.0.2) for v1beta1). Download the appropriate executable assets (darwin-arm64 for Macbook) and add them to your PATH

2. [Install cilium-cli](https://formulae.brew.sh/formula/cilium-cli) with `brew install cilium-cli`, or find instructions [here](https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/). Another container networking choice is to use [kindnet](https://github.com/aojea/kindnet), but its usage will not be described here

3. Create a local docker registry to save your docker image - otherwise, you need an image registry to push it somewhere else.
   
4. Download this [script](https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/main/hack/kind-install-for-capd.sh) into your local and run it.
   This script will create a kind cluster and configure it to use local docker registry:
    ```
    wget https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/main/hack/kind-install-for-capd.sh
    chmod +x ./kind-install-for-capd.sh
    ./kind-install-for-capd.sh
    ```
5. Set up CloudStack credentials
    1. Create a file named `cloud-config` in the repo's root directory, substituting in your own environment's values
        ```
        [Global]
        api-url = <cloudstackApiUrl>
        api-key = <cloudstackApiKey>
        secret-key = <cloudstackSecretKey>
        ```

    2. Run the following command to save the above cloudstack connection info into an environment variable, to be used by `./config/default/credentials.yaml`:

        ```
        export CLOUDSTACK_B64ENCODED_SECRET=`base64 -i cloud-config`
        ```
6. Set the IMG environment variable so that the Makefile knows where to push docker image (if building your own)
   1. `export IMG=localhost:5000/cluster-api-provider-capc`
   2. `make docker-build`
   3. `make docker-push`
   
7. Set the source image location so that the CAPC deployment manifest files have the right image path in them in `config/default/manager_image_patch.yaml`

8. Generate the CAPC manifests (if building your own) into `$RELEASE_DIR`
   1. `make build` will generate and copy `infrastructure-components.yaml` and metadata.yaml files to `$RELEASE_DIR`, which is `./out` by default. You may want to override the default value with `export RELEASE_DIR=${HOME}/.cluster-api/overrides/infrastructure-cloudstack/<VERSION>/` to deploy the generated manifests for use by clusterctl before running `make build`.


8. Generate clusterctl config file so that clusterctl knows how to provision the CloudStack cluster, referencing whatever you set for `$RELEASE_DIR` from above for the url:
    ```
    cat << EOF > ~/.cluster-api/cloudstack.yaml
    providers:
    - name: "cloudstack"
      type: "InfrastructureProvider"
      url: ${HOME}/.cluster-api/overrides/infrastructure-cloudstack/<VERSION>/infrastructure-components.yaml
    EOF
    ```

9. Assure that the required CloudStack resources have been created: zone, pod cluster, and k8s-compatible template, compute offerings defined (2GB+ of RAM for control plane offering).

### Creating a CAPC Cluster:

1. Run the following command to turn your previously generated kind cluster into a management cluster and load the cloudstack components into it.
    1. `clusterctl init --infrastructure cloudstack --config ~/.cluster-api/cloudstack.yaml`

2. Set up the environment variables used by `templates/cluster-template.yaml`. It will be populated by the values set here. See the example values below (and replace with your own!)
    ```
    # Machine offerings must be pre-created. Control plane offering
    # must have have >2GB RAM available
    export CLOUDSTACK_WORKER_MACHINE_OFFERING="Small Instance"
    export CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING="Large Instance"
    
    # If the referenced network doesn't exist, a new isolated network
    # will be created.
    export CLOUDSTACK_NETWORK_NAME=GuestNet1
    export CLOUDSTACK_SSH_KEY_NAME=CAPCKeyPair6
    # Referring to a prerequisite kubernetes-compatible image you've loaded into CloudStack
    export CLOUDSTACK_TEMPLATE_NAME=kube-v1.20.10/ubuntu-2004
    export CLOUDSTACK_ZONE_NAME=zone1
    
    # The IP you put here must be available as an unused public IP on the network 
    # referenced above. If it's not available, the control plane will fail to create.
    # You can see the list of available IP's when you try allocating a public
    # IP in the network at 
    # Network -> Guest Networks -> <Network Name> -> IP Addresses
    export CLUSTER_ENDPOINT_IP=192.168.1.161
    
    # This is the standard port that the Control Plane process runs on
    export CLUSTER_ENDPOINT_PORT=6443

    # Pick any name for your cluster
    export CLUSTER_NAME="capc-cluster"
    export CONTROL_PLANE_MACHINE_COUNT=1
    export KUBERNETES_VERSION="v1.20.10"
    export WORKER_MACHINE_COUNT=1
    ```

3. Generate the CAPC cluster spec yaml file
    ```
    clusterctl generate cluster \
        --from ~/.cluster-api/overrides/infrastructure-cloudstack/<VERSION>/cluster-template.yaml \
        > capc-cluster-spec.yaml
    
    ```

4. Apply the CAPC cluster spec to your kind management cluster
    ```
    kubectl apply -f capc-cluster-spec.yaml
    ```

5. Check the progress of capc-cluster, and wait for all the components to be ready
    ```
    clusterctl describe cluster capc-cluster 
    ```

6. Get the generated kubeconfig for your newly created CloudStack cluster `capc-cluster`
    ```
    clusterctl get kubeconfig capc-cluster > capc-cluster.kubeconfig
    ```

7. Install cilium on the workload cluster so that pods can see each other
    ```
    KUBECONFIG=capc-cluster.kubeconfig cilium install
    ```
    1. cilium must be installed into this newly created capc-cluster
    2. Run `KUBECONFIG=capc-cluster.kubeconfig cilium status` to confirm cilium status

8. Verify the K8s cluster is fully up
   1. Run `KUBECONFIG=capc-cluster.kubeconfig get nodes`, and observe the following output
   ```
   NAME                               STATUS   ROLES                  AGE     VERSION
   capc-cluster-control-plane-xsnxt   Ready    control-plane,master   2m56s   v1.20.10
   capc-cluster-md-0-9fr9d            Ready    <none>                 112s    v1.20.10
   ```

### Validating the CAPC Cluster:

Run a simple kubernetes app called 'test-thing'
1. Create the container
```
KUBECONFIG=capc-cluster.kubeconfig kubectl run test-thing --image=rockylinux/rockylinux:8 --restart=Never -- /bin/bash -c 'echo Hello, World!'
KUBECONFIG=capc-cluster.kubeconfig kubectl get pods
 ```
2. Wait for the container to complete, and check the logs for 'Hello, World!'
```
KUBECONFIG=capc-cluster.kubeconfig kubectl logs test-thing
```
### kubectl/clusterctl Reference:
- Pods in capc-cluster -- cluster running in cloudstack
```
cluster-api-provider-cloudstack-staging % KUBECONFIG=capc-cluster.kubeconfig kubectl get pods -A    
NAMESPACE     NAME                                                       READY   STATUS      RESTARTS   AGE
default       test-thing                                                 0/1     Completed   0          2m43s
kube-system   cilium-jxw68                                               1/1     Running     0          6m
kube-system   cilium-nw9x6                                               1/1     Running     0          6m
kube-system   cilium-operator-885b58448-c6wtq                            1/1     Running     0          6m
kube-system   coredns-74ff55c5b-n6zp7                                    1/1     Running     0          9m18s
kube-system   coredns-74ff55c5b-r8gvj                                    1/1     Running     0          9m18s
kube-system   etcd-capc-cluster-control-plane-tknwx                      1/1     Running     0          9m21s
kube-system   kube-apiserver-capc-cluster-control-plane-tknwx            1/1     Running     0          9m21s
kube-system   kube-controller-manager-capc-cluster-control-plane-tknwx   1/1     Running     0          9m21s
kube-system   kube-proxy-6g9zb                                           1/1     Running     0          9m3s
kube-system   kube-proxy-7gjbv                                           1/1     Running     0          9m18s
kube-system   kube-scheduler-capc-cluster-control-plane-tknwx            1/1     Running     0          9m21s
```

- Pods in original kind cluster (also called bootstrap cluster, management cluster)
```
cluster-api-provider-cloudstack-staging % kubectl  get pods -A
NAMESPACE                           NAME                                                             READY   STATUS    RESTARTS   AGE
capc-system                         capc-controller-manager-55798f8594-lp2xs                         1/1     Running   0          30m
capi-kubeadm-bootstrap-system       capi-kubeadm-bootstrap-controller-manager-7857cd7bb8-rldnw       1/1     Running   0          30m
capi-kubeadm-control-plane-system   capi-kubeadm-control-plane-controller-manager-6cc4b4d964-tz5zq   1/1     Running   0          30m
capi-system                         capi-controller-manager-7cfcfdf99b-79lr9                         1/1     Running   0          30m
cert-manager                        cert-manager-848f547974-dl7hc                                    1/1     Running   0          31m
cert-manager                        cert-manager-cainjector-54f4cc6b5-gfgsw                          1/1     Running   0          31m
cert-manager                        cert-manager-webhook-7c9588c76-5m2b2                             1/1     Running   0          31m
kube-system                         coredns-558bd4d5db-22zql                                         1/1     Running   0          48m
kube-system                         coredns-558bd4d5db-7g7kh                                         1/1     Running   0          48m
kube-system                         etcd-capi-test-control-plane                                     1/1     Running   0          48m
kube-system                         kindnet-7p2dq                                                    1/1     Running   0          48m
kube-system                         kube-apiserver-capi-test-control-plane                           1/1     Running   0          48m
kube-system                         kube-controller-manager-capi-test-control-plane                  1/1     Running   0          48m
kube-system                         kube-proxy-cwrhv                                                 1/1     Running   0          48m
kube-system                         kube-scheduler-capi-test-control-plane                           1/1     Running   0          48m
local-path-storage                  local-path-provisioner-547f784dff-f2g7r                          1/1     Running   0          48m
```
