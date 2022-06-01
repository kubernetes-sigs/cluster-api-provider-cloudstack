# Move From Bootstrap

This documentation describes how to move `Cluster API` related objects from `bootstrap` cluster to `target` cluster.
Check [clusterctl move][clusterctl-move] for further information.

# Pre-condition

Bootstrap cluster
```
# kubectl get pods --all-namespaces
NAMESPACE                           NAME                                                             READY   STATUS    RESTARTS   AGE
capc-system                         capc-controller-manager-5d8b989c5c-zqvcn                         1/1     Running   0          23m
capi-kubeadm-bootstrap-system       capi-kubeadm-bootstrap-controller-manager-58db4b5555-crcc8       1/1     Running   0          23m
capi-kubeadm-control-plane-system   capi-kubeadm-control-plane-controller-manager-86c4dcbc4c-8xvql   1/1     Running   0          23m
capi-system                         capi-controller-manager-56f77c8f7b-s5q8f                         1/1     Running   0          23m
cert-manager                        cert-manager-848f547974-tzqtg                                    1/1     Running   0          23m
cert-manager                        cert-manager-cainjector-54f4cc6b5-hslzq                          1/1     Running   0          23m
cert-manager                        cert-manager-webhook-7c9588c76-pz42g                             1/1     Running   0          23m
kube-system                         coredns-558bd4d5db-2xxlz                                         1/1     Running   0          34m
kube-system                         coredns-558bd4d5db-wjdbw                                         1/1     Running   0          34m
kube-system                         etcd-kind-control-plane                                          1/1     Running   0          34m
kube-system                         kindnet-lkgjb                                                    1/1     Running   0          34m
kube-system                         kube-apiserver-kind-control-plane                                1/1     Running   0          34m
kube-system                         kube-controller-manager-kind-control-plane                       1/1     Running   0          34m
kube-system                         kube-proxy-gv7pv                                                 1/1     Running   0          34m
kube-system                         kube-scheduler-kind-control-plane                                1/1     Running   0          34m
local-path-storage                  local-path-provisioner-547f784dff-79kq4                          1/1     Running   0          34m
```

Target cluster
```
# kubectl get pods --kubeconfig target.kubeconfig --all-namespaces
NAMESPACE            NAME                                           READY   STATUS    RESTARTS   AGE
kube-system          calico-kube-controllers-784dcb7597-dw42t       1/1     Running   0          41m
kube-system          calico-node-mmp2x                              1/1     Running   0          41m
kube-system          calico-node-vz99f                              1/1     Running   0          41m
kube-system          coredns-558bd4d5db-5pvfm                       1/1     Running   0          43m
kube-system          coredns-558bd4d5db-gcv5j                       1/1     Running   0          43m
kube-system          etcd-target-control-plane                      1/1     Running   0          43m
kube-system          kindnet-4w84z                                  1/1     Running   0          43m
kube-system          kube-apiserver-target-control-plane            1/1     Running   0          43m
kube-system          kube-controller-manager-target-control-plane   1/1     Running   0          43m
kube-system          kube-proxy-zstvt                               1/1     Running   0          43m
kube-system          kube-scheduler-target-control-plane            1/1     Running   0          43m
```

The bootstrap cluster is currently managing an existing workload cluster
```
# clusterctl describe cluster cloudstack-capi
NAME                                                                                 READY  SEVERITY  REASON  SINCE  MESSAGE
Cluster/cloudstack-capi                                                              True                     9m31s
├─ClusterInfrastructure - CloudStackCluster/cloudstack-capi
├─ControlPlane - KubeadmControlPlane/cloudstack-capi-control-plane                   True                     9m31s
│ └─Machine/cloudstack-capi-control-plane-xhgb9                                      True                     9m51s
│   └─MachineInfrastructure - CloudStackMachine/cloudstack-capi-control-plane-59qrb
└─Workers
  └─MachineDeployment/cloudstack-capi-md-0                                           True                     7m20s
    └─Machine/cloudstack-capi-md-0-75499bbf6-zqktd                                   True                     8m56s
      └─MachineInfrastructure - CloudStackMachine/cloudstack-capi-md-0-cl5ht
```

# Install Cloudstack Cluster API provider into target cluster

You need install Apache CloudStack Cluster API providers into `target` cluster first.
```
# clusterctl --kubeconfig target.kubeconfig init --infrastructure cloudstack
Fetching providers
Installing cert-manager Version="v1.5.3"
Waiting for cert-manager to be available...
Installing Provider="cluster-api" Version="v1.1.3" TargetNamespace="capi-system"
Installing Provider="bootstrap-kubeadm" Version="v1.1.3" TargetNamespace="capi-kubeadm-bootstrap-system"
Installing Provider="control-plane-kubeadm" Version="v1.1.3" TargetNamespace="capi-kubeadm-control-plane-system"
Installing Provider="infrastructure-cloudstack" Version="v1.0.0" TargetNamespace="capc-system"

Your management cluster has been initialized successfully!

You can now create your first workload cluster by running the following:

  clusterctl generate cluster [name] --kubernetes-version [version] | kubectl apply -f -

```

# Move objects from `bootstrap` cluster into `target` cluster.

CRD, objects such as `CloudstackCluster`, `CloudstackMachine` etc need to be moved.
```
# clusterctl move --to-kubeconfig target.kubeconfig -v 10
```
```
Using configuration File="/home/djumani/.cluster-api/clusterctl.yaml"
Performing move...
Discovering Cluster API objects
Cluster Count=1
KubeadmConfigTemplate Count=1
KubeadmControlPlane Count=1
MachineDeployment Count=1
MachineSet Count=1
CloudStackCluster Count=1
CloudStackMachine Count=2
CloudStackMachineTemplate Count=2
Machine Count=2
KubeadmConfig Count=2
ConfigMap Count=1
Secret Count=8
Total objects Count=23
Excluding secret from move (not linked with any Cluster) name="default-token-nd9nb"
Moving Cluster API objects Clusters=1
Moving Cluster API objects ClusterClasses=0
Pausing the source cluster
Set Cluster.Spec.Paused Paused=true Cluster="cloudstack-capi" Namespace="default"
Pausing the source cluster classes
Creating target namespaces, if missing
Creating objects in the target cluster
Creating Cluster="cloudstack-capi" Namespace="default"
Creating CloudStackMachineTemplate="cloudstack-capi-md-0" Namespace="default"
Creating KubeadmControlPlane="cloudstack-capi-control-plane" Namespace="default"
Creating CloudStackCluster="cloudstack-capi" Namespace="default"
Creating KubeadmConfigTemplate="cloudstack-capi-md-0" Namespace="default"
Creating MachineDeployment="cloudstack-capi-md-0" Namespace="default"
Creating CloudStackMachineTemplate="cloudstack-capi-control-plane" Namespace="default"
Creating Secret="cloudstack-capi-proxy" Namespace="default"
Creating Machine="cloudstack-capi-control-plane-xhgb9" Namespace="default"
Creating Secret="cloudstack-capi-ca" Namespace="default"
Creating Secret="cloudstack-capi-etcd" Namespace="default"
Creating MachineSet="cloudstack-capi-md-0-75499bbf6" Namespace="default"
Creating Secret="cloudstack-capi-kubeconfig" Namespace="default"
Creating Secret="cloudstack-capi-sa" Namespace="default"
Creating Machine="cloudstack-capi-md-0-75499bbf6-zqktd" Namespace="default"
Creating CloudStackMachine="cloudstack-capi-control-plane-59qrb" Namespace="default"
Creating KubeadmConfig="cloudstack-capi-control-plane-r6ns8" Namespace="default"
Creating KubeadmConfig="cloudstack-capi-md-0-z9ndx" Namespace="default"
Creating Secret="cloudstack-capi-control-plane-r6ns8" Namespace="default"
Creating CloudStackMachine="cloudstack-capi-md-0-cl5ht" Namespace="default"
Creating Secret="cloudstack-capi-md-0-z9ndx" Namespace="default"
Deleting objects from the source cluster
Deleting Secret="cloudstack-capi-md-0-z9ndx" Namespace="default"
Deleting KubeadmConfig="cloudstack-capi-md-0-z9ndx" Namespace="default"
Deleting Secret="cloudstack-capi-control-plane-r6ns8" Namespace="default"
Deleting CloudStackMachine="cloudstack-capi-md-0-cl5ht" Namespace="default"
Deleting Machine="cloudstack-capi-md-0-75499bbf6-zqktd" Namespace="default"
Deleting CloudStackMachine="cloudstack-capi-control-plane-59qrb" Namespace="default"
Deleting KubeadmConfig="cloudstack-capi-control-plane-r6ns8" Namespace="default"
Deleting Secret="cloudstack-capi-proxy" Namespace="default"
Deleting Machine="cloudstack-capi-control-plane-xhgb9" Namespace="default"
Deleting Secret="cloudstack-capi-ca" Namespace="default"
Deleting Secret="cloudstack-capi-etcd" Namespace="default"
Deleting MachineSet="cloudstack-capi-md-0-75499bbf6" Namespace="default"
Deleting Secret="cloudstack-capi-kubeconfig" Namespace="default"
Deleting Secret="cloudstack-capi-sa" Namespace="default"
Deleting CloudStackMachineTemplate="cloudstack-capi-md-0" Namespace="default"
Deleting KubeadmControlPlane="cloudstack-capi-control-plane" Namespace="default"
Deleting CloudStackCluster="cloudstack-capi" Namespace="default"
Deleting KubeadmConfigTemplate="cloudstack-capi-md-0" Namespace="default"
Deleting MachineDeployment="cloudstack-capi-md-0" Namespace="default"
Deleting CloudStackMachineTemplate="cloudstack-capi-control-plane" Namespace="default"
Deleting Cluster="cloudstack-capi" Namespace="default"
Resuming the target cluter classes
Resuming the target cluster
Set Cluster.Spec.Paused Paused=false Cluster="cloudstack-capi" Namespace="default"
```
# Check cluster status

```
# clusterctl --kubeconfig target.kubeconfig describe cluster cloudstack-capi
NAME                                                                                 READY  SEVERITY  REASON  SINCE  MESSAGE
Cluster/cloudstack-capi                                                              True                     107s
├─ClusterInfrastructure - CloudStackCluster/cloudstack-capi
├─ControlPlane - KubeadmControlPlane/cloudstack-capi-control-plane                   True                     107s
│ └─Machine/cloudstack-capi-control-plane-xhgb9                                      True                     115s
│   └─MachineInfrastructure - CloudStackMachine/cloudstack-capi-control-plane-59qrb
└─Workers
  └─MachineDeployment/cloudstack-capi-md-0                                           True                     115s
    └─Machine/cloudstack-capi-md-0-75499bbf6-zqktd                                   True                     115s
      └─MachineInfrastructure - CloudStackMachine/cloudstack-capi-md-0-cl5ht

# kubectl get cloudstackcluster --kubeconfig target.kubeconfig --all-namespaces
NAMESPACE   NAME              CLUSTER           READY   NETWORK
default     cloudstack-capi   cloudstack-capi   true

# kubectl get cloudstackmachines --kubeconfig target.kubeconfig --all-namespaces
NAMESPACE   NAME                                  CLUSTER           INSTANCESTATE   READY   PROVIDERID                                           MACHINE
default     cloudstack-capi-control-plane-59qrb   cloudstack-capi                   true    cloudstack:///81b9d65e-b365-4535-956f-9f845b730c54   cloudstack-capi-control-plane-xhgb9
default     cloudstack-capi-md-0-cl5ht            cloudstack-capi                   true    cloudstack:///a05408b1-47fe-46b5-b128-a9d5492eaabf   cloudstack-capi-md-0-75499bbf6-zqktd

```

<!-- References -->

[clusterctl-move]: https://cluster-api.sigs.k8s.io/clusterctl/commands/move.html
