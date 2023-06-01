<!--
// ANCHOR: common-development -->
### Creating a CAPC Cluster:

1. Set up the environment variables. It will be populated by the values set here. See the example values below (and replace with your own!)

    The entire list of configuration variables as well as how to fetch them can be found [here](./clustercloudstack/configuration.html)
    ```
    # The Apache CloudStack zone in which the cluster is to be deployed
    export CLOUDSTACK_ZONE_NAME=zone1

    # If the referenced network doesn't exist, a new isolated network
    # will be created.
    export CLOUDSTACK_NETWORK_NAME=GuestNet1

    # The IP you put here must be available as an unused public IP on the network
    # referenced above. If it's not available, the control plane will fail to create.
    # You can see the list of available IP's when you try allocating a public
    # IP in the network at
    # Network -> Guest Networks -> <Network Name> -> IP Addresses
    export CLUSTER_ENDPOINT_IP=192.168.1.161

    # This is the standard port that the Control Plane process runs on
    export CLUSTER_ENDPOINT_PORT=6443

    # Machine offerings must be pre-created. Control plane offering
    # must have have >2GB RAM available
    export CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING="Large Instance"
    export CLOUDSTACK_WORKER_MACHINE_OFFERING="Small Instance"

    # Referring to a prerequisite capi-compatible image you've loaded into Apache CloudStack
    export CLOUDSTACK_TEMPLATE_NAME=kube-v1.23.3/ubuntu-2004

    # The SSH KeyPair to log into the VM (Optional: you must use clusterctl --flavor *managed-ssh*)
    export CLOUDSTACK_SSH_KEY_NAME=CAPCKeyPair6
    ```

2. Generate the CAPC cluster spec yaml file
    ```
    clusterctl generate cluster capc-cluster \
        --kubernetes-version v1.23.3 \
        --control-plane-machine-count=1 \
        --worker-machine-count=1 \
        > capc-cluster-spec.yaml

    ```

3. Apply the CAPC cluster spec to your kind management cluster
    ```
    kubectl apply -f capc-cluster-spec.yaml
    ```

4. Check the progress of capc-cluster, and wait for all the components (with the exception of MachineDeployment/capc-cluster-md-0) to be ready.  (MachineDeployment/capc-cluster-md-0 will not show ready until the CNI is installed.)
    ```
    clusterctl describe cluster capc-cluster
    ```

5. Get the generated kubeconfig for your newly created Apache CloudStack cluster `capc-cluster`
    ```
    clusterctl get kubeconfig capc-cluster > capc-cluster.kubeconfig
    ```

6. Install calico or weave net cni plugin on the workload cluster so that pods can see each other
    ```
    KUBECONFIG=capc-cluster.kubeconfig kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/master/manifests/calico.yaml
    
    ```    
     or
    
    ```
    KUBECONFIG=capc-cluster.kubeconfig kubectl apply -f https://raw.githubusercontent.com/weaveworks/weave/master/prog/weave-kube/weave-daemonset-k8s-1.11.yaml
    
    ```

7. Verify the K8s cluster is fully up.  (It may take a minute for the nodes status to all reach *ready* state.)
   1. Run `KUBECONFIG=capc-cluster.kubeconfig kubectl get nodes`, and observe the following output
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
- Pods in capc-cluster -- cluster running in Apache CloudStack with calico cni
```
% KUBECONFIG=capc-cluster.kubeconfig kubectl get pods -A
NAMESPACE     NAME                                                       READY   STATUS      RESTARTS   AGE
default       test-thing                                                 0/1     Completed   0          2m43s
kube-system   calico-kube-controllers-784dcb7597-dw42t                   1/1     Running     0          4m31s
kube-system   calico-node-mmp2x                                          1/1     Running     0          4m31s
kube-system   calico-node-vz99f                                          1/1     Running     0          4m31s
kube-system   coredns-74ff55c5b-n6zp7                                    1/1     Running     0          9m18s
kube-system   coredns-74ff55c5b-r8gvj                                    1/1     Running     0          9m18s
kube-system   etcd-capc-cluster-control-plane-tknwx                      1/1     Running     0          9m21s
kube-system   kube-apiserver-capc-cluster-control-plane-tknwx            1/1     Running     0          9m21s
kube-system   kube-controller-manager-capc-cluster-control-plane-tknwx   1/1     Running     0          9m21s
kube-system   kube-proxy-6g9zb                                           1/1     Running     0          9m3s
kube-system   kube-proxy-7gjbv                                           1/1     Running     0          9m18s
kube-system   kube-scheduler-capc-cluster-control-plane-tknwx            1/1     Running     0          9m21s
```
- Pods in capc-cluster -- cluster running in Apache CloudStack with weave net cni

```
%KUBECONFIG=capc-cluster.kubeconfig kubectl get pods -A
NAMESPACE     NAME                                                       READY   STATUS      RESTARTS       AGE
default       test-thing                                                 0/1     Completed   0              38s
kube-system   coredns-5d78c9869d-9xq2s                                   1/1     Running     0              21h
kube-system   coredns-5d78c9869d-gphs2                                   1/1     Running     0              21h
kube-system   etcd-capc-cluster-control-plane-49khm                      1/1     Running     0              21h
kube-system   kube-apiserver-capc-cluster-control-plane-49khm            1/1     Running     0              21h
kube-system   kube-controller-manager-capc-cluster-control-plane-49khm   1/1     Running     0              21h
kube-system   kube-proxy-8lfnm                                           1/1     Running     0              21h
kube-system   kube-proxy-brj78                                           1/1     Running     0              21h
kube-system   kube-scheduler-capc-cluster-control-plane-49khm            1/1     Running     0              21h
kube-system   weave-net-rqckr                                            2/2     Running     1 (3h8m ago)   3h8m
kube-system   weave-net-rzms4                                            2/2     Running     1 (3h8m ago)   3h8m
```

- Pods in original kind cluster (also called bootstrap cluster, management cluster)
```
% kubectl  get pods -A
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
<!--
// ANCHOR_END: common-development
-->
