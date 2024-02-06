/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiControlPlanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
)

// getMachineSetFromCAPIMachine attempts to fetch a MachineSet from CAPI machine owner reference.
func getMachineSetFromCAPIMachine(
	ctx context.Context,
	c client.Client,
	capiMachine *clusterv1.Machine,
) (*clusterv1.MachineSet, error) {

	ref := GetManagementOwnerRef(capiMachine)
	if ref == nil {
		return nil, errors.New("management owner not found")
	}
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if gv.Group == clusterv1.GroupVersion.Group {
		key := client.ObjectKey{
			Namespace: capiMachine.Namespace,
			Name:      ref.Name,
		}

		machineSet := &clusterv1.MachineSet{}
		if err := c.Get(ctx, key, machineSet); err != nil {
			return nil, err
		}

		return machineSet, nil
	}
	return nil, nil
}

// getKubeadmControlPlaneFromCAPIMachine attempts to fetch a KubeadmControlPlane from a CAPI machine owner reference.
func getKubeadmControlPlaneFromCAPIMachine(
	ctx context.Context,
	c client.Client,
	capiMachine *clusterv1.Machine,
) (*capiControlPlanev1.KubeadmControlPlane, error) {

	ref := GetManagementOwnerRef(capiMachine)
	if ref == nil {
		return nil, errors.New("management owner not found")
	}
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if gv.Group == capiControlPlanev1.GroupVersion.Group {
		key := client.ObjectKey{
			Namespace: capiMachine.Namespace,
			Name:      ref.Name,
		}

		controlPlane := &capiControlPlanev1.KubeadmControlPlane{}
		if err := c.Get(ctx, key, controlPlane); err != nil {
			return nil, err
		}

		return controlPlane, nil
	}
	return nil, nil
}

// IsOwnerDeleted returns a boolean if the owner of the CAPI machine has been deleted.
func IsOwnerDeleted(ctx context.Context, client client.Client, capiMachine *clusterv1.Machine) (bool, error) {
	if util.IsControlPlaneMachine(capiMachine) {
		// The controlplane sticks around after deletion pending the deletion of its machines.
		// As such, need to check the deletion timestamp thereof.
		if cp, err := getKubeadmControlPlaneFromCAPIMachine(ctx, client, capiMachine); cp != nil && cp.DeletionTimestamp == nil {
			return false, nil
		} else if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			return false, err
		}
	} else {
		// The machineset is deleted immediately, regardless of machine ownership.
		// It is sufficient to check for its existence.
		if ms, err := getMachineSetFromCAPIMachine(ctx, client, capiMachine); ms != nil {
			return false, nil
		} else if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			return false, err
		}
	}
	return true, nil
}

// fetchOwnerRef simply searches a list of OwnerReference objects for a given kind.
func fetchOwnerRef(refList []metav1.OwnerReference, kind string) *metav1.OwnerReference {
	for _, ref := range refList {
		if ref.Kind == kind {
			return &ref
		}
	}
	return nil
}

// GetManagementOwnerRef returns the reference object pointing to the CAPI machine's manager.
func GetManagementOwnerRef(capiMachine *clusterv1.Machine) *metav1.OwnerReference {
	if util.IsControlPlaneMachine(capiMachine) {
		return fetchOwnerRef(capiMachine.OwnerReferences, "KubeadmControlPlane")
	} else if ref := fetchOwnerRef(capiMachine.OwnerReferences, "EtcdadmCluster"); ref != nil {
		return ref
	}
	return fetchOwnerRef(capiMachine.OwnerReferences, "MachineSet")
}

// GetOwnerOfKind returns the Cluster object owning the current resource of passed kind.
func GetOwnerOfKind(ctx context.Context, c client.Client, owned client.Object, owner client.Object) error {
	gvks, _, err := c.Scheme().ObjectKinds(owner)
	if err != nil {
		return errors.Wrapf(err, "finding owner kind for %s/%s", owned.GetName(), owned.GetNamespace())
	} else if len(gvks) != 1 {
		return errors.Errorf(
			"found more than one GVK for owner when finding owner kind for %s/%s", owned.GetName(), owned.GetNamespace())
	}
	gvk := gvks[0]

	for _, ref := range owned.GetOwnerReferences() {
		if ref.Kind != gvk.Kind {
			continue
		}
		key := client.ObjectKey{Name: ref.Name, Namespace: owned.GetNamespace()}
		if err := c.Get(ctx, key, owner); err != nil {
			return errors.Wrapf(err, "finding owner of kind %s in namespace %s", gvk.Kind, owned.GetNamespace())
		}
		return nil
	}

	return errors.Errorf("couldn't find owner of kind %s in namespace %s", gvk.Kind, owned.GetNamespace())
}

func ContainsNoMatchSubstring(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "no match")
}

func ContainsAlreadyExistsSubstring(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}

// WithClusterSuffix appends a hyphen and the cluster name to a name if not already present.
func WithClusterSuffix(name string, clusterName string) string {
	newName := name
	if !strings.HasSuffix(name, "-"+clusterName) { // Add cluster name suffix if missing.
		newName = name + "-" + clusterName
	}
	return newName
}

// GetOwnerClusterName returns the Cluster name of the cluster owning the current resource.
func GetOwnerClusterName(obj metav1.ObjectMeta) (string, error) {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Kind != "Cluster" {
			continue
		}
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return "", errors.WithStack(err)
		}
		if gv.Group == clusterv1.GroupVersion.Group {
			return ref.Name, nil
		}
	}
	return "", errors.New("failed to get owner Cluster name")
}

// CloudStackClusterToCloudStackMachines is a handler.ToRequestsFunc to be used to enqeue requests for reconciliation
// of CloudStackMachines.
func CloudStackClusterToCloudStackMachines(c client.Client, obj runtime.Object, scheme *runtime.Scheme, log logr.Logger) (handler.MapFunc, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find GVK for CloudStackMachine")
	}

	return func(ctx context.Context, o client.Object) []ctrl.Request {
		csCluster, ok := o.(*infrav1.CloudStackCluster)
		if !ok {
			log.Error(fmt.Errorf("expected a CloudStackCluster but got a %T", o), "Error in CloudStackClusterToCloudStackMachines")
		}

		log = log.WithValues("objectMapper", "cloudstackClusterToCloudStackMachine", "cluster", klog.KRef(csCluster.Namespace, csCluster.Name))

		// Don't handle deleted CloudStackClusters
		if !csCluster.ObjectMeta.DeletionTimestamp.IsZero() {
			log.V(4).Info("CloudStackCluster has a deletion timestamp, skipping mapping.")
			return nil
		}

		clusterName, err := GetOwnerClusterName(csCluster.ObjectMeta)
		if err != nil {
			log.Error(err, "Failed to get owning cluster, skipping mapping.")
			return nil
		}

		machineList := &clusterv1.MachineList{}
		machineList.SetGroupVersionKind(gvk)
		// list all of the requested objects within the cluster namespace with the cluster name label
		if err := c.List(ctx, machineList, client.InNamespace(csCluster.Namespace), client.MatchingLabels{clusterv1.ClusterNameLabel: clusterName}); err != nil {
			return nil
		}

		mapFunc := util.MachineToInfrastructureMapFunc(gvk)
		var results []ctrl.Request
		for _, machine := range machineList.Items {
			m := machine
			csMachines := mapFunc(ctx, &m)
			results = append(results, csMachines...)
		}

		return results
	}, nil
}
