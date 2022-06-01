# Unstacked etcd

There are two types of etcd topologies for configuring a Kubernetes cluster:
- Stacked: The etcd members and control plane components are co-located (run on the same node/machines)
- Unstacked/External: With the unstacked or external etcd topology, etcd members have dedicated machines and are not co-located with control plane components

The unstacked etcd topology is recommended for a HA cluster for the following reasons:
- External etcd topology decouples the control plane components and etcd member. So if a control plane-only node fails, or if there is a memory leak in a component like kube-apiserver, it won’t directly impact an etcd member.
- Etcd is resource intensive, so it is safer to have dedicated nodes for etcd, since it could use more disk space or higher bandwidth. Having a separate etcd cluster for these reasons could ensure a more resilient HA setup.

More details can be found [here][unstacked-etcd]

## Local storage

In this configuration, the storage used by etcd is a local directory on the node, rather than within the container itself. This provides resilience in case the etcd pod is terminated as no data is lost, and can be read from the local directory once the new etcd pod comes up

Local storage for etcd can be configured by adding the `etcd` field to the `KubeadmControlPlane.spec.kubeadmConfigSpec.clusterConfiguration` spec.
The value should point to an empty directory on the node

```yaml
  kubeadmConfigSpec:
    clusterConfiguration:
      imageRepository: k8s.gcr.io
      etcd:
        local:
          dataDir: /var/lib/etcddisk/etcd
```

If the user wishes to use a separate data disk as local storage, the can be formatted and mounted as shown :

```yaml
  kubeadmConfigSpec:
    diskSetup:
      filesystems:
      - device: /dev/vdb
        filesystem: ext4
        label: etcd_disk
    mounts:
    - - LABEL=etcd_disk
      - /var/lib/etcddisk
```

## External etcd

In this configuration, etcd does not run on the Kubernetes Cluster. Instead, the Kubernetes Cluster uses an externally managed etcd cluster.
This provides additional availability if the entire control plane node goes down, the caveat being that the externally managed etcd cluster must be always available.

External etcd can be configured by adding the `etcd` field to the `KubeadmControlPlane.spec.kubeadmConfigSpec.clusterConfiguration` spec.
The value should point to an empty directory on the node

```yaml
  kubeadmConfigSpec:
    clusterConfiguration:
      imageRepository: k8s.gcr.io
      etcd:
        external:
          endpoints:
            - ${ETCD_ENDPOINT}
          caFile: /etc/kubernetes/pki/etcd/ca.crt
          certFile: /etc/kubernetes/pki/apiserver-etcd-client.crt
          keyFile: /etc/kubernetes/pki/apiserver-etcd-client.key
```

Additionally, the certificates have to be passed as Secrets as shown below :

```yaml
# Ref: https://github.com/kubernetes-retired/cluster-api-bootstrap-provider-kubeadm/blob/master/docs/external-etcd.md
kind: Secret
apiVersion: v1
metadata:
  name: ${CLUSTER_NAME}-apiserver-etcd-client
data:
  # base64 encoded /etc/etcd/pki/apiserver-etcd-client.crt
  tls.crt: |
    ${BASE64_ENCODED__APISERVER_ETCD_CLIENT_CRT}
  # base64 encoded /etc/etcd/pki/apiserver-etcd-client.key
  tls.key: |
    ${BASE64_ENCODED__APISERVER_ETCD_CLIENT_KEY}
```

<!-- References -->

[unstacked-etcd]: https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/ha-topology/
