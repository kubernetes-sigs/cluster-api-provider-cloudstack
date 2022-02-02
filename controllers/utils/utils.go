/*
Copyright 2022.

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

	"github.com/pkg/errors"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	clientPkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetMachineSet attempts to fetch a MachineSet from CAPI machine owner reference.
func GetMachineSet(ctx context.Context, client clientPkg.Client, capiMachine *capiv1.Machine) (*capiv1.MachineSet, error) {
	for _, ref := range capiMachine.OwnerReferences {
		if ref.Kind != "MachineSet" {
			continue
		}
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if gv.Group == capiv1.GroupVersion.Group {
			machineSet := &capiv1.MachineSet{}
			key := clientPkg.ObjectKey{
				Namespace: capiMachine.Namespace,
				Name:      ref.Name,
			}

			if err := client.Get(ctx, key, machineSet); err != nil {
				return nil, errors.Wrapf(err, "failed to get MachineSet/%s", ref.Name)
			}

			return machineSet, nil
		}
	}
	return nil, nil
}

// GetKubeadmControlPlaneFromCAPIMachine attempts to fetch a KubeadmControlPlane from a CAPI machine owner reference.
func GetKubeadmControlPlaneFromCAPIMachine(
	ctx context.Context,
	client clientPkg.Client,
	capiMachine *capiv1.Machine,
) (*controlplanev1.KubeadmControlPlane, error) {

	for _, ref := range capiMachine.OwnerReferences {
		if ref.Kind != "KubeadmControlPlane" {
			continue
		}
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if gv.Group == controlplanev1.GroupVersion.Group {
			controlPlane := &controlplanev1.KubeadmControlPlane{}
			key := clientPkg.ObjectKey{
				Namespace: capiMachine.Namespace,
				Name:      ref.Name,
			}

			if err := client.Get(ctx, key, controlPlane); err != nil {
				return nil, errors.Wrapf(err, "failed to get KubeadmControlPlane/%s", ref.Name)
			}

			return controlPlane, nil
		}
	}
	return nil, nil
}

// IsOwnerDeleted returns a boolean if the owner of the CAPI machine has been deleted.
func IsOwnerDeleted(ctx context.Context, client clientPkg.Client, capiMachine *capiv1.Machine) bool {

	if !util.IsControlPlaneMachine(capiMachine) {
		if md, _ := GetKubeadmControlPlaneFromCAPIMachine(ctx, client, capiMachine); md == nil {
			return true
		}
	} else {
		if md, _ := GetMachineSet(ctx, client, capiMachine); md == nil {
			return true
		}
	}
	return false
}

// fetchRef simply searches a list of OwnerReference objects for a given kind.
func fetchRef(reflist []meta.OwnerReference, kind string) *meta.OwnerReference {
	for _, ref := range reflist {
		if ref.Kind == kind {
			fmt.Println("here")
			return &ref
		}
	}
	fmt.Println("here")
	fmt.Println(reflist)
	return nil
}

// GetManagementOwnerRef returns the reference object pointing to the CAPI machine's manager.
func GetManagementOwnerRef(capiMachine *capiv1.Machine) *meta.OwnerReference {
	if !util.IsControlPlaneMachine(capiMachine) {
		return fetchRef(capiMachine.OwnerReferences, "MachineSet")
	} else {
		return fetchRef(capiMachine.OwnerReferences, "KubeadmControlPlane")
	}
}
