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

	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
)

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackzones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackzones/status,verbs=create;get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackzones/finalizers,verbs=update

// CloudStackZoneReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackCluster reconciliation.
type CloudStackZoneReconciliationRunner struct {
	csCtrlrUtils.ReconciliationRunner
	Zones                 infrav1.CloudStackZoneList
	ReconciliationSubject *infrav1.CloudStackZone
	CSUser                cloud.Client
}

// CloudStackZoneReconciler reconciles a CloudStackZone object
type CloudStackZoneReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

func (r *CloudStackZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	runner := &CloudStackZoneReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackZone{}}
	runner.CSCluster = &infrav1.CloudStackCluster{}
	runner.CAPICluster = &capiv1.Cluster{}
	return runner.
		UsingBaseReconciler(r.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunReconciliationStages(
			runner.SetupLogger,
			runner.SetReconciliationSubjectToConcreteSubject(runner.ReconciliationSubject),
			runner.GetReconciliationSubject,
			//runner.LogReconciliationSubject,
			runner.GetParent(runner.ReconciliationSubject, runner.CSCluster),
			runner.GetCAPICluster,
			runner.CheckIfPaused,
			runner.SetupPatcher,
			runner.IfDeletionTimestampIsZero(runner.Reconcile),
			runner.Else(runner.ReconcileDelete),
			runner.PatchChangesBackToAPI)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudStackZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackZone{}).
		Complete(r)
}

func (r *CloudStackZoneReconciliationRunner) Reconcile() (retRes ctrl.Result, reterr error) {

	r.Log.V(1).Info("Reconciling CloudStackCluster.", "clusterSpec", r.ReconciliationSubject.Spec)

	r.ReconciliationSubject.Status.Ready = true
	return ctrl.Result{}, nil
}

func (r *CloudStackZoneReconciliationRunner) ReconcileDelete() (retRes ctrl.Result, reterr error) {
	return ctrl.Result{}, nil
}
