package utils

import (
	"context"
	"strings"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ZoneUsingReconciler is an interface required of a reconciler to be able to CRUD zones.
type ZoneUsingReconciler interface {
	Subject() client.Object
	Base() CloudStackBaseReconciler
	ZoneSpecs() []infrav1.Zone
	SettableZones() *infrav1.CloudStackClusterList // For retrieving zone information.
}

// CreateZone generates a specified CloudStackZone CRD owned by the ReconcilationSubject.
func CreateZone(ctx context.Context, reconciler ZoneUsingReconciler, zoneSpec infrav1.Zone) error {
	ownerKind := reconciler.Subject().GetObjectKind().GroupVersionKind().Kind
	csZone := &infrav1.CloudStackZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:        strings.ToLower(zoneSpec.Name),
			Namespace:   reconciler.Subject().GetNamespace(),
			Labels:      map[string]string{"OwnedBy": reconciler.Subject().GetName()},
			Annotations: map[string]string{},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(reconciler.Subject(), controlplanev1.GroupVersion.WithKind(ownerKind)),
			},
		},
		Spec: infrav1.CloudStackZoneSpec{Name: zoneSpec.Name},
	}

	if err := reconciler.Base().Client.Create(ctx, csZone); err != nil {
		return errors.Wrap(err, "failed to create zone")
	}
	return nil
}

// CreateZones generates a CloudStackClusterZone CRD for each of the ReconcilationSubject's Zones.
// Assumes ReconcilationSubject has a Spec.Zones field.
func CreateZones(ctx context.Context, reconciler ZoneUsingReconciler) (ctrl.Result, error) {
	for _, zone := range reconciler.ZoneSpecs() {
		if err := CreateZone(ctx, reconciler, zone); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
				return reconcile.Result{}, errors.Wrap(err, "error encountered when creating CloudStackZone")
			}
		}
	}
	return ctrl.Result{}, nil
}

// GetZones gets CloudStackZones owned by a CloudStackCluster via an ownership label.
func GetZones(ctx context.Context, reconciler ZoneUsingReconciler) (ctrl.Result, error) {
	labels := map[string]string{"OwnedBy": reconciler.Subject().GetName()}
	if err := reconciler.Base().Client.List(
		ctx,
		reconciler.SettableZones(),
		client.InNamespace(reconciler.Subject().GetNamespace()),
		client.MatchingLabels(labels),
	); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to list zones")
	}
	return ctrl.Result{}, nil
}
