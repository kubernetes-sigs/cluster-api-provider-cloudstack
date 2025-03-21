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

package controllers

import (
	"context"
	"fmt"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
)

const CksClusterFinalizer = "ckscluster.infrastructure.cluster.x-k8s.io"

// RBAC permissions for CloudStackCluster.
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters/status,verbs=create;get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters/finalizers,verbs=update

// CksClusterReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackClusters.
// The runner does the actual reconciliation.
type CksClusterReconciliationRunner struct {
	*csCtrlrUtils.ReconciliationRunner
	FailureDomains        *infrav1.CloudStackFailureDomainList
	ReconciliationSubject *infrav1.CloudStackCluster
}

// CksClusterReconciler is the k8s controller manager's interface to reconcile a CloudStackCluster.
type CksClusterReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

// Initialize a new CloudStackCluster reconciliation runner with concrete types and initialized member fields.
func NewCksClusterReconciliationRunner() *CksClusterReconciliationRunner {
	// Set concrete type and init pointers.
	runner := &CksClusterReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackCluster{}}
	runner.FailureDomains = &infrav1.CloudStackFailureDomainList{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	runner.ReconciliationRunner = csCtrlrUtils.NewRunner(runner, runner.ReconciliationSubject, "CKSClusterController")
	runner.CSCluster = runner.ReconciliationSubject
	return runner
}

// Reconcile is the method k8s will call upon a reconciliation request.
func (reconciler *CksClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	r := NewCksClusterReconciliationRunner()
	r.UsingBaseReconciler(reconciler.ReconcilerBase).ForRequest(req).WithRequestCtx(ctx)
	r.WithAdditionalCommonStages(r.GetFailureDomains(r.FailureDomains))
	return r.RunBaseReconciliationStages()
}

// Reconcile actually reconciles the CloudStackCluster.
func (r *CksClusterReconciliationRunner) Reconcile() (res ctrl.Result, reterr error) {
	if r.CSCluster.Spec.SyncWithACS == nil || !*r.CSCluster.Spec.SyncWithACS || len(r.FailureDomains.Items) == 0 {
		return ctrl.Result{}, nil
	}
	// Prevent premature deletion.
	controllerutil.AddFinalizer(r.ReconciliationSubject, CksClusterFinalizer)

	res, err := r.AsFailureDomainUser(&r.FailureDomains.Items[0].Spec)()
	if r.ShouldReturn(res, err) {
		return res, err
	}

	r.Log.Info("Creating entry with CKS")
	err = r.CSUser.GetOrCreateCksCluster(r.CAPICluster, r.ReconciliationSubject, &r.FailureDomains.Items[0].Spec)
	if err != nil {
		return r.RequeueWithMessage(fmt.Sprintf("Failed creating ExternalManaged CKS cluster on CloudStack. error: %s", err.Error()))
	}
	return ctrl.Result{}, nil
}

// ReconcileDelete cleans up resources used by the cluster and finally removes the CloudStackCluster's finalizers.
func (r *CksClusterReconciliationRunner) ReconcileDelete() (ctrl.Result, error) {
	if r.ReconciliationSubject.Status.CloudStackClusterID != "" {
		if len(r.FailureDomains.Items) == 0 {
			return ctrl.Result{}, fmt.Errorf("no failure domains found")
		}
		res, err := r.AsFailureDomainUser(&r.FailureDomains.Items[0].Spec)()
		if r.ShouldReturn(res, err) {
			return res, err
		}
		err = r.CSUser.DeleteCksCluster(r.ReconciliationSubject)
		if err != nil && !strings.Contains(err.Error(), " not found") {
			return r.RequeueWithMessage(fmt.Sprintf("Deleting cks cluster on CloudStack failed. error: %s", err.Error()))
		}
	}
	controllerutil.RemoveFinalizer(r.ReconciliationSubject, CksClusterFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *CksClusterReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		Named("cks-cluster-controller").
		For(&infrav1.CloudStackCluster{}).
		Complete(reconciler)
}
