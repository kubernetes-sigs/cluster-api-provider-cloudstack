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

	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pkg/errors"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
)

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackisolatednetworks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackisolatednetworks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackisolatednetworks/finalizers,verbs=update

// CloudStackIsoNetReconciler reconciles a CloudStackZone object
type CloudStackIsoNetReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

// CloudStackZoneReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStack isolated network reconciliation.
type CloudStackIsoNetReconciliationRunner struct {
	*csCtrlrUtils.ReconciliationRunner
	FailureDomain         *infrav1.CloudStackFailureDomain
	ReconciliationSubject *infrav1.CloudStackIsolatedNetwork
}

// Initialize a new CloudStackIsoNet reconciliation runner with concrete types and initialized member fields.
func NewCSIsoNetReconciliationRunner() *CloudStackIsoNetReconciliationRunner {
	// Set concrete type and init pointers.
	r := &CloudStackIsoNetReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackIsolatedNetwork{}}
	r.FailureDomain = &infrav1.CloudStackFailureDomain{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	r.ReconciliationRunner = csCtrlrUtils.NewRunner(r, r.ReconciliationSubject, "CloudStackIsolatedNetwork")
	return r
}

func (reconciler *CloudStackIsoNetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	r := NewCSIsoNetReconciliationRunner()
	r.UsingBaseReconciler(reconciler.ReconcilerBase).ForRequest(req).WithRequestCtx(ctx)
	r.WithAdditionalCommonStages(
		r.GetFailureDomainByName(func() string { return r.ReconciliationSubject.Spec.FailureDomainName }, r.FailureDomain),
		r.AsFailureDomainUser(r.RequestCtx, &r.FailureDomain.Spec),
	)
	return r.RunBaseReconciliationStages()
}

func (r *CloudStackIsoNetReconciliationRunner) Reconcile() (retRes ctrl.Result, retErr error) {
	controllerutil.AddFinalizer(r.ReconciliationSubject, infrav1.IsolatedNetworkFinalizer)

	// Setup isolated network, endpoint, egress, and load balancing.
	// Set endpoint of CloudStackCluster if it is not currently set. (uses patcher to do so)
	csClusterPatcher, err := patch.NewHelper(r.CSCluster, r.K8sClient)
	if err != nil {
		return r.ReturnWrappedError(retErr, "setting up CloudStackCluster patcher")
	}
	if r.FailureDomain.Spec.Zone.ID == "" {
		return r.RequeueWithMessage("Zone ID not resolved yet.")
	}
	if err := r.CSUser.GetOrCreateIsolatedNetwork(r.FailureDomain, r.ReconciliationSubject, r.CSCluster); err != nil {
		return ctrl.Result{}, err
	}
	// Tag the created network.
	if err := r.CSUser.AddClusterTag(cloud.ResourceTypeNetwork, r.ReconciliationSubject.Spec.ID, r.CSCluster); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "tagging network with id %s", r.ReconciliationSubject.Spec.ID)
	}
	if err := csClusterPatcher.Patch(r.RequestCtx, r.CSCluster); err != nil {
		return r.ReturnWrappedError(err, "patching endpoint update to CloudStackCluster")
	}

	r.ReconciliationSubject.Status.Ready = true
	return ctrl.Result{}, nil
}

func (r *CloudStackIsoNetReconciliationRunner) ReconcileDelete() (retRes ctrl.Result, retErr error) {
	r.Log.Info("Deleting IsolatedNetwork.")
	if err := r.CSUser.DisposeIsoNetResources(r.FailureDomain, r.ReconciliationSubject, r.CSCluster); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "no match found") {
			return ctrl.Result{}, err
		}
	}
	controllerutil.RemoveFinalizer(r.ReconciliationSubject, infrav1.IsolatedNetworkFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *CloudStackIsoNetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackIsolatedNetwork{}).
		Complete(reconciler)
}
