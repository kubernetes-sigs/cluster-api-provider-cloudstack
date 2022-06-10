# Getting Started

### Prerequisites

1. Follow the instructions [here][capi-quick-start] to install the following tools:
    - [kubectl][kubectl-install]
    - [clusterctl][clusterctl-install]

    TODO : Update this once there is an official release

    Optional if you do not have an existing Kubernetes cluster
    - [kind][kind-install]
    - [Docker][docker-install]

2. Set up Apache CloudStack credentials
    - Create a file named `cloud-config` in the repo's root directory, substituting in your own environment's values
        ```
        [Global]
        api-url = <cloudstackApiUrl>
        api-key = <cloudstackApiKey>
        secret-key = <cloudstackSecretKey>
        ```

    - Run the following command to save the above Apache CloudStack connection info into an environment variable, to be used by clusterctl, where it gets passed to CAPC:
        ```
        export CLOUDSTACK_B64ENCODED_SECRET=$(base64 -w0 -i cloud-config)
        ```

3. Register the capi-compatible templates in your Apache CloudStack installation.
    - Prebuilt images can be found [here][prebuilt-images]
    - To build a compatible image see [CloudStack CAPI Images][cloudstack-capi-images]

4. Create a management cluster. This can either be :
    - An existing Kubernetes cluster : For production use-cases a "real" Kubernetes cluster should be used with appropriate backup and DR policies and procedures in place. The Kubernetes cluster must be at least v1.19.1.

    - A local cluster created with `kind`, for non production use
        ```
        kind create cluster
        ```


### Initialize the management cluster

Run the following command to turn your cluster into a management cluster and load the Apache CloudStack components into it.

    clusterctl init --infrastructure cloudstack

<!-- References -->

[capi-quick-start]: https://cluster-api.sigs.k8s.io/user/quick-start.html
[clusterctl-install]: https://cluster-api.sigs.k8s.io/user/quick-start.html#install-clusterctl
[cloudstack-capi-images]: https://image-builder.sigs.k8s.io/capi/providers/cloudstack.html
[docker-install]: https://www.docker.com/
[kind-install]: https://kind.sigs.k8s.io/
[kubectl-install]: [https://kubernetes.io/docs/tasks/tools/install-kubectl/]
[prebuilt-images]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/


{{#include ./development/common.md:common-development}}
