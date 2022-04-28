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

	ctrl "sigs.k8s.io/controller-runtime"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
)

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinehealthcheckers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinehealthcheckers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinehealthcheckers/finalizers,verbs=update

// CloudStackMachineHealthCheckerReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackCluster reconciliation.
type CloudStackMachineHealthCheckerReconciliationRunner struct {
	csCtrlrUtils.ReconciliationRunner
	MachineHealthCheckers *infrav1.CloudStackMachineHealthCheckerList
	ReconciliationSubject *infrav1.CloudStackMachineHealthChecker
	CSUser                cloud.Client
	CSInstance            *infrav1.CloudStackMachine
}

// CloudStackMachineHealthCheckerReconciler reconciles a CloudStackMachineHealthChecker object
type CloudStackMachineHealthCheckerReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

// Initialize a new CloudStackMachineHealthChecker reconciliation runner with concrete types and initialized member fields.
func NewCSMachineHCReconciliationRunner() *CloudStackMachineHealthCheckerReconciliationRunner {
	// Set concrete type and init pointers.
	runner := &CloudStackMachineHealthCheckerReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackMachineHealthChecker{}}
	runner.CSInstance = &infrav1.CloudStackMachine{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	runner.ReconciliationRunner = csCtrlrUtils.NewRunner(runner, runner.ReconciliationSubject)
	return runner
}

func (reconciler *CloudStackMachineHealthCheckerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return NewCSMachineHCReconciliationRunner().
		UsingBaseReconciler(reconciler.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunBaseReconciliationStages()
}

func (r *CloudStackMachineHealthCheckerReconciliationRunner) Reconcile() (ctrl.Result, error) {
	if res, err := r.GetParent(r.ReconciliationSubject, r.CSInstance)(); r.ShouldReturn(res, err) {
		return res, err
	}
	r.Log.Info(r.ReconciliationSubject.Spec.InstanceID)
	return ctrl.Result{}, nil
}

func (r *CloudStackMachineHealthCheckerReconciliationRunner) ReconcileDelete() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudStackMachineHealthCheckerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackMachineHealthChecker{}).
		Complete(r)
}
