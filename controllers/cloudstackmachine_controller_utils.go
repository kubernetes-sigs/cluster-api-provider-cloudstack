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

package controllers

import (
	"fmt"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// ResolveManagedAffinity assumes the machine's affinity is managed by CAPC, computes what it should be, and then either
// creates or fetches said group.
func (r *CloudStackMachineReconciler) ResolveManagedAffinity(
	log logr.Logger,
	csMachine *infrav1.CloudStackMachine,
	capiMachine *capiv1.Machine,
	csCluster *infrav1.CloudStackCluster) (*cloud.AffinityGroup, error) {

	ownerRef := csCtrlrUtils.GetManagementOwnerRef(capiMachine)
	if ownerRef == nil {
		return nil, errors.Errorf("Could not find management owner reference for %s/%s", csMachine.Namespace, csMachine.Name)
	}
	name := fmt.Sprintf("Affinity-%s", ownerRef.UID)
	ag := cloud.AffinityGroup{Name: name}
	if csMachine.Spec.Affinity == "anti" {
		ag.Type = cloud.AntiAffinityGroupType
	} else {
		ag.Type = cloud.AffinityGroupType
	}
	err := r.CS.GetOrCreateAffinityGroup(csCluster, &ag)
	return &ag, err
}

// RemoveManagedAffinity considers a machine's affinity management strategy and removes the created affinity group
// if it exists.
func (r *CloudStackMachineReconciler) RemoveManagedAffinity(
	log logr.Logger,
	capiMachine *capiv1.Machine,
	csMachine *infrav1.CloudStackMachine,
) error {

	ownerRef := csCtrlrUtils.GetManagementOwnerRef(capiMachine)
	if ownerRef == nil {
		return errors.Errorf("Could not find management owner reference for %s/%s", csMachine.Namespace, csMachine.Name)
	}
	name := fmt.Sprintf("Affinity-%s", ownerRef.UID)
	group := &cloud.AffinityGroup{Name: name}
	_ = r.CS.FetchAffinityGroup(group)
	if group.Id == "" { // Affinity group not found, must have been deleted.
		return nil
	}

	log.Info(fmt.Sprintf("Deleting affinity group '%s'", name))

	return r.CS.DeleteAffinityGroup(group)
}
