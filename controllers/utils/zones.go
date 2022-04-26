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
		Spec:       infrav1.CloudStackZoneSpec{Name: zoneSpec.Name, ID: zoneSpec.ID, Network: zoneSpec.Network},
		Status:     infrav1.CloudStackZoneStatus{Ready: false},
	}
	return errors.Wrap(r.Client.Create(r.RequestCtx, csZone), "error encountered when creating CloudStackZone")
}

// CreateZones generates a CloudStackClusterZone CRD for each of the ReconcilationSubject's Zones.
// Returns a CloudStackReconcilerMethod to curry zoneSpecs.
func (runner *ReconciliationRunner) CreateZones(zoneSpecs []infrav1.Zone) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		for _, zone := range zoneSpecs {
			if err := runner.CreateZone(zone); err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
					return reconcile.Result{}, errors.Wrap(err, "error encountered when creating CloudStackZone")
				}
			}
		}
		return ctrl.Result{}, nil
	}
}

// GetZones gets CloudStackZones owned by a CloudStackCluster via an ownership label.
func (runner *ReconciliationRunner) GetZones(zones *infrav1.CloudStackZoneList) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		// labels := map[string]string{"OwnedBy": runner.Request.Name}
		capiClusterLabel := map[string]string{
			capiv1.ClusterLabelName: runner.CSCluster.GetLabels()[capiv1.ClusterLabelName]}
		if err := runner.Client.List(
			runner.RequestCtx,
			zones,
			client.InNamespace(runner.Request.Namespace),
			client.MatchingLabels(capiClusterLabel),
		); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to list zones")
		}
		return ctrl.Result{}, nil
	}
}
