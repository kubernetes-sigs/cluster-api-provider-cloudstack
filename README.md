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

## Tutorial w/o Tilt - with detailed instructions

To run capc locally and against cloudstack environments:

Generally speaking, this cloudstack infrastructure provider will generate a docker image and 3 yaml files.

clusterctl (a binary tool) will use above docker image and 3 yaml files to provision a cluster using cloudstack as a provider.

### Some tools: 
assuming your running environment is macOs.
- please follow this [link](https://cluster-api.sigs.k8s.io/user/quick-start.html) to have following tools installed
 * docker
 * kind
 * kubectl
 * clusterctl

- [install cilium-cli](https://formulae.brew.sh/formula/cilium-cli) - `brew install cilium-cli` - another choice is to use [kindnet](https://github.com/aojea/kindnet)
- gotcha: 
 * use `clusterctl version` to confirm your version. 

- create a local docker registry to save your docker image - otherwise, you need extra account to push it somewhere else
Download this [script](https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/main/hack/kind-install-for-capd.sh) into your local and run it.
This script will create a kind cluster and configure it to use local docker registry

* `wget https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/main/hack/kind-install-for-capd.sh`
* `chmod +x ./kind-install-for-capd.sh`
* `./kind-install-for-capd.sh`

- create a file named `cloud-config` in same directory as this README.md
```
[Global]
api-url = http://192.168.1.129:8080/client/api
api-key = bJStwHrk32ojY70rfgkyq7GVqf1Sb0I40BMzSZMID_2nT2_KkWzDBX3rddr8WB-QJmXXiYovlCk8k8mMkRywlg
secret-key = fNLrsn8tafVd0aVJANFu25gWpn-_f7lejBdFb6LpOJqxXShJTxmz0zFmkSINWcJg-Ud4dkacZzRqmcdloC7MDw
```

run following command to save above cloudstack connection info into an environment variable:

```
export CLOUDSTACK_B64ENCODED_SECRET=`base64 -i cloud-config`
```

./config/default/credentials.yaml is using above env var.

- gotcha:
 * replace the sample with your own cloudstack info in the above file. 

- set IMG env var so that ./Makefile knows where to push docker image
 * `export IMG=localhost:5000/cluster-api-provider-capc`
 * `make docker-push`

- generate manifest
 * `make dev-manifests` this will copy infrastructure-components.yaml file to ~/.cluster-api/overrides/infrastructure-cloudstack/v0.1.0/


- generate clusterctl config file, so that clusterctl knows how to provision cloudstack cluster:
```
cat << EOF > ~/.cluster-api/cloudstack.yaml
providers:
- name: "cloudstack"
  type: "InfrastructureProvider"
  url: ${HOME}/.cluster-api/overrides/infrastructure-cloudstack/v0.1.0/infrastructure-components.yaml
EOF
```

- run following command to turn kind cluster into a management cluster and load cloudstack component into it.
 * `clusterctl init --infrastructure cloudstack --config ~/.cluster-api/cloudstack.yaml`

- set up env vars used by cluster-template.yaml 
 * cluster template file is here (already existed): ./templates/cluster-template.yaml 
 ```
# Machine offerings must be pre-created. Control plane offering
# must have have >2GB RAM available
export CLOUDSTACK_WORKER_MACHINE_OFFERING="Small Instance"
export CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING="Large Instance"

# If the referenced network doesn't exist, a new isolated network
# will be created.
export CLOUDSTACK_NETWORK_NAME=GuestNet1
export CLOUDSTACK_SSH_KEY_NAME=CAPCKeyPair6
export CLOUDSTACK_TEMPLATE_NAME=kube-v1.20.10/ubuntu-2004
export CLOUDSTACK_ZONE_NAME=zone1

# This IP must be available on the network referenced above. If
# it's not available, the control plane will fail to create.
# You can see the list of available IP's when you try allocating a public
# IP in the network
export CLUSTER_ENDPOINT_IP=192.168.1.161

# This is the standard port that the Control Plane process runs on
export CLUSTER_ENDPOINT_PORT=6443
```

- gotcha 
 * make sure all the env var values matching your cloudstack, offering/template/zone/network/keypair
 * replace 192.168.1.161 with the ip in your GuestNet1 (sourceNat). shows in cloudstack UI: Home -> Network -> Guest networks -> GuestNet1 -> IP Addresses


- generate the capc cluster spec yaml file 
```
clusterctl generate cluster capc-cluster \
    --kubernetes-version v1.20.10 \
    --config ~/.cluster-api/cloudstack.yaml \
    --control-plane-machine-count=1 \
    --worker-machine-count=1 \
    > capc-cluster-spec.yaml

```

- apply capc cluster spec to this kind management cluster
```
kubectl apply -f capc-cluster-spec.yaml
```

- check the progress of capc-cluster
```
clusterctl describe cluster capc-cluster 
```

- get kubeconfig for this newly created cloudstack cluster `capc-cluster`
```
clusterctl get kubeconfig capc-cluster > capc-cluster.kubeconfig
```

- install cilium, so that pods can see each other
```
KUBECONFIG=capc-cluster.kubeconfig cilium install
```
* cilium must be installed into this newly created capc-cluster
* `KUBECONFIG=capc-cluster.kubeconfig cilium status` to confirm cilium status

- run a simple kubenetes app called 'test-thing'
```
KUBECONFIG=capc-cluster.kubeconfig kubectl run test-thing --image=rockylinux/rockylinux:8 --restart=Never -- /bin/bash -c 'echo Hello, World!'
KUBECONFIG=capc-cluster.kubeconfig kubectl  logs test-thing
```

### for your reference: 
- pods in capc-cluster -- cluster running in cloudstack
```
cluster-api-provider-cloudstack-staging % KUBECONFIG=capc-cluster.kubeconfig kubectl  get pods -A    
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

- pods in original kind cluster (also called bootstrap cluster, management cluster)
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

