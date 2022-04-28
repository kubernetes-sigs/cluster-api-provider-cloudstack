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
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinestatecheckers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinestatecheckers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinestatecheckers/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines,verbs=get;list;watch;delete

// CloudStackMachineStateCheckerReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackCluster reconciliation.
type CloudStackMachineStateCheckerReconciliationRunner struct {
	csCtrlrUtils.ReconciliationRunner
	MachineStateCheckers  *infrav1.CloudStackMachineStateCheckerList
	ReconciliationSubject *infrav1.CloudStackMachineStateChecker
	CAPIMachine           *capiv1.Machine
	CSMachine             *infrav1.CloudStackMachine
}

// CloudStackMachineStateCheckerReconciler reconciles a CloudStackMachineStateChecker object
type CloudStackMachineStateCheckerReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

// Initialize a new CloudStackMachineStateChecker reconciliation runner with concrete types and initialized member fields.
func NewCSMachineStateCheckerReconciliationRunner() *CloudStackMachineStateCheckerReconciliationRunner {
	// Set concrete type and init pointers.
	runner := &CloudStackMachineStateCheckerReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackMachineStateChecker{}}
	runner.CAPIMachine = &capiv1.Machine{}
	runner.CSMachine = &infrav1.CloudStackMachine{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	runner.ReconciliationRunner = csCtrlrUtils.NewRunner(runner, runner.ReconciliationSubject)
	return runner
}

func (reconciler *CloudStackMachineStateCheckerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return NewCSMachineStateCheckerReconciliationRunner().
		UsingBaseReconciler(reconciler.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunBaseReconciliationStages()
}

func (r *CloudStackMachineStateCheckerReconciliationRunner) Reconcile() (ctrl.Result, error) {
	if res, err := r.GetParent(r.ReconciliationSubject, r.CSMachine)(); r.ShouldReturn(res, err) {
		return res, err
	}
	if res, err := r.GetParent(r.CSMachine, r.CAPIMachine)(); r.ShouldReturn(res, err) {
		return res, err
	}

	if err := r.CS.ResolveVMInstanceDetails(r.CSMachine); err != nil {
		return r.ReturnWrappedError(err, "failed to resolve VM instance details")
	}
	if r.CSMachine.Status.InstanceState == "Running" {
		r.ReconciliationSubject.Status.Ready = true
	} else {
		if err := r.Client.Delete(r.RequestCtx, r.CAPIMachine); err != nil {
			return r.ReturnWrappedError(err, "failed to delete CAPI machine")
		}
	}

	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *CloudStackMachineStateCheckerReconciliationRunner) ReconcileDelete() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudStackMachineStateCheckerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackMachineStateChecker{}).
		Complete(r)
}
