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
	"strings"

	capcv1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiControlPlanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientPkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// getMachineSetFromCAPIMachine attempts to fetch a MachineSet from CAPI machine owner reference.
func getMachineSetFromCAPIMachine(
	ctx context.Context,
	client clientPkg.Client,
	capiMachine *capiv1.Machine,
) (*capiv1.MachineSet, error) {

	ref := GetManagementOwnerRef(capiMachine)
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if gv.Group == capiv1.GroupVersion.Group {
		key := clientPkg.ObjectKey{
			Namespace: capiMachine.Namespace,
			Name:      ref.Name,
		}

		machineSet := &capiv1.MachineSet{}
		if err := client.Get(ctx, key, machineSet); err != nil {
			return nil, err
		}

		return machineSet, nil
	}
	return nil, nil
}

// getKubeadmControlPlaneFromCAPIMachine attempts to fetch a KubeadmControlPlane from a CAPI machine owner reference.
func getKubeadmControlPlaneFromCAPIMachine(
	ctx context.Context,
	client clientPkg.Client,
	capiMachine *capiv1.Machine,
) (*capiControlPlanev1.KubeadmControlPlane, error) {

	ref := GetManagementOwnerRef(capiMachine)
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if gv.Group == capiControlPlanev1.GroupVersion.Group {
		key := clientPkg.ObjectKey{
			Namespace: capiMachine.Namespace,
			Name:      ref.Name,
		}

		controlPlane := &capiControlPlanev1.KubeadmControlPlane{}
		if err := client.Get(ctx, key, controlPlane); err != nil {
			return nil, err
		}

		return controlPlane, nil
	}
	return nil, nil
}

// IsOwnerDeleted returns a boolean if the owner of the CAPI machine has been deleted.
func IsOwnerDeleted(ctx context.Context, client clientPkg.Client, capiMachine *capiv1.Machine) (bool, error) {
	if util.IsControlPlaneMachine(capiMachine) {
		// The controlplane sticks around after deletion pending the deletion of its machiens.
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
func fetchOwnerRef(refList []meta.OwnerReference, kind string) *meta.OwnerReference {
	for _, ref := range refList {
		if ref.Kind == kind {
			return &ref
		}
	}
	return nil
}

// GetManagementOwnerRef returns the reference object pointing to the CAPI machine's manager.
func GetManagementOwnerRef(capiMachine *capiv1.Machine) *meta.OwnerReference {
	if util.IsControlPlaneMachine(capiMachine) {
		return fetchOwnerRef(capiMachine.OwnerReferences, "KubeadmControlPlane")
	}
	return fetchOwnerRef(capiMachine.OwnerReferences, "MachineSet")
}

// GetOwnerOfKind returns the Cluster object owning the current resource of passed kind.
func GetOwnerOfKind(ctx context.Context, c clientPkg.Client, owned client.Object, owner client.Object) error {
	gvks, _, err := c.Scheme().ObjectKinds(owner)
	if err != nil {
		return errors.Wrapf(err, "error encountered when finding owner kind for %s/%s", owned.GetName(), owned.GetNamespace())
	} else if len(gvks) != 1 {
		return errors.Errorf(
			"found more than one GVK for owner when finding owner kind for %s/%s", owned.GetName(), owned.GetNamespace())
	}
	kind := gvks[0].Kind
	for _, ref := range owned.GetOwnerReferences() {
		if ref.Kind != kind {
			continue
		}
		key := client.ObjectKey{Name: ref.Name, Namespace: owned.GetNamespace()}
		if err := c.Get(ctx, key, owner); err != nil {
			return errors.Wrapf(err, "error encountered when finding owner of kind %s %s/%s",
				owner.GetObjectKind().GroupVersionKind().Kind, owner.GetNamespace(), owner.GetName())
		}
		return nil
	}
	return errors.Errorf("couldn't find owner of kind % %s/%s",
		owner.GetObjectKind().GroupVersionKind().Kind, owner.GetNamespace(), owner.GetName())
}

// GetOwnerCloudStackCluster returns the Cluster object owning the current resource.
func GetOwnerCloudStackCluster(ctx context.Context, c clientPkg.Client, obj metav1.ObjectMeta) (*capcv1.CloudStackCluster, error) {

	for _, ref := range obj.OwnerReferences {
		if ref.Kind != "CloudStackCluster" {
			continue
		}
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if gv.Group == capcv1.GroupVersion.Group {
			csCluster := &capcv1.CloudStackCluster{}
			key := client.ObjectKey{Name: ref.Name, Namespace: obj.Namespace}
			if err := c.Get(ctx, key, csCluster); err != nil {
				return nil, err
			}
			return csCluster, nil
		}
	}
	return nil, nil
}

// GetOwnerZone returns the Cluster object owning the current resource.
func GetOwnerZone(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*capcv1.CloudStackZone, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind != "Cluster" {
			continue
		}
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if gv.Group == capcv1.GroupVersion.Group {
			zone := &capcv1.CloudStackZone{}
			key := client.ObjectKey{Name: ref.Name, Namespace: obj.Namespace}
			if err := c.Get(ctx, key, zone); err != nil {
				return nil, err
			}
			return zone, nil
		}
	}
	return nil, nil
}

// GetOwnerZone returns the Cluster object owning the current resource.
func GetZoneByID(ctx context.Context, c client.Client, obj metav1.ObjectMeta, zoneID string) (*capcv1.CloudStackZone, error) {
	zoneList := &capcv1.CloudStackZoneList{}
	opts := clientPkg.MatchingFields{"spec.id": zoneID}
	if err := c.List(ctx, zoneList, client.InNamespace(obj.Namespace), opts); err != nil {
		return nil, err
	}
	return &zoneList.Items[0], nil
}

func ContainsNoMatchSubstring(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "no match")
}

func ContainsAlreadyExistsSubstring(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}
