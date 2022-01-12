# Testing

This document is to help developers understand how to test CAPC.

## Code Origin

The most of the code under test/e2e is from CAPD (Cluster API for Docker) e2e testing (https://github.com/kubernetes-sigs/cluster-api/tree/main/test/e2e)
The ACS specific things are under test/e2e/config and test/e2e/data/infrastructure-cloudstack. 

## e2e

This section describes how to run end-to-end (e2e) testing with CAPC.

### Requirements

* Admin access to a Apache CloudStack (ACS) server
* The testing must occur on a host that can access the ACS server
* Docker ([download](https://www.docker.com/get-started))
* Kind ([download](https://kind.sigs.k8s.io/docs/user/quick-start/#installing-with-a-package-manager))

### Environment variables

The first step to running the e2e tests is setting up the required environment variables:

| Environment variable                         | Description                                                                      | Default Value               |
| -------------------------------------------- | ---------------------------------------------------------------------------------| --------------------------- |
|  `CLOUDSTACK_ZONE_NAME`                      | The zone name                                                                    | `zone1`                     |
|  `CLOUDSTACK_NETWORK_NAME`                   | The network name. If not exisiting an isolated network with the name is created. | `Shared1`                   |
|  `CLUSTER_ENDPOINT_IP`                       | The cluster endpoint IP                                                          | `192.168.1.38`              |
|  `CLUSTER_ENDPOINT_PORT`                     | The cluster endpoint port                                                        | `6443`                      |
|  `CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING` | The machine offering for the control plane VM instances                          | `Large Instance`            |
|  `CLOUDSTACK_WORKER_MACHINE_OFFERING`        | The machine offering for the worker node VM instances                            | `Medium Instance`           |
|  `CLOUDSTACK_TEMPLATE_NAME`                  | The machine template for both control plane and worke node VM instances          | `kube-v1.20.10/ubuntu-2004` |
|  `CLOUDSTACK_SSH_KEY_NAME`                   | The name of SSH key added to the VM instances                                    | `CAPCKeyPair6`              |

You also have to export `CLOUDSTACK_B64ENCODED_SECRET` environment variable using this command `export CLOUDSTACK_B64ENCODED_SECRET=$(base64 -i cloud-config)` after creating `cloud-config` file with the following format.

```
[Global]
api-key    = XXXXX
secret-key = XXXXX
api-url    = http://192.168.1.96:8080/client/api
```

The api-key and secret-key can be found or generated at Home > Accounts > admin > Users > admin of the ACS management UI. 

### Running the e2e tests

Run the following command to execute the CAPC e2e tests:

```shell
make run-e2e
```
This command runs all e2e test cases except k8s conformance testing

```shell
make run-e2e-pr-blocking
```
This command runs the quick e2e tests for the sanity checks

```shell
make run-conformance
```
This command runs the k8s conformance testing
