# Building Cluster API Provider for CloudStack

## Prerequisites:

1. Follow the instructions [here][capi-quick-start] to install the following tools:
    1. docker
    2. kind
    3. kubectl
    4. clusterctl [here][clusterctl-release]
    TODO : Update this once there is an official release

2. Create a local docker registry to save your docker image - otherwise, you need an image registry to push it somewhere else.

3. Download this [script][kind-capd] into your local and run it.
   This script will create a kind cluster and configure it to use local docker registry:
    ```
    wget https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/main/hack/kind-install-for-capd.sh
    chmod +x ./kind-install-for-capd.sh
    ./kind-install-for-capd.sh
    ```
4. Set up Apache CloudStack credentials
    1. Create a file named `cloud-config` in the repo's root directory, substituting in your own environment's values
        ```
        [Global]
        api-url = <cloudstackApiUrl>
        api-key = <cloudstackApiKey>
        secret-key = <cloudstackSecretKey>
        ```

    2. Run the following command to save the above Apache CloudStack connection info into an environment variable, to be used by `./config/default/credentials.yaml` and ultimately the generated `infrastructure-components.yaml`, where it gets passed to CAPC:
        ```
        export CLOUDSTACK_B64ENCODED_SECRET=$(base64 -w0 -i cloud-config)
        ```

5. Set the IMG environment variable so that the Makefile knows where to push docker image (if building your own)
   1. `export IMG=localhost:5000/cluster-api-provider-capc`
   2. `make docker-build`
   3. `make docker-push`

6. Set the source image location so that the CAPC deployment manifest files have the right image path in them in `config/default/manager_image_patch.yaml`

7. Generate the CAPC manifests (if building your own) into `$RELEASE_DIR`

   `make build` will generate and copy `infrastructure-components.yaml` and metadata.yaml files to `$RELEASE_DIR`, which is `./out` by default. You may want to override the default value with `export RELEASE_DIR=${HOME}/.cluster-api/overrides/infrastructure-cloudstack/<VERSION>/` to deploy the generated manifests for use by clusterctl before running `make build`.


8. Generate clusterctl config file so that clusterctl knows how to provision the Apache CloudStack cluster, referencing whatever you set for `$RELEASE_DIR` from above for the url:
    ```
    cat << EOF > ~/.cluster-api/cloudstack.yaml
    providers:
    - name: "cloudstack"
      type: "InfrastructureProvider"
      url: ${HOME}/.cluster-api/overrides/infrastructure-cloudstack/<VERSION>/infrastructure-components.yaml
    EOF
    ```

9. Assure that the required Apache CloudStack resources have been created: zone, pod cluster, and k8s-compatible template, compute offerings defined (2GB+ of RAM for control plane offering with 2vCPU).


## Deploying Custom Builds

### Initialize the management cluster

Run the following command to turn your cluster into a management cluster and load the Apache CloudStack components into it.

    clusterctl init --infrastructure cloudstack --config ~/.cluster-api/cloudstack.yaml

{{#include ./common.md:common-development}}


<!-- References -->

[capi-quick-start]: https://cluster-api.sigs.k8s.io/user/quick-start.html
[capi-w-tilt]: https://cluster-api.sigs.k8s.io/developer/tilt.html
[clusterctl-release]: https://github.com/kubernetes-sigs/cluster-api/releases
[kind-capd]: https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/main/hack/kind-install-for-capd.sh
[tilt]: https://tilt.dev/
