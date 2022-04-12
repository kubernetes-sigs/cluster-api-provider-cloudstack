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
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
)

// CloudStackZoneReconciler reconciles a CloudStackZone object
type CloudStackZoneReconciler struct {
	utils.CloudStackBaseReconciler
	ReconciliationSubject infrav1.CloudStackZone
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackzones,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackzones/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackzones/finalizers,verbs=update
func (r *CloudStackZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	r.CloudStackBaseReconciler.UsingConcreteSubject(&r.ReconciliationSubject)

	return r.RunWith(ctx, req,
		r.SetupLogger,
		r.GetBaseCRDs,
		r.FetchReconcilationSubject,
		r.CheckIfPaused,
		r.reconcileDelete,
		r.reconcile,
		r.PatchChangesBackToAPI,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudStackZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackZone{}).
		Complete(r)
}

func (r *CloudStackZoneReconciler) reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, reterr error) {

	r.ReconciliationSubject.Status.Ready = true
	fmt.Println(r.ReconciliationSubject)
	return ctrl.Result{}, nil
}

func (r *CloudStackZoneReconciler) reconcileDelete(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, reterr error) {
	return ctrl.Result{}, nil
}

// func (r *CloudStackZoneReconciler) generateIsolatedNetwork(
// 	ctx context.Context, zone *infrav1.CloudStackZone, csCluster *infrav1.CloudStackCluster) error {

// 	// csIsoNet := &infrav1.CloudStackIsolatedNetwork{
// 	// 	ObjectMeta: metav1.ObjectMeta{
// 	// 		Name:      zone.Spec.Name,
// 	// 		Namespace: zone.Namespace,
// 	// 		// Labels:      internal.ControlPlaneMachineLabelsForCluster(csCluster, csCluster.Name),
// 	// 		Annotations: map[string]string{},
// 	// 		OwnerReferences: []metav1.OwnerReference{
// 	// 			*metav1.NewControllerRef(zone, controlplanev1.GroupVersion.WithKind("CloudStackZone")),
// 	// 			*metav1.NewControllerRef(csCluster, controlplanev1.GroupVersion.WithKind("CloudStackCluster")),
// 	// 		},
// 	// 	},
// 	// 	Spec: infrav1.CloudStackIsolatedNetworkSpec{Name: zone.Spec.Network.Name},
// 	// }

// 	// if err := r.Client.Create(ctx, csIsoNet); err != nil {
// 	// 	return errors.Wrap(err, "failed to create machine")
// 	// }
// 	return nil
// }
