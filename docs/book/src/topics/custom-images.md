# Custom Images

This document will help you get a CAPC Kubernetes cluster up and running with your custom image.

## Prebuilt Images

An *image* defines the operating system and Kubernetes components that will populate the disk of each node in your cluster.

As of now, prebuilt images for KVM, VMware and XenServer are available [here][prebuilt-images]

## Building a custom image

Cluster API uses the Kubernetes [Image Builder][image-builder] tools. You should use the [QEMU images][image-builder-qemu] from that project as a starting point for your custom image.

[The Image Builder Book][capi-images] explains how to build the images defined in that repository, with instructions for [CloudStack CAPI Images][cloudstack-capi-images] in particular.

The image is built using KVM hypervisor as a `qcow2` image.
Depending on they hypervisor requirements, it can then converted into `ova` for VMware and `vhd` for XenServer via the `convert-cloudstack-image.sh` script.

### Operating system requirements

For your custom image to work with Cluster API, it must meet the operating system requirements of the bootstrap provider. For example, the default `kubeadm` bootstrap provider has a set of [`preflight checks`][kubeadm-preflight-checks] that a VM is expected to pass before it can join the cluster.

### Kubernetes version requirements

The reference images are each built to support a specific version of Kubernetes. When using your custom images based on them, take care to match the image to the `version:` field of the `KubeadmControlPlane` and `MachineDeployment` in the YAML template for your workload cluster.

## Creating a cluster from a custom image

To use a custom image, it needs to be referenced in an `image:` section of your `CloudStackMachineTemplate`.
Be sure to also update the `version` in the `KubeadmControlPlane` and `MachineDeployment` cluster spec.

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: CloudStackMachineTemplate
metadata:
  name: capi-quickstart-control-plane
spec:
  template:
    spec:
      offering: ControlPlaneOffering
      template: custom-image-name
```

## Upgrading Kubernetes Versions

To upgrade to a new Kubernetes release with custom images requires this preparation:

- Create a new custom image which supports the Kubernetes release version
- Register the custom image as a template in Apache CloudStack
- Copy the existing `CloudStackMachineTemplate` and change its `image:` section to reference the new custom image
- Create the new `CloudStackMachineTemplate` on the management cluster
- Modify the existing `KubeadmControlPlane` and `MachineDeployment` to reference the new `CloudStackMachineTemplate` and update the `version:` field to match

See [Upgrading workload clusters][upgrading-workload-clusters] for more details.

<!-- References -->

[capi-images]: https://image-builder.sigs.k8s.io/capi/capi.html
[cloudstack-capi-images]: https://image-builder.sigs.k8s.io/capi/providers/cloudstack.html
[image-builder]: https://github.com/kubernetes-sigs/image-builder
[image-builder-qemu]: https://github.com/kubernetes-sigs/image-builder/tree/master/images/capi/packer/qemu
[kubeadm-preflight-checks]: https://github.com/kubernetes/kubeadm/blob/master/docs/design/design_v1.10.md#preflight-checks
[prebuilt-images]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/
[upgrading-workload-clusters]: https://cluster-api.sigs.k8s.io/tasks/kubeadm-control-plane.html#upgrading-workload-clusters
