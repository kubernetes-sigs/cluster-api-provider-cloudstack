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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
)

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackaffinitygroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackaffinitygroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackaffinitygroups/finalizers,verbs=update
// Need to watch machine templates for creation of an affinity group.
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplate,verbs=get;list;watch;

// CloudStackAGReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStack affinity group reconciliation.
type CloudStackAGReconciliationRunner struct {
	csCtrlrUtils.ReconciliationRunner
	ReconciliationSubject *infrav1.CloudStackAffinityGroup
}

// CloudStackAGReconciler is the base reconciler to adapt to k8s.
type CloudStackAffinityGroupReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

// Initialize a new CloudStackAffinityGroup reconciliation runner with concrete types and initialized member fields.
func NewCSAGReconciliationRunner() *CloudStackAGReconciliationRunner {
	// Set concrete type and init pointers.
	r := &CloudStackAGReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackAffinityGroup{}}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	r.ReconciliationRunner = csCtrlrUtils.NewRunner(r, r.ReconciliationSubject, "CloudStackAffinityGroup")
	return r
}

func (reconciler *CloudStackAffinityGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return NewCSAGReconciliationRunner().
		UsingBaseReconciler(reconciler.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunBaseReconciliationStages()
}

func (r *CloudStackAGReconciliationRunner) Reconcile() (ctrl.Result, error) {
	controllerutil.AddFinalizer(r.ReconciliationSubject, infrav1.AffinityGroupFinalizer)
	affinityGroup := &cloud.AffinityGroup{Name: r.ReconciliationSubject.Spec.Name, Type: r.ReconciliationSubject.Spec.Type}
	if err := r.CSUser.GetOrCreateAffinityGroup(affinityGroup); err != nil {
		return ctrl.Result{}, err
	}
	r.ReconciliationSubject.Spec.ID = affinityGroup.ID
	r.ReconciliationSubject.Status.Ready = true
	return ctrl.Result{}, nil
}

func (r *CloudStackAGReconciliationRunner) ReconcileDelete() (ctrl.Result, error) {
	group := &cloud.AffinityGroup{Name: r.ReconciliationSubject.Name}
	_ = r.CSUser.FetchAffinityGroup(group)
	if group.ID == "" { // Affinity group not found, must have been deleted.
		return ctrl.Result{}, nil
	}
	if err := r.CSUser.DeleteAffinityGroup(group); err != nil {
		return ctrl.Result{}, err
	}
	controllerutil.RemoveFinalizer(r.ReconciliationSubject, infrav1.AffinityGroupFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *CloudStackAffinityGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackAffinityGroup{}).
		Complete(reconciler)
}
