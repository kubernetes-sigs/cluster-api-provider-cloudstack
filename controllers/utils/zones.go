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
	"strings"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"

	"github.com/pkg/errors"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CreateZone generates a specified CloudStackZone CRD owned by the ReconcilationSubject.
func (r *ReconciliationRunner) CreateZone(zoneSpec infrav1.Zone) error {
	csZone := &infrav1.CloudStackZone{
		ObjectMeta: r.NewChildObjectMeta(zoneSpec.Name),
		Spec:       infrav1.CloudStackZoneSpec(zoneSpec),
		Status:     infrav1.CloudStackZoneStatus{Ready: false},
	}
	return errors.Wrap(r.Client.Create(r.RequestCtx, csZone), "error encountered when creating CloudStackZone")
}

// CreateZones generates a CloudStackClusterZone CRD for each of the ReconcilationSubject's Zones.
// Returns a CloudStackReconcilerMethod to curry zoneSpecs.
func (r *ReconciliationRunner) CreateZones(zoneSpecs []infrav1.Zone) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		for _, zone := range zoneSpecs {
			if err := r.CreateZone(zone); err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
					return reconcile.Result{}, errors.Wrap(err, "error encountered when creating CloudStackZone")
				}
			}
		}
		return ctrl.Result{}, nil
	}
}

// GetZones gets CloudStackZones owned by a CloudStackCluster via an ownership label.
func (r *ReconciliationRunner) GetZones(zones *infrav1.CloudStackZoneList) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		// labels := map[string]string{"OwnedBy": r.Request.Name}
		capiClusterLabel := map[string]string{
			capiv1.ClusterLabelName: r.CSCluster.GetLabels()[capiv1.ClusterLabelName]}
		if err := r.Client.List(
			r.RequestCtx,
			zones,
			client.InNamespace(r.Request.Namespace),
			client.MatchingLabels(capiClusterLabel),
		); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to list zones")
		}
		return ctrl.Result{}, nil
	}
}
