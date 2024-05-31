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

	ctrl "sigs.k8s.io/controller-runtime"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
)

// RBAC permissions for CloudStackCluster.
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=get;list;watch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines/status,verbs=get

// CksMachineReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackClusters.
// The runner does the actual reconciliation.
type CksMachineReconciliationRunner struct {
	*csCtrlrUtils.ReconciliationRunner
	FailureDomain         *infrav1.CloudStackFailureDomain
	ReconciliationSubject *infrav1.CloudStackMachine
}

// CksMachineReconciler is the k8s controller manager's interface to reconcile a CloudStackCluster.
// This is primarily to adapt to k8s.
type CksMachineReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

// Initialize a new CloudStackCluster reconciliation runner with concrete types and initialized member fields.
func NewCksMachineReconciliationRunner() *CksMachineReconciliationRunner {
	// Set concrete type and init pointers.
	runner := &CksMachineReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackMachine{}}
	runner.FailureDomain = &infrav1.CloudStackFailureDomain{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	runner.ReconciliationRunner = csCtrlrUtils.NewRunner(runner, runner.ReconciliationSubject, "CKSMachineController")
	return runner
}

// Reconcile is the method k8s will call upon a reconciliation request.
func (reconciler *CksMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	r := NewCksMachineReconciliationRunner()
	r.UsingBaseReconciler(reconciler.ReconcilerBase).ForRequest(req).WithRequestCtx(ctx)
	r.WithAdditionalCommonStages(
		r.GetFailureDomainByName(func() string { return r.ReconciliationSubject.Spec.FailureDomainName }, r.FailureDomain),
		r.AsFailureDomainUser(&r.FailureDomain.Spec))
	return r.RunBaseReconciliationStages()
}

// Reconcile actually reconciles the CloudStackCluster.
func (r *CksMachineReconciliationRunner) Reconcile() (res ctrl.Result, reterr error) {
	if !r.CSCluster.Spec.SyncWithACS {
		return ctrl.Result{}, nil
	}
	if r.CSCluster.Status.CloudStackClusterID == "" {
		return r.RequeueWithMessage("CloudStackClusterID is not set")
	}

	if r.ReconciliationSubject.Spec.InstanceID == nil || *r.ReconciliationSubject.Spec.InstanceID == "" {
		return r.RequeueWithMessage("InstanceID is not set")
	}

	res, err := r.AsFailureDomainUser(&r.FailureDomain.Spec)()
	if r.ShouldReturn(res, err) {
		return res, err
	}
	r.Log.Info("Assigning VM to CKS")
	err = r.CSUser.AddVMToCksCluster(r.CSCluster, r.ReconciliationSubject)
	if err != nil {
		return r.RequeueWithMessage(fmt.Sprintf("Adding VM to CloudStack CKS cluster failed. error: %s", err.Error()))
	}
	return ctrl.Result{}, nil

}

// ReconcileDelete cleans up resources used by the cluster and finally removes the CloudStackCluster's finalizers.
func (r *CksMachineReconciliationRunner) ReconcileDelete() (ctrl.Result, error) {
	if r.ReconciliationSubject.Spec.InstanceID != nil && *r.ReconciliationSubject.Spec.InstanceID != "" {
		err := r.CSUser.RemoveVMFromCksCluster(r.CSCluster, r.ReconciliationSubject)
		if err != nil {
			return r.RequeueWithMessage(fmt.Sprintf("Removing VM from CloudStack CKS cluster failed. error: %s", err.Error()))
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *CksMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackMachine{}).
		Complete(reconciler)
}
