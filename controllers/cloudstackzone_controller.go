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

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/pkg/errors"
)

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackzones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackzones/status,verbs=create;get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackzones/finalizers,verbs=update

// CloudStackZoneReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStack zone reconciliation.
type CloudStackZoneReconciliationRunner struct {
	csCtrlrUtils.ReconciliationRunner
	Zones                 *infrav1.CloudStackZoneList
	ReconciliationSubject *infrav1.CloudStackZone
	IsoNet                *infrav1.CloudStackIsolatedNetwork
}

// CloudStackZoneReconciler reconciles a CloudStackZone object
type CloudStackZoneReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

// Initialize a new CloudStackZone reconciliation runner with concrete types and initialized member fields.
func NewCSZoneReconciliationRunner() *CloudStackZoneReconciliationRunner {
	// Set concrete type and init pointers.
	r := &CloudStackZoneReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackZone{}}
	r.Zones = &infrav1.CloudStackZoneList{}
	r.IsoNet = &infrav1.CloudStackIsolatedNetwork{}
	// Setup the base r. Initializes pointers and links reconciliation methods.
	r.ReconciliationRunner = csCtrlrUtils.NewRunner(r, r.ReconciliationSubject)
	return r
}

// Reconciler Reconcile adapts the runner to the runner to what k8s expects.
func (reconciler *CloudStackZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	return NewCSZoneReconciliationRunner().
		UsingBaseReconciler(reconciler.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunBaseReconciliationStages()
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *CloudStackZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackZone{}).
		Complete(reconciler)
}

// Reconcile attempts to move the state of CRs to the requested state.
func (r *CloudStackZoneReconciliationRunner) Reconcile() (retRes ctrl.Result, reterr error) {
	if res, err := r.RequeueIfMissingBaseCRs(); r.ShouldReturn(res, err) {
		return res, err
	}
	// Prevent premature deletion.
	controllerutil.AddFinalizer(r.ReconciliationSubject, infrav1.ZoneFinalizer)

	r.Log.V(1).Info("Reconciling CloudStackZone.", "zoneSpec", r.ReconciliationSubject.Spec)
	// Start by purely data fetching information about the zone and specified network.
	if err := r.CSUser.ResolveZone(r.ReconciliationSubject); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "resolving CloudStack zone information")
	}
	if err := r.CSUser.ResolveNetworkForZone(r.ReconciliationSubject); err != nil &&
		!csCtrlrUtils.ContainsNoMatchSubstring(err) {
		return ctrl.Result{}, errors.Wrap(err, "resolving Cloudstack network information")
	}

	// Address Isolated Networks.
	// Check if the passed network was an isolated network or the network was missing. In either case, create a
	// CloudStackIsolatedNetwork to manage the many intricacies and wait until CloudStackIsolatedNetwork is ready.
	if r.ReconciliationSubject.Spec.Network.ID == "" || r.ReconciliationSubject.Spec.Network.Type == infrav1.NetworkTypeIsolated {
		netName := r.ReconciliationSubject.Spec.Network.Name
		if res, err := r.GenerateIsolatedNetwork(netName)(); r.ShouldReturn(res, err) {
			return res, err
		} else if res, err := r.GetObjectByName(netName, r.IsoNet)(); r.ShouldReturn(res, err) {
			return res, err
		}
		if r.IsoNet.Name == "" {
			return r.RequeueWithMessage("Couldn't find isolated network.")
		}
		if !r.IsoNet.Status.Ready {
			return r.RequeueWithMessage("Isolated network dependency not ready.")
		}
	}
	r.ReconciliationSubject.Status.Ready = true
	return ctrl.Result{}, nil
}

// The CloudStackZone only fetches information, and in some cases creates CloudStackIsolatedNetwork CRDs.
// Deletion does not require cleanup, but should not occur until any owned CRDs are deleted.
func (r *CloudStackZoneReconciliationRunner) ReconcileDelete() (retRes ctrl.Result, reterr error) {
	r.Log.Info("Deleting CloudStackZone")
	// Address Isolated Networks.
	if r.ReconciliationSubject.Spec.Network.Type == infrav1.NetworkTypeIsolated {
		netName := r.ReconciliationSubject.Spec.Network.Name
		if res, err := r.GetObjectByName(netName, r.IsoNet)(); r.ShouldReturn(res, err) {
			return res, err
		}
		if r.IsoNet.Name != "" {
			if err := r.K8sClient.Delete(r.RequestCtx, r.IsoNet); err != nil {
				return ctrl.Result{}, err
			}
			return r.RequeueWithMessage("Child IsolatedNetwork still present, requeueing.")
		}
	}
	controllerutil.RemoveFinalizer(r.ReconciliationSubject, infrav1.ZoneFinalizer)
	return ctrl.Result{}, nil
}
