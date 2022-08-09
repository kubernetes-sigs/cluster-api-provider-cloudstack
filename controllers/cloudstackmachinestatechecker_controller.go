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
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinestatecheckers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinestatecheckers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinestatecheckers/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines,verbs=get;list;watch;delete

// CloudStackMachineStateCheckerReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStack machine state checker reconciliation.
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

func (r *CloudStackMachineStateCheckerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return NewCSMachineStateCheckerReconciliationRunner().
		UsingBaseReconciler(r.ReconcilerBase).
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

	if err := r.CSClient.ResolveVMInstanceDetails(r.CSMachine); err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match found") {
		return r.ReturnWrappedError(err, "failed to resolve VM instance details")
	}

	const requeueAfter = 5 * time.Second

	csState := r.CSMachine.Status.InstanceState
	csTimeInState := r.CSMachine.Status.TimeSinceLastStateChange()

	if csState == "" || csTimeInState < 0 {
		// Wasn't able to get the CS state, and can't continue without it.
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	capiPhase := capiv1.MachinePhase(r.CAPIMachine.Status.Phase)
	capiRunning := capiPhase == capiv1.MachinePhaseRunning

	if capiPhase == capiv1.MachinePhaseDeleting || capiPhase == capiv1.MachinePhaseDeleted {
		// The machine is being deleted, so do nothing for now.
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	var (
		csRunning     bool
		csStopTimeout bool
		csOtherState  bool
		capiTimeout   bool
	)

	switch csState {
	case "Running":
		csRunning = true
		// capiTimeout indicates that a new VM is running, but it isn't reachable due to a network issue or a
		// misconfiguration.  When this happens, the machine should be deleted or the cluster won't recover.
		capiTimeout = !capiRunning && csTimeInState > 5*time.Minute
	case "Starting":
	case "Migrating":
		// Give CS time to do what it needs to do
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	case "Stopping":
	case "Stopped":
		// A machine could be stopped as part of a migration, in which case it needs more time.  If it's stopped for another
		// reason, it can be deleted.  If it's stopped for too long, assume it's not part of a migration and delete it.
		csStopTimeout = csTimeInState > 5*time.Minute
	default:
		csOtherState = true
	}

	if csRunning && capiRunning {
		r.ReconciliationSubject.Status.Ready = true
	} else if csOtherState || capiTimeout || csStopTimeout {
		r.Log.Info("CloudStack instance in bad state",
			"name", r.CSMachine.Name,
			"instance-id", r.CSMachine.Spec.InstanceID,
			"cs-state", csState,
			"cs-time-in-state", csTimeInState.String(),
			"capi-phase", capiPhase)

		if err := r.K8sClient.Delete(r.RequestCtx, r.CAPIMachine); err != nil {
			return r.ReturnWrappedError(err, "failed to delete CAPI machine")
		}
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
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
