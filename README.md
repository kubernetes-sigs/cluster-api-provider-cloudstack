
<p align="center">
  <!-- <h1 style="text-align: center"> Kubernetes Cluster API Provider CloudStack </h1> -->
  <a href="https://cloudstack.apache.org/">
    <img width="75%" src="https://raw.githubusercontent.com/shapeblue/cluster-api-provider-cloudstack/add-docs/docs/book/src/images/capc.png"
    alt="Powered by Apache CloudStack"/>
  </a>
  <br /><br /><br />

  <!-- go doc / reference card -->
  <a href="https://pkg.go.dev/sigs.k8s.io/cluster-api-provider-cloudstack">
    <img src="https://pkg.go.dev/badge/sigs.k8s.io/cluster-api-provider-cloudstack">
  </a>
  <!-- goreportcard badge -->
  <a href="https://goreportcard.com/report/sigs.k8s.io/cluster-api-provider-cloudstack">
    <img src="https://goreportcard.com/badge/sigs.k8s.io/cluster-api-provider-cloudstack">
  </a>
  <!-- join kubernetes slack channel for cluster-api-cloudstack-provider -->
  <a href="https://kubernetes.slack.com/messages/cluster-api-cloudstack">
    <img src="https://img.shields.io/badge/join%20slack-%23cluster--api--cloudstack-brightgreen">
  </a>
</p>

------------------------------------------------------------------------------

## What is the Cluster API Provider CloudStack

The [Cluster API][cluster_api] brings declarative, Kubernetes-style APIs to cluster creation, configuration and management.

The API itself is shared across multiple cloud providers allowing for true Apache CloudStack hybrid deployments of Kubernetes.
It is built atop the lessons learned from previous cluster managers such as [kops][kops] and [kubicorn][kubicorn].


## Launching a Kubernetes cluster on Apache CloudStack

Check out the [Getting Started Guide][getting_started] to create your first Kubernetes cluster on Apache CloudStack using Cluster API.

## Features

- Native Kubernetes manifests and API
- Choice of Linux distribution (as long as a current cloud-init is available). Tested on Ubuntu, Centos, Rocky and RHEL
- Support for single and multi-node control plane clusters
- Deploy clusters on Isolated and Shared Networks
- cloud-init based nodes bootstrapping


------

## Compatibility with Cluster API and Kubernetes Versions


This provider's versions are able to install and manage the following versions of Kubernetes:

| Kubernetes Version          | v1.22 | v1.23 | v1.24 |
| --------------------------- | ----- | ----- | ----- |
| CloudStack Provider  (v0.4) |   ✓   |   ✓   |   ✓   |

## Compatibility with Apache CloudStack Versions


This provider's versions are able to work on the following versions of Apache CloudStack:

| CloudStack Version          | 4.14 | 4.15 | 4.16 | 4.17 |
| --------------------------- | ---- | ---- | ---- | ---- |
| CloudStack Provider  (v0.4) |   ✓  |   ✓  |   ✓  |   ✓  |

------

## Operating system images

Note: Cluster API Provider CloudStack relies on a few prerequisites which have to be already
installed in the used operating system images, e.g. a container runtime, kubelet, kubeadm, etc.
Reference images can be found in [kubernetes-sigs/image-builder][image-builder].

Prebuilt images can be found below :

| Hypervisor | Kubernetes Version | Rocky Linux 8                                        | Ubuntu 20.04                                         |
| ---------- | ------------------ | ---------------------------------------------------- | ---------------------------------------------------- |
| KVM        | v1.22              | [qcow2][k1.22-rl8-qcow2], [md5][k1.22-rl8-qcow2-md5] | [qcow2][k1.22-u20-qcow2], [md5][k1.22-u20-qcow2-md5] |
|            | v1.23              | [qcow2][k1.23-rl8-qcow2], [md5][k1.23-rl8-qcow2-md5] | [qcow2][k1.23-u20-qcow2], [md5][k1.23-u20-qcow2-md5] |
|            | v1.24              | [qcow2][k1.24-rl8-qcow2], [md5][k1.24-rl8-qcow2-md5] | [qcow2][k1.24-u20-qcow2], [md5][k1.24-u20-qcow2-md5] |
| VMware     | v1.22              | [ova][k1.22-rl8-ova], [md5][k1.22-rl8-ova-md5]       | [ova][k1.22-u20-ova], [md5][k1.22-u20-ova-md5]       |
|            | v1.23              | [ova][k1.23-rl8-ova], [md5][k1.23-rl8-ova-md5]       | [ova][k1.23-u20-ova], [md5][k1.23-u20-ova-md5]       |
|            | v1.24              | [ova][k1.24-rl8-ova], [md5][k1.24-rl8-ova-md5]       | [ova][k1.24-u20-ova], [md5][k1.24-u20-ova-md5]       |
| XenServer  | v1.22              | [vhd][k1.22-rl8-vhd], [md5][k1.22-rl8-vhd-md5]       | [vhd][k1.22-u20-vhd], [md5][k1.22-u20-vhd-md5]       |
|            | v1.23              | [vhd][k1.23-rl8-vhd], [md5][k1.23-rl8-vhd-md5]       | [vhd][k1.23-u20-vhd], [md5][k1.23-u20-vhd-md5]       |
|            | v1.24              | [vhd][k1.24-rl8-vhd], [md5][k1.24-rl8-vhd-md5]       | [vhd][k1.24-u20-vhd], [md5][k1.24-u20-vhd-md5]       |

------
## Getting involved and contributing

Are you interested in contributing to cluster-api-provider-cloudstack? We, the
maintainers and community, would love your suggestions, contributions, and help!
Also, the maintainers can be contacted at any time to learn more about how to get
involved:

- via the [cluster-api-cloudstack channel on Kubernetes Slack][slack]

## Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct][code-of-conduct].

## Github issues

### Bugs

If you think you have found a bug please follow the instructions below.

- Please spend a small amount of time giving due diligence to the issue tracker. Your issue might be a duplicate.
- Get the logs from the cluster controllers. Please paste this into your issue.
- Open a [new issue][new_bug_issue].
- Remember that users might be searching for your issue in the future, so please give it a meaningful title to help others.
- Feel free to reach out to the Cluster API community on the [Kubernetes Slack][slack].

### Tracking new features

We also use the issue tracker to track features. If you have an idea for a feature, or think you can help Cluster API Provider CloudStack become even more awesome follow the steps below.

- Open a [new issue][new_feature_issue].
- Remember that users might be searching for your issue in the future, so please
  give it a meaningful title to help others.
- Clearly define the use case, using concrete examples.
- Some of our larger features will require some design. If you would like to
  include a technical design for your feature, please include it in the issue.
- After the new feature is well understood, and the design agreed upon, we can
  start coding the feature. We would love for you to code it. So please open
  up a **WIP** *(work in progress)* pull request, and happy coding.


## Our Contributors

Thank you to all contributors and a special thanks to our current maintainers & reviewers:

| Maintainers                                               | Reviewers                                              |
| --------------------------------------------------------- | ------------------------------------------------------ |
| [@rohityadavcloud](https://github.com/rohityadavcloud)    | [@rohityadavcloud](https://github.com/rohityadavcloud) |
| [@davidjumani](https://github.com/davidjumani)            | [@davidjumani](https://github.com/davidjumani)         |
| [@maxdrib](https://github.com/maxdrib)                    | [@maxdrib](https://github.com/maxdrib)                 |

All the CAPC contributors:

<p>
  <a href="https://sigs.k8s.io/cluster-api-provider-cloudstack/graphs/contributors">
    <img src="https://contrib.rocks/image?repo=aws/cluster-api-provider-cloudstack" />
  </a>
</p>
<!-- References -->

[capi-quick-start]: https://cluster-api.sigs.k8s.io/user/quick-start.html
[cluster_api]: https://sigs.k8s.io/cluster-api
[code-of-conduct]: https://kubernetes.io/community/code-of-conduct/
[getting_started]: https://cluster-api-cloudstack.sigs.k8s.io/getting-started.html
[image-builder]: https://github.com/kubernetes-sigs/image-builder/tree/master/images/capi
[kops]: https://github.com/kubernetes/kops
[kubicorn]: http://kubicorn.io/
[prebuilt-images]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/
[slack]: https://kubernetes.slack.com/messages/cluster-api-cloudstack
[new_bug_issue]: https://github.com/kubernetes-sigs/cluster-api-provider-cloudstack/issues/new
[new_feature_issue]: https://github.com/kubernetes-sigs/cluster-api-provider-cloudstack/issues/new

<!-- KVM -->
[k1.22-rl8-qcow2]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/rockylinux-8-kube-v1.22.6-kvm.qcow2.bz2
[k1.22-rl8-qcow2-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/rockylinux-8-kube-v1.22.6-kvm.qcow2.bz2.md5
[k1.23-rl8-qcow2]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/rockylinux-8-kube-v1.23.3-kvm.qcow2.bz2
[k1.23-rl8-qcow2-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/rockylinux-8-kube-v1.23.3-kvm.qcow2.bz2.md5
[k1.24-rl8-qcow2]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/rockylinux-8-kube-v1.24.7-kvm.qcow2.bz2
[k1.24-rl8-qcow2-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/rockylinux-8-kube-v1.24.7-kvm.qcow2.bz2.md5
[k1.22-u20-qcow2]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/ubuntu-2004-kube-v1.22.6-kvm.qcow2.bz2
[k1.22-u20-qcow2-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/ubuntu-2004-kube-v1.22.6-kvm.qcow2.bz2.md5
[k1.23-u20-qcow2]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/ubuntu-2004-kube-v1.23.3-kvm.qcow2.bz2
[k1.23-u20-qcow2-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/ubuntu-2004-kube-v1.23.3-kvm.qcow2.bz2.md5
[k1.24-u20-qcow2]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/ubuntu-2004-kube-v1.24.7-kvm.qcow2.bz2
[k1.24-u20-qcow2-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/kvm/ubuntu-2004-kube-v1.24.7-kvm.qcow2.bz2.md5

<!-- VMware -->
[k1.22-rl8-ova]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/rockylinux-8-kube-v1.22.6-vmware.ova
[k1.22-rl8-ova-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/rockylinux-8-kube-v1.22.6-vmware.ova.md5
[k1.23-rl8-ova]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/rockylinux-8-kube-v1.23.3-vmware.ova
[k1.23-rl8-ova-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/rockylinux-8-kube-v1.23.3-vmware.ova.md5
[k1.24-rl8-ova]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/rockylinux-8-kube-v1.24.7-vmware.ova
[k1.24-rl8-ova-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/rockylinux-8-kube-v1.24.7-vmware.ova.md5
[k1.22-u20-ova]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/ubuntu-2004-kube-v1.22.6-vmware.ova
[k1.22-u20-ova-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/ubuntu-2004-kube-v1.22.6-vmware.ova.md5
[k1.23-u20-ova]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/ubuntu-2004-kube-v1.23.3-vmware.ova
[k1.23-u20-ova-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/ubuntu-2004-kube-v1.23.3-vmware.ova.md5
[k1.24-u20-ova]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/ubuntu-2004-kube-v1.24.7-vmware.ova
[k1.24-u20-ova-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/vmware/ubuntu-2004-kube-v1.24.7-vmware.ova.md5

<!-- XenServer -->
[k1.22-rl8-vhd]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/rockylinux-8-kube-v1.22.6-xen.vhd.bz2
[k1.22-rl8-vhd-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/rockylinux-8-kube-v1.22.6-xen.vhd.bz2.md5
[k1.23-rl8-vhd]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/rockylinux-8-kube-v1.23.3-xen.vhd.bz2
[k1.23-rl8-vhd-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/rockylinux-8-kube-v1.23.3-xen.vhd.bz2.md5
[k1.24-rl8-vhd]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/rockylinux-8-kube-v1.24.7-xen.vhd.bz2
[k1.24-rl8-vhd-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/rockylinux-8-kube-v1.24.7-xen.vhd.bz2.md5
[k1.22-u20-vhd]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/ubuntu-2004-kube-v1.22.6-xen.vhd.bz2
[k1.22-u20-vhd-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/ubuntu-2004-kube-v1.22.6-xen.vhd.bz2.md5
[k1.23-u20-vhd]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/ubuntu-2004-kube-v1.23.3-xen.vhd.bz2
[k1.23-u20-vhd-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/ubuntu-2004-kube-v1.23.3-xen.vhd.bz2.md5
[k1.24-u20-vhd]:     http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/ubuntu-2004-kube-v1.24.7-xen.vhd.bz2
[k1.24-u20-vhd-md5]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/xen/ubuntu-2004-kube-v1.24.7-xen.vhd.bz2.md5
