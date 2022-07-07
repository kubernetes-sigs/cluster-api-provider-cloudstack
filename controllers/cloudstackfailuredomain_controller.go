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

	"github.com/onsi/ginkgo/v2"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
)

// CloudStackFailureDomainReconciler is the k8s controller manager's interface to reconcile a CloudStackFailureDomain.
// This is primarily to adapt to k8s.
type CloudStackFailureDomainReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains/finalizers,verbs=update

// CloudStackFailureDomainReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackFailureDomains.
// The runner does the actual reconciliation.
type CloudStackFailureDomainReconciliationRunner struct {
	csCtrlrUtils.ReconciliationRunner
	ReconciliationSubject *infrav1.CloudStackFailureDomain
}

// Initialize a new CloudStackFailureDomain reconciliation runner with concrete types and initialized member fields.
func NewCSFailureDomainReconciliationRunner() *CloudStackFailureDomainReconciliationRunner {
	// Set concrete type and init pointers.
	r := &CloudStackFailureDomainReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackFailureDomain{}}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	r.ReconciliationRunner = csCtrlrUtils.NewRunner(r, r.ReconciliationSubject)

	return r
}

// Reconcile is the method k8s will call upon a reconciliation request.
func (reconciler *CloudStackFailureDomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	defer ginkgo.GinkgoRecover()
	return NewCSFailureDomainReconciliationRunner().
		UsingBaseReconciler(reconciler.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunBaseReconciliationStages()
}

// Reconcile on the ReconciliationRunner actually attempts to modify or create the reconciliation subject.
func (r *CloudStackFailureDomainReconciliationRunner) Reconcile() (retRes ctrl.Result, retErr error) {
	defer ginkgo.GinkgoRecover()
	endpointCredentials := &corev1.Secret{}
	ref := r.ReconciliationSubject.Spec.ACSEndpoint
	key := client.ObjectKey{Name: ref.Name, Namespace: ref.Namespace}
	if err := r.K8sClient.Get(r.RequestCtx, key, endpointCredentials); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "getting ACSEndpoint secret with ref: %v", ref)
	}
	// Prevent premature deletion.
	controllerutil.AddFinalizer(r.ReconciliationSubject, infrav1.)

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
		} else if res, err := r.GetObjectByName(r.IsoNetMetaName(netName), r.IsoNet)(); r.ShouldReturn(res, err) {
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
	r.ReconciliationSubject.Status.Ready = true
	return ctrl.Result{}, nil
}

// ReconcileDelete on the ReconciliationRunner actually attempts to delete the reconciliation subject.
func (r *CloudStackFailureDomainReconciliationRunner) ReconcileDelete() (retRes ctrl.Result, retErr error) {
	r.Log.Info("blah3")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *CloudStackFailureDomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	_, err := ctrl.NewControllerManagedBy(mgr).For(&infrav1.CloudStackFailureDomain{}).Build(reconciler)
	return err
}
