# Developing Cluster API Provider CloudStack with Tilt

This document describes how to use kind and [Tilt][tilt] for a simplified workflow that offers easy deployments and rapid iterative builds.

Before the next steps, make sure [initial setup for development environment][initial-setup] steps are complete.

Also, visit the [Cluster API documentation on Tilt][cluster_api_tilt] for more information on how to set up your development environment.

## Create a kind cluster

First, make sure you have a kind cluster and that your `KUBECONFIG` is set up correctly:

``` bash
kind create cluster
```

This local cluster will be running all the cluster api controllers and become the management cluster which then can be used to spin up workload clusters on Apache CloudStack.

## Get the source

Get the source for core cluster-api for development with Tilt along with cluster-api-provider-cloudstack.

```bash
cd "$(go env GOPATH)"
mkdir sigs.k8s.io
cd sigs.k8s.io/
git clone git@github.com:kubernetes-sigs/cluster-api.git
cd cluster-api
git fetch upstream
```

## Create a tilt-settings.json file

Next, create a `tilt-settings.json` file and place it in your local copy of `cluster-api`. Here is an example:

**Example `tilt-settings.json` for CAPC clusters:**

```json
{
    "default_registry": "gcr.io/your-project-name-here",
    "provider_repos": ["../cluster-api-provider-cloudstack"],
    "enable_providers": ["kubeadm-bootstrap", "kubeadm-control-plane", "cloudstack"],
    "kustomize_substitutions": {
        "CLOUDSTACK_B64ENCODED_CREDENTIALS": "RANDOM_STRING==",
    }
}
```

### Debugging

If you would like to debug CAPC (or core CAPI / another provider) you can run the provider with delve. This will then allow you to attach to delve and debug.

To do this you need to use the **debug** configuration in **tilt-settings.json**. Full details of the options can be seen [here](https://cluster-api.sigs.k8s.io/developer/tilt.html).

An example **tilt-settings.json**:

```json
{
    "default_registry": "gcr.io/your-project-name-here",
    "provider_repos": ["../cluster-api-provider-cloudstack"],
    "enable_providers": ["kubeadm-bootstrap", "kubeadm-control-plane", "cloudstack"],
    "kustomize_substitutions": {
        "CLOUDSTACK_B64ENCODED_CREDENTIALS": "RANDOM_STRING==",
    },
    "debug": {
    "CloudStack": {
        "continue": true,
        "port": 30000
    }
  }
}
```

Once you have run tilt (see section below) you will be able to connect to the running instance of delve.

For vscode, you can use the a launch configuration like this:

```json
    {
        "name": "Connect to CAPC",
        "type": "go",
        "request": "attach",
        "mode": "remote",
        "remotePath": "",
        "port": 30000,
        "host": "127.0.0.1",
        "showLog": true,
        "trace": "log",
        "logOutput": "rpc"
    }
```

For GoLand/IntelliJ add a new run configuration following [these instructions](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html#step-3-create-the-remote-run-debug-configuration-on-the-client-computer).

Or you could use delve directly from the CLI using a command similar to this:

```bash
dlv-dap connect 127.0.0.1:3000
```

## Run Tilt!

To launch your development environment, run:

``` bash
tilt up
```

kind cluster becomes a management cluster after this point, check the pods running on the kind cluster `kubectl get pods -A`.

## Create Workload Cluster

{{#include ./common.md:common-development}}

## Clean up

Before deleting the kind cluster, make sure you delete all the workload clusters.

```bash
kubectl delete cluster <clustername>
tilt up (ctrl-c)
kind delete cluster
```

<!-- References -->
[tilt]: https://tilt.dev
[cluster_api_tilt]: https://cluster-api.sigs.k8s.io/developer/tilt.html
[initial-setup]: ./index.html#initial-setup-for-development-environment
