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

## Running CAPC from local artifacts (manifests, image)

### Prerequisites to starting:

1. Have an instance of cloudstack up and running with a successfully installed k8s compatible template, some compute offering defined (2GB+ of RAM), a zone, pod, and cluster
2. Local clone of CAPC staging repo, and [cluster-api repo](https://github.com/kubernetes-sigs/cluster-api). 
3. CAPC release image path. **Make sure to modify `config/default/manager_image_patch.yaml` file to refer to your desired image revision** (i.e. `- image: localhost:5000/cluster-api-provider-capc:latest` for this guide). 
You can also optionally import the CAPC release image by downloading the tar file from github and running `docker load`, `docker tag`, and `docker push` with the necessary arguments
4. Clusterctl executable installed (v0.4.4 for CAPC-v1alpha4, v1.0.0+ for CAPC-v1beta1). Also kubectl, and cilium-cli.
5. Existing cloud-config file with the following structure
```
[Global]
api-url = <urlOfCloudStackAPI>
api-key = <cloudstackUserApiKey>
secret-key = <cloudstackUserSecretKey>
```

### Running CAPC:

1. delete any previous kind clusters (kind delete cluster)
    1. `kind delete cluster --name capi-test`
2. start a new bootstrap cluster and ensure local docker repo is running with hack script in cluster-api repo: [./kind-install-for-capd.sh](https://github.com/kubernetes-sigs/cluster-api/blob/main/hack/kind-install-for-capd.sh)
    1. .`../cluster-api/hack/kind-install-for-capd.sh`
3. (OPTIONAL, only required if changes were made and using local CAPC code) build code into image, and push to local docker repo (or use an existing released image, loaded into local repo)
    1. ```
       export IMG=localhost:5000/cluster-api-provider-capc
       make docker-build
       make docker-push
       ```
4. (OPTIONAL, only required if changes were made and using local CAPC code) build manifests (infrastructure-components.yaml) into correct directory, and make sure clusterctl-config file is referencing them.
    1. `make dev-manifests`, which writes infrastructure-components.yaml file to `~/.cluster-api/overrides/infrastructure-cloudstack/<CAPC_VERSION>/`
5. Create clusterctl config file which references the manifests generated from the previous step
    1. ```
       cat << EOF > ~/.cluster-api/cloudstack.yaml
       providers:
        - name: "cloudstack"
          type: "InfrastructureProvider"
          url: ${HOME}/.cluster-api/overrides/infrastructure-cloudstack/<CAPC_VERSION>/infrastructure-components.yaml
          EOF
       ```
6. set base 64 encoded cloudstack secret environment variable from cloud-config INI file which has access key, secret key, and url
    1. ```export CLOUDSTACK_B64ENCODED_SECRET=$(base64 -i <PATH_TO_CLOUD_CONFIG_INI>)```
7. load capc components into bootstrap cluster and wait for all the pods to be RUNNING. If they fail, use kubectl describe on the namespaced pods to see what events occurred to make them fail
    1. `clusterctl init --infrastructure cloudstack --config <PATH_TO_CLUSTERCTL_CONFIG_FILE>`
8. set CAPC specific environment variables
    1. ```
       # Machine offerings must be pre-created. Control plane offering
       # must have have >2GB RAM available
       export CLOUDSTACK_WORKER_MACHINE_OFFERING="Small Instance"
       export CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING="Large"

       # If the referenced network doesn't exist, a new isolated network
       # will be created.
       export CLOUDSTACK_NETWORK_NAME=isolated1
       export CLOUDSTACK_SSH_KEY_NAME=CAPCKeyPair6
       # K8s-compatible template should be pre-loaded into cloudstack
       export CLOUDSTACK_TEMPLATE_NAME=kube-v1.20.10/ubuntu-2004
       export CLOUDSTACK_ZONE_NAME=zone1

       # This IP must be available on the network referenced above. If
       # it's not available, the control plane will fail to create.
       # You can see the list of available IP's when you try allocating a public
       # IP in the network
       export CLUSTER_ENDPOINT_IP=172.16.2.99

       # This is the standard port that the Control Plane process runs on
       export CLUSTER_ENDPOINT_PORT=6443
       ```
9. generate capc-cluster spec yaml file, passing in necessary arguments. Note that control-plane-machine-count must be an odd number for leader election to succeed
    1. ```
       clusterctl generate cluster capc-cluster \
       --kubernetes-version v1.20.10 \
       --config <PATH_TO_CLUSTERCTL_CONFIG> \
       --control-plane-machine-count=1 \
       --worker-machine-count=1 \
       > capc-cluster-spec.yaml
       ```
10. apply the cluster spec yaml file generated in the previous step to the bootstrap cluster
    1. `kubectl apply -f capc-cluster-spec.yaml`
11. Observe the progression with `watch clusterctl describe cluster capc-cluster`. If something is stuck, you can look at the controllers’ pods’ logs with `kubectl logs` or in the cloudstack server’s management logs, look for exceptions.
12. Install Cilium
    1. ```
       clusterctl get kubeconfig capc-cluster > capc-cluster.kubeconfig
       KUBECONFIG=capc-cluster.kubeconfig cilium install
       ```
    2. Make sure everything is working with `KUBECONFIG=capc-cluster.kubeconfig kubectl get nodes `Two nodes should be running, and all statuses should be Ready
13. Deploy a job to your cluster!
    1. `KUBECONFIG=capc-cluster.kubeconfig kubectl run test-thing --image=rockylinux/rockylinux:8 --restart=Never -- /bin/bash -c 'echo Hello, World!'`
    2. Observe the logs with `KUBECONFIG=capc-cluster.kubeconfig kubectl logs test-thing`



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
