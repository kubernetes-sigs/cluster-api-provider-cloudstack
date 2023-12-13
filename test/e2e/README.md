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

| Environment variable                        | Description                                                                      | Default Value               |
|---------------------------------------------|----------------------------------------------------------------------------------|-----------------------------|
| `CLOUDSTACK_ZONE_NAME`                      | The zone name                                                                    | `zone1`                     |
| `CLOUDSTACK_NETWORK_NAME`                   | The network name. If not exisiting an isolated network with the name is created. | `Shared1`                   |
| `CLUSTER_ENDPOINT_IP`                       | The cluster endpoint IP                                                          | `172.16.2.199`              |
| `CLUSTER_ENDPOINT_IP_2`                     | The cluster endpoint IP for a second cluster                                     | `172.16.2.199`              |
| `CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING` | The machine offering for the control plane VM instances                          | `Large Instance`            |
| `CLOUDSTACK_WORKER_MACHINE_OFFERING`        | The machine offering for the worker node VM instances                            | `Medium Instance`           |
| `CLOUDSTACK_TEMPLATE_NAME`                  | The machine template for both control plane and worke node VM instances          | `kube-v1.20.10/ubuntu-2004` |
| `CLOUDSTACK_SSH_KEY_NAME`                   | The name of SSH key added to the VM instances                                    | `CAPCKeyPair6`              |

Default values for these variables are defined in *config/cloudstack.yaml*.  This cloudstack.yaml can be completely overridden 
by providing make with the *fully qualified* path of another cloudstack.yaml via environment variable `E2E_CONFIG`

You will also have to define a k8s secret in a *cloud-config.yaml* file in the project root, containing a pointer to and 
credentials for the CloudStack backend that will be used for the test:

```
apiVersion: v1
kind: Secret
metadata:
  name: secret1
  namespace: default
type: Opaque
stringData:
  api-key: XXXX
  secret-key: XXXX
  api-url: http://1.2.3.4:8080/client/api
  verify-ssl: "false"
```
This will be applied to the kind cluster that hosts CAPI/CAPC for the test, allowing CAPC to access the cluster. 
The api-key and secret-key can be found or generated at Home > Accounts > admin > Users > admin of the ACS management UI. `verify-ssl` is an optional flag and its default value is true. CAPC skips verifying the host SSL certificates when the flag is set to false.

### Running the e2e tests

Run the following command to execute the CAPC e2e tests:

```shell
make run-e2e
```
This command runs all e2e test cases.

You can specify JOB environment variable which value is a regular expression to select test cases to execute. 
For example, 

```shell
JOB=PR-Blocking make run-e2e
```
This command runs the e2e tests that contains `PR-Blocking` in their spec names. 

### Debugging the e2e tests
The E2E tests can be debugged by attaching a debugger to the e2e process after it is launched (*i.e., make run-e2e*).
To facilitate this, the E2E tests can be run with environment variable PAUSE_FOR_DEBUGGER_ATTACH=true.
(This is only strictly needed when you want the debugger to break early in the test process, i.e., in SynchronizedBeforeSuite.
There's usually quite enough time to attach if you're not breaking until your actual test code runs.)

When this environment variable is set to *true* a 15s pause is inserted at the beginning of the test process
(i.e., in the SynchronizedBeforeSuite).  The workflow is:
- Launch the e2e test: *PAUSE_FOR_DEBUGGER_ATTACH=true JOB=MyTest make run-e2e*
- Wait for console message: *Pausing 15s so you have a chance to attach a debugger to this process...*
- Quickly attach your debugger to the e2e process (i.e., e2e.test)

## CI/CD for e2e testing

The community has set up a CI/CD pipeline using Jenkins for e2e testing.

### How it works

The CI/CD pipeline works as below

- User triggers e2e testing by a Github PR comment, only repository OWNERS and a list of engineers are allowed;
- A program monitors the PR comments of the Github repository, parses the comments and kick Jenkins jobs;
- Jenkins creates a Apache CloudStack with specific version and hypervisor type, if needed;
- Jenkins runs CAPC e2e testing with specific Kubernetes versions and images;
- Jenkins posts the results of e2e testing as a Github PR comment, with the link of test logs.

### How to use it

Similar as other prow commands(see [here](https://prow.k8s.io/command-help?repo=kubernetes-sigs%2Fcluster-api-provider-cloudstack)), the e2e testing can be triggered by PR comment `/run-e2e`:

```
Usage: /run-e2e [-k Kubernetes_Version] [-c CloudStack_Version] [-h Hypervisor] [-i Template/Image]
       [-f Kubernetes_Version_Upgrade_From] [-t Kubernetes_Version_Upgrade_To]
```

- Supported Kubernetes versions are: ['1.27.2', '1.26.5', '1.25.10', '1.24.14', '1.23.3', '1.22.6']. The default value is '1.27.2'.
- Supported CloudStack versions are: ['4.18', '4.17', '4.16']. If it is not set, an existing environment will be used.
- Supported hypervisors are: ['kvm', 'vmware', 'xen']. The default value is 'kvm'.
- Supported templates are: ['ubuntu-2004-kube', 'rockylinux-8-kube']. The default value is 'ubuntu-2004-kube'.
- By default it tests Kubernetes upgrade from version '1.26.5' to '1.27.2'.

### Examples

1. Examples of `/run-e2e` commands

```
/run-e2e
/run-e2e -k 1.27.2 -h kvm -i ubuntu-2004-kube
/run-e2e -k 1.27.2 -c 4.18 -h kvm -i ubuntu-2004-kube -f 1.26.5 -t 1.27.2
```

2. Example of test results
```
Test Results : (tid-126)
Environment: kvm Rocky8(x3), Advanced Networking with Management Server Rocky8
Kubernetes Version: v1.27.2
Kubernetes Version upgrade from: v1.26.5
Kubernetes Version upgrade to: v1.27.2
CloudStack Version: 4.18
Template: ubuntu-2004-kube
E2E Test Run Logs: https://github.com/blueorangutan/capc-prs/releases/download/capc-pr-ci-cd/capc-e2e-artifacts-pr277-sl-126.zip

[PASS] When testing Kubernetes version upgrades Should successfully upgrade kubernetes versions when there is a change in relevant fields
[PASS] When testing subdomain Should create a cluster in a subdomain
[PASS] When testing K8S conformance [Conformance] Should create a workload cluster and run kubetest
[PASS] When testing app deployment to the workload cluster with slow network [ToxiProxy] Should be able to download an HTML from the app deployed to the workload cluster
[PASS] When testing multiple CPs in a shared network with kubevip Should successfully create a cluster with multiple CPs in a shared network
[PASS] When testing resource cleanup Should create a new network when the specified network does not exist
[PASS] When testing app deployment to the workload cluster with network interruption [ToxiProxy] Should be able to create a cluster despite a network interruption during that process
[PASS] When testing node drain timeout A node should be forcefully removed if it cannot be drained in time
[PASS] When testing machine remediation Should replace a machine when it is destroyed
[PASS] When testing with custom disk offering Should successfully create a cluster with a custom disk offering
[PASS] When testing horizontal scale out/in [TC17][TC18][TC20][TC21] Should successfully scale machine replicas up and down horizontally
[PASS] When testing MachineDeployment rolling upgrades Should successfully upgrade Machines upon changes in relevant MachineDeployment fields
[PASS] with two clusters should successfully add and remove a second cluster without breaking the first cluster
[PASS] When testing with disk offering Should successfully create a cluster with disk offering
[PASS] When testing affinity group Should have host affinity group when affinity is pro
[PASS] When testing affinity group Should have host affinity group when affinity is anti
[PASS] When testing affinity group Should have host affinity group when affinity is soft-pro
[PASS] When testing affinity group Should have host affinity group when affinity is soft-anti
[PASS] When the specified resource does not exist Should fail due to the specified account is not found [TC4a]
[PASS] When the specified resource does not exist Should fail due to the specified domain is not found [TC4b]
[PASS] When the specified resource does not exist Should fail due to the specified control plane offering is not found [TC7]
[PASS] When the specified resource does not exist Should fail due to the specified template is not found [TC6]
[PASS] When the specified resource does not exist Should fail due to the specified zone is not found [TC3]
[PASS] When the specified resource does not exist Should fail due to the specified disk offering is not found
[PASS] When the specified resource does not exist Should fail due to the compute resources are not sufficient for the specified offering [TC8]
[PASS] When the specified resource does not exist Should fail due to the specified disk offer is not customized but the disk size is specified
[PASS] When the specified resource does not exist Should fail due to the specified disk offer is customized but the disk size is not specified
[PASS] When the specified resource does not exist Should fail due to the public IP can not be found
[PASS] When the specified resource does not exist When starting with a healthy cluster Should fail to upgrade worker machine due to insufficient compute resources
[PASS] When the specified resource does not exist When starting with a healthy cluster Should fail to upgrade control plane machine due to insufficient compute resources
[PASS] When testing app deployment to the workload cluster [TC1][PR-Blocking] Should be able to download an HTML from the app deployed to the workload cluster


Ran 28 of 29 Specs in 10458.173 seconds
SUCCESS! -- 28 Passed | 0 Failed | 0 Pending | 1 Skipped
PASS

```
