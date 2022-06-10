# Developer Guide

## Initial setup for development environment

### Prerequisites

Please install the following tools :
1. [go][go]
    - Get the latest patch version for go v1.17.
2. [kind][kind]
    - `GO111MODULE="on" go get sigs.k8s.io/kind@v0.12.0`.
3. [kustomize][kustomize]
    - [install instructions](https://kubectl.docs.kubernetes.io/installation/kustomize/)
4. [envsubst][envsubst]
5. make

### Get the source

Fork the [cluster-api-provider-cloudstack repo](https://github.com/kubernetes-sigs/cluster-api-provider-cloudstack):

```bash
cd "$(go env GOPATH)"/src
mkdir sigs.k8s.io
cd sigs.k8s.io/
git clone git@github.com:<GITHUB USERNAME>/cluster-api-provider-cloudstack.git
cd cluster-api-provider-cloudstack
git remote add upstream git@github.com:kubernetes-sigs/cluster-api-provider-cloudstack.git
git fetch upstream
```


### Setup the CloudStack Environment

1. Set up Apache CloudStack credentials
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

2. Register the capi-compatible templates in your Apache CloudStack installation.
    - Prebuilt images can be found [here][prebuilt-images]
    - To build a compatible image see [CloudStack CAPI Images][cloudstack-capi-images]


## Running local management cluster for development

Before the next steps, make sure [initial setup for development environment][initial-setup] steps are complete.


There are two ways to build Apache CloudStack manager from local cluster-api-provider-cloudstack source and run it in local kind cluster:

### Option 1: Setting up Development Environment with Tilt

[Tilt][tilt] is a tool for quickly building, pushing, and reloading Docker containers as part of a Kubernetes deployment.
Many of the Cluster API engineers use it for quick iteration. Please see our [Tilt instructions][tilt-instructions] to get started.


### Option 2: The Old-fashioned way

Running cluster-api and cluster-api-provider-cloudstack controllers in a kind cluster:

1. Create a local kind cluster
   - `kind create cluster`
2. Install core cluster-api controllers (the version must match the cluster-api version in [go.mod][go.mod])
   - `clusterctl init`
3. Release manifests under `./out` directory
   - `RELEASE_TAG="e2e" make release-manifests`
4. Apply the manifests
   - `kubectl apply -f ./out/infrastructure.yaml`

[cloudstack-capi-images]: https://image-builder.sigs.k8s.io/capi/providers/cloudstack.html
[go]: https://golang.org/doc/install
[go.mod]: https://github.com/kubernetes-sigs/cluster-api-provider-cloudstack/blob/master/go.mod
[initial-setup]: ../index.html#initial-setup
[kind]: https://sigs.k8s.io/kind
[kustomize]: https://github.com/kubernetes-sigs/kustomize
[envsubst]: https://github.com/a8m/envsubst
[prebuilt-images]: http://packages.shapeblue.com/cluster-api-provider-cloudstack/images/
[tilt]: https://tilt.dev
[tilt-instructions]: ./tilt.md
