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

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackisolatednetworks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackisolatednetworks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackisolatednetworks/finalizers,verbs=update

// CloudStackIsolatedNetworkReconciler reconciles a CloudStackIsolatedNetwork object
type CloudStackIsolatedNetworkReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	CS     cloud.Client
}

type IsoNetContext struct {
	IsoNet *infrav1.CloudStackIsolatedNetwork
	Zone   *infrav1.CloudStackZone
}

// Initialize a new CloudStackIsoNet reconciliation runner with concrete types and initialized member fields.
func NewCSIsoNetReconciliationRunner() *CloudStackIsoNetReconciliationRunner {
	runner := &CloudStackIsoNetReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackIsolatedNetwork{}}
	runner.ReconciliationRunner = csCtrlrUtils.NewRunner(runner.ReconciliationSubject) // Initializes base pointers.
	runner.Zone = &infrav1.CloudStackZone{}
	return runner
}

// CloudStackZoneReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackCluster reconciliation.
type CloudStackIsoNetReconciliationRunner struct {
	csCtrlrUtils.ReconciliationRunner
	Zone                  *infrav1.CloudStackZone
	ReconciliationSubject *infrav1.CloudStackIsolatedNetwork
	CSUser                cloud.Client
}

// CloudStackIsoNetReconciler reconciles a CloudStackZone object
type CloudStackIsoNetReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

func (reconciler *CloudStackIsoNetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	r := NewCSIsoNetReconciliationRunner()
	return r.
		UsingBaseReconciler(reconciler.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunReconciliationStages(
			r.SetupLogger,
			r.GetReconciliationSubject,
			r.GetCAPICluster,
			r.GetCSCluster,
			r.GetParent(r.ReconciliationSubject, r.Zone),
			r.CheckIfPaused,
			r.SetupPatcher,
			r.IfDeletionTimestampIsZero(r.Reconcile),
			r.Else(r.ReconcileDelete),
			r.PatchChangesBackToAPI)
}

func (r *CloudStackIsoNetReconciliationRunner) Reconcile() (retRes ctrl.Result, retErr error) {
	// TODO: Add finalizers.
	if err := r.CS.GetOrCreateIsolatedNetwork(r.Zone, r.ReconciliationSubject); err != nil {
		return ctrl.Result{}, err
	}
	r.ReconciliationSubject.Spec.ControlPlaneEndpoint = r.CSCluster.Spec.ControlPlaneEndpoint

	// Setup isolated network endpoint, egress, and load balancing.
	// Set endpoint of CloudStackCluster if it is not currently set. (uses patcher to do so)
	if csClusterPatcher, err := patch.NewHelper(r.CSCluster, r.Client); err != nil {
		return r.ReturnWrappedError(retErr, "error encountered while setting up CloudStackCluster patcher")
	} else if err := r.CS.AssociatePublicIPAddress(r.Zone, r.ReconciliationSubject, r.CSCluster); err != nil {
		return r.ReturnWrappedError(err, "error encountered when associating isolated network public IP address")
	} else if r.ReconciliationSubject.Spec.ControlPlaneEndpoint.Host == "" {
		return ctrl.Result{}, errors.New("Endpoint not set.")
	} else if err := r.CS.GetOrCreateLoadBalancerRule(r.Zone, r.ReconciliationSubject, r.CSCluster); err != nil {
		return r.ReturnWrappedError(err, "error encountered when setting up load balancer for isolated network")
	} else {
		r.CSCluster.Spec.ControlPlaneEndpoint.Host = r.ReconciliationSubject.Spec.ControlPlaneEndpoint.Host
		if err := csClusterPatcher.Patch(r.RequestCtx, r.CSCluster); err != nil {
			return r.ReturnWrappedError(err, "error encountered when patching endpoint update to CloudStackCluster")
		}
	}
	fmt.Println("here")
	if r.ReconciliationSubject.Spec.ID != "" && r.ReconciliationSubject.Status.LBRuleID != "" && r.ReconciliationSubject.Status.PublicIPID != "" {
		r.ReconciliationSubject.Status.Ready = true
	} else {
		return r.RequeueWithMessage("network not yet ready")
	}
	return ctrl.Result{}, nil
}

func (r *CloudStackIsoNetReconciliationRunner) ReconcileDelete() (retRes ctrl.Result, retErr error) {
	// TODO: Remove finalizers
	// TODO: Cleanup any IsoNets tagged as created.
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudStackIsoNetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackIsolatedNetwork{}).
		Complete(r)
}
