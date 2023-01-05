/*
Copyright 2023 The Kubernetes Authors.

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
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinestatecheckers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinestatecheckers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinestatecheckers/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines,verbs=get;list;watch;delete

// CloudStackMachineStateCheckerReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStack machine state checker reconciliation.
type CloudStackMachineStateCheckerReconciliationRunner struct {
	*csCtrlrUtils.ReconciliationRunner
	MachineStateCheckers  *infrav1.CloudStackMachineStateCheckerList
	ReconciliationSubject *infrav1.CloudStackMachineStateChecker
	FailureDomain         *infrav1.CloudStackFailureDomain
	CAPIMachine           *clusterv1.Machine
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
	runner.CAPIMachine = &clusterv1.Machine{}
	runner.CSMachine = &infrav1.CloudStackMachine{}
	runner.FailureDomain = &infrav1.CloudStackFailureDomain{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	runner.ReconciliationRunner = csCtrlrUtils.NewRunner(runner, runner.ReconciliationSubject, "CloudStackMachineStateChecker")
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
	return r.RunReconciliationStages(
		r.GetParent(r.ReconciliationSubject, r.CSMachine),
		r.GetParent(r.CSMachine, r.CAPIMachine),
		r.CheckPresent(map[string]client.Object{"CloudStackMachine": r.CSMachine, "Machine": r.CAPIMachine}),
		r.GetFailureDomainByName(func() string { return r.CSMachine.Spec.FailureDomainName }, r.FailureDomain),
		r.AsFailureDomainUser(&r.FailureDomain.Spec),
		func() (ctrl.Result, error) {
			if err := r.CSClient.ResolveVMInstanceDetails(r.CSMachine); err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), "no match found") {
					return r.ReturnWrappedError(err, "failed to resolve VM instance details")
				}
			}

			// capiTimeout indicates that a new VM is running, but it isn't reachable.
			// The cluster may not recover if the machine isn't replaced.
			csRunning := r.CSMachine.Status.InstanceState == "Running"
			csTimeInState := r.CSMachine.Status.TimeSinceLastStateChange()
			capiRunning := r.CAPIMachine.Status.Phase == "Running"
			capiTimeout := csRunning && !capiRunning && csTimeInState > 5*time.Minute

			if csRunning && capiRunning {
				r.ReconciliationSubject.Status.Ready = true
			} else if !csRunning || capiTimeout {
				r.Log.Info("CloudStack instance in bad state",
					"name", r.CSMachine.Name,
					"instance-id", r.CSMachine.Spec.InstanceID,
					"cs-state", r.CSMachine.Status.InstanceState,
					"cs-time-in-state", csTimeInState.String(),
					"capi-phase", r.CAPIMachine.Status.Phase)

				if err := r.K8sClient.Delete(r.RequestCtx, r.CAPIMachine); err != nil {
					return r.ReturnWrappedError(err, "failed to delete CAPI machine")
				}
			}

			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		})
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
