# Releasing Cluster API Provider for CloudStack

## Prerequisites:

1. Please install the following tools :
    - [go][go]
    - [Docker][docker-install]
    - [gcloud][gcloud-install]

2. Set up and log in to gcloud by running `gcloud init`
   > In order to publish any artifact, you need to be a member of the [k8s-infra-staging-capi-cloudstack][k8s-infra-staging-capi-cloudstack] group

## Creating only the docker container

If you would just like to build only the docker container and upload it rather than creating a release, you can run the following command :
```
REGISTRY=<your custom registry> IMAGE_NAME=<your custom image name> TAG=<your custom tag> make docker-build
```
It defaults to `gcr.io/k8s-staging-capi-cloudstack/capi-cloudstack-controller:dev`


## Creating a new release

Run the following command to create the new release artifacts as well as publish them to the upstream gcr.io repository :
```
RELEASE_TAG=<your custom tag> make release-staging
```

Create the necessary release in GitHub along with the following artifacts ( found in the `out` directory after running the previous command )
- metadata.yaml
- infrastructure-components.yaml
- cluster-template*.yaml



[docker-install]: https://www.docker.com/
[go]: https://golang.org/doc/install
[gcloud-install]: https://cloud.google.com/sdk/docs/install
[k8s-infra-staging-capi-cloudstack]: https://github.com/kubernetes/k8s.io/blob/main/groups/sig-cluster-lifecycle/groups.yaml#L106