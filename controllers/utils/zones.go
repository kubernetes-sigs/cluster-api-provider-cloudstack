package utils

import (
	"strings"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CreateZone generates a specified CloudStackZone CRD owned by the ReconcilationSubject.
func (runner *ReconciliationRunner) CreateZone(zoneSpec infrav1.Zone) error {
	ownerKind := runner.ReconciliationSubject.GetObjectKind().GroupVersionKind().Kind
	csZone := &infrav1.CloudStackZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:        strings.ToLower(zoneSpec.Name),
			Namespace:   runner.Request.Namespace,
			Labels:      map[string]string{"OwnedBy": runner.Request.Name},
			Annotations: map[string]string{},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(runner.ReconciliationSubject, controlplanev1.GroupVersion.WithKind(ownerKind)),
			},
		},
		Spec:   infrav1.CloudStackZoneSpec{Name: zoneSpec.Name},
		Status: infrav1.CloudStackZoneStatus{Ready: false},
	}

	if err := runner.Client.Create(runner.RequestCtx, csZone); err != nil {
		return errors.Wrap(err, "failed to create zone")
	}
	return nil
}

func (runner *ReconciliationRunner) CreateZones2(zoneSpecs []infrav1.Zone) (ctrl.Result, error) {
	return func() (ctrl.Result, error) {
		return ctrl.Result{}, nil
	}()
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
func (runner *ReconciliationRunner) GetZones(zones *infrav1.CloudStackZoneList) (ctrl.Result, error) {
	labels := map[string]string{"OwnedBy": runner.Request.Name}

	if err := runner.Client.List(
		runner.RequestCtx,
		zones,
		client.InNamespace(runner.Request.Namespace),
		client.MatchingLabels(labels),
	); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to list zones")
	}
	return ctrl.Result{}, nil
}
