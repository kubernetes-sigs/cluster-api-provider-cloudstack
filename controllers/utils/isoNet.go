package utils

import (
	"context"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
)

func (r *ReconciliationRunner) GenerateIsolatedNetwork(
	ctx context.Context, zone *infrav1.CloudStackZone, csCluster *infrav1.CloudStackCluster) error {

	csIsoNet := &infrav1.CloudStackIsolatedNetwork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zone.Spec.Name,
			Namespace: zone.Namespace,
			// Labels:      internal.ControlPlaneMachineLabelsForCluster(csCluster, csCluster.Name),
			Annotations: map[string]string{},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(zone, controlplanev1.GroupVersion.WithKind("CloudStackZone")),
				*metav1.NewControllerRef(csCluster, controlplanev1.GroupVersion.WithKind("CloudStackCluster")),
			},
		},
		Spec: infrav1.CloudStackIsolatedNetworkSpec{Name: zone.Spec.Network.Name},
	}

	if err := r.Client.Create(ctx, csIsoNet); err != nil {
		return errors.Wrap(err, "failed to create machine")
	}
	return nil
}
