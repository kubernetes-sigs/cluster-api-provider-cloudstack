# Upgrading to CAPC v0.7 (Cluster API v1beta2 contract)

CAPC v0.7 implements the Cluster API v1beta2 contract and is built against Cluster API v1.13.
This page covers what changes for existing users and the recommended upgrade path for a management cluster
running CAPC v0.6 with Cluster API v1.9 or older.

## What changes

- **New API version.** CAPC adds `infrastructure.cluster.x-k8s.io/v1beta4`, which is now the storage version.
  `v1beta1`, `v1beta2` and `v1beta3` are still served and are converted to and from `v1beta4` by the conversion
  webhook, so existing manifests and objects keep working.
- **Cluster API core types move to v1beta2.** The cluster templates now use `cluster.x-k8s.io/v1beta2`,
  `bootstrap.cluster.x-k8s.io/v1beta2` and `controlplane.cluster.x-k8s.io/v1beta2`. Templates you maintain
  yourself need the same changes, for example list-form `kubeletExtraArgs`, `apiGroup` based object references,
  the `KubeadmControlPlane` `machineTemplate.spec` layout and the new `MachineHealthCheck` fields. See the Cluster
  API [v1.10 to v1.11 provider migration guide][capi-migration-v1.11] for the full list.
- **Status fields follow the v1beta2 contract.** `CloudStackCluster` and `CloudStackMachine` report
  `status.initialization.provisioned`, and `CloudStackCluster` reports `status.failureDomains` as a list.
  `status.ready` is still set, so tooling that reads it keeps working for now.
- **An empty control plane endpoint is still accepted.** You can leave `spec.controlPlaneEndpoint.host` empty
  (for example by setting `CLUSTER_ENDPOINT_IP=""` on an isolated network) and CAPC picks a free public IP,
  as before. The variable has no default in the templates, so it must be set to an empty string. If it is
  unset, `clusterctl generate cluster` fails. On `v1beta4` the defaulting webhook removes the empty `host`
  and a zero `port` before the object is validated. Any non-zero port you set is kept.

## Recommended upgrade path

Cluster API recommends that every management cluster runs v1.10 before moving to v1.11 or later. Cluster API
v1.10 added stricter validation on API fields. Running it first lets you find and fix existing objects with
invalid field values before v1.11+ starts converting v1beta1 objects to v1beta2. See the Cluster API
[recommendation for everyone][capi-v1.10-step] for details.

Do not jump straight from Cluster API v1.9 or older to v1.13.

### 1. Move to the latest v1beta1 contract releases

Use a `clusterctl` v1.10.x binary and upgrade every provider to the newest release that still implements the
v1beta1 contract. For CAPC this is the latest v0.6.x release.

```shell
clusterctl version   # must report v1.10.x
clusterctl upgrade plan
clusterctl upgrade apply --contract v1beta1
```

Wait until every provider deployment is available and the Cluster API controllers have reconciled your
clusters. `clusterctl` labels every component it installs with `cluster.x-k8s.io/provider`, so this lists the
core, bootstrap, control plane and CloudStack deployments:

```shell
kubectl get deployments -A -l cluster.x-k8s.io/provider
clusterctl describe cluster <cluster-name> -n <namespace>
```

Then check the controller logs and object conditions for validation errors. Fix any object that Cluster API
v1.10 reports as invalid before you continue, because those objects can fail to convert in the next step.

### 2. Move to the v1beta2 contract

Switch to a `clusterctl` v1.13.x binary and upgrade to Cluster API v1.13 and CAPC v0.7.

```shell
clusterctl version   # must report v1.13.x
clusterctl upgrade plan
clusterctl upgrade apply --contract v1beta2
```

`clusterctl upgrade` migrates stored CAPC objects to the new `v1beta4` storage version.

### 3. Update your templates

Regenerate cluster manifests with `clusterctl generate cluster` against CAPC v0.7, or update your own templates
to the `v1beta4` CAPC types and the Cluster API `v1beta2` core types listed above. Older CAPC API versions keep
working through conversion, but `v1beta4` is the version new features land in.

<!-- References -->

[capi-v1.10-step]: https://cluster-api.sigs.k8s.io/developer/providers/migrations/v1.10-to-v1.11#recommendation-for-everyone
[capi-migration-v1.11]: https://cluster-api.sigs.k8s.io/developer/providers/migrations/v1.10-to-v1.11
