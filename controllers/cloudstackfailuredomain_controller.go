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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
	*csCtrlrUtils.ReconciliationRunner
	ReconciliationSubject *infrav1.CloudStackFailureDomain
	IsoNet                *infrav1.CloudStackIsolatedNetwork
}

// Initialize a new CloudStackFailureDomain reconciliation runner with concrete types and initialized member fields.
func NewCSFailureDomainReconciliationRunner() *CloudStackFailureDomainReconciliationRunner {
	// Set concrete type and init pointers.
	r := &CloudStackFailureDomainReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackFailureDomain{}}
	r.IsoNet = &infrav1.CloudStackIsolatedNetwork{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	r.ReconciliationRunner = csCtrlrUtils.NewRunner(r, r.ReconciliationSubject, "CloudStackFailureDomain")

	return r
}

// Reconcile is the method k8s will call upon a reconciliation request.
func (reconciler *CloudStackFailureDomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	return NewCSFailureDomainReconciliationRunner().
		UsingBaseReconciler(reconciler.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunBaseReconciliationStages()
}

// Reconcile on the ReconciliationRunner actually attempts to modify or create the reconciliation subject.
func (r *CloudStackFailureDomainReconciliationRunner) Reconcile() (retRes ctrl.Result, retErr error) {
	res, err := r.AsFailureDomainUser(&r.ReconciliationSubject.Spec)()
	if r.ShouldReturn(res, err) {
		return res, err
	}
	// Prevent premature deletion.
	controllerutil.AddFinalizer(r.ReconciliationSubject, infrav1.FailureDomainFinalizer)

	if err := r.SetOwnerReferenceToEndpointSecret(); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Couldn't set owner reference for endpoint secret")
	}

	// Start by purely data fetching information about the zone and specified network.
	if err := r.CSUser.ResolveZone(&r.ReconciliationSubject.Spec.Zone); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "resolving CloudStack zone information")
	}
	if err := r.CSUser.ResolveNetworkForZone(&r.ReconciliationSubject.Spec.Zone); err != nil &&
		!csCtrlrUtils.ContainsNoMatchSubstring(err) {
		return ctrl.Result{}, errors.Wrap(err, "resolving Cloudstack network information")
	}

	// Check if the passed network was an isolated network or the network was missing. In either case, create a
	// CloudStackIsolatedNetwork to manage the many intricacies and wait until CloudStackIsolatedNetwork is ready.
	if r.ReconciliationSubject.Spec.Zone.Network.ID == "" ||
		r.ReconciliationSubject.Spec.Zone.Network.Type == infrav1.NetworkTypeIsolated {
		netName := r.ReconciliationSubject.Spec.Zone.Network.Name
		if res, err := r.GenerateIsolatedNetwork(
			netName, func() string { return r.ReconciliationSubject.Spec.Name })(); r.ShouldReturn(res, err) {
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

// SetOwnerReferenceToEndpointSecret sets an endpoint secret's owner reference to the target cluster if needed.
func (r *CloudStackFailureDomainReconciliationRunner) SetOwnerReferenceToEndpointSecret() error {
	secret, err := r.GetSecretForFailureDomain(&r.ReconciliationSubject.Spec)
	if err != nil {
		return err
	}
	if IsSetOwnerReferenceNeeded(secret, r.CAPICluster.Name) {
		secret.OwnerReferences = append(secret.OwnerReferences,
			metav1.OwnerReference{
				Name:       r.CAPICluster.Name,
				Kind:       r.CAPICluster.Kind,
				APIVersion: r.CAPICluster.APIVersion,
				UID:        r.CAPICluster.UID,
			})
		return r.Patcher.Patch(r.RequestCtx, secret)
	}
	return nil
}

// IsSetOwnerReferenceNeeded returns true only if the secret has a matching cluster name label
// and it's not already owned by other clusters.
func IsSetOwnerReferenceNeeded(secret *corev1.Secret, clusterName string) bool {
	needSetOwner := false
	for key, value := range secret.GetLabels() {
		if strings.EqualFold(key, clusterv1.ClusterLabelName) && strings.EqualFold(value, clusterName) {
			needSetOwner = true
			for _, ref := range secret.GetOwnerReferences() {
				if strings.EqualFold(ref.Kind, "Cluster") {
					needSetOwner = false
					break
				}
			}
			break
		}
	}
	return needSetOwner
}

// ReconcileDelete on the ReconciliationRunner attempts to delete the reconciliation subject.
func (r *CloudStackFailureDomainReconciliationRunner) ReconcileDelete() (ctrl.Result, error) {
	r.Log.Info("Deleting CloudStackFailureDomain")

	return r.RunReconciliationStages(
		r.ClearMachines,
		r.DeleteOwnedObjects(
			infrav1.GroupVersion.WithKind("CloudStackAffinityGroup"),
			infrav1.GroupVersion.WithKind("CloudStackIsolatedNetwork")),
		r.CheckOwnedObjectsDeleted(
			infrav1.GroupVersion.WithKind("CloudStackAffinityGroup"),
			infrav1.GroupVersion.WithKind("CloudStackIsolatedNetwork")),
		r.RemoveFinalizer,
	)
}

// ClearMachines checks for any machines in failure domain, deletes the CAPI machine for any still in FailureDomain,
// and requeus until all CloudStack machines are cleared from the FailureDomain.
func (r *CloudStackFailureDomainReconciliationRunner) ClearMachines() (ctrl.Result, error) {
	machines := &infrav1.CloudStackMachineList{}
	if err := r.K8sClient.List(r.RequestCtx, machines, client.MatchingLabels{infrav1.FailureDomainLabelName: r.ReconciliationSubject.Name}); err != nil {
		return ctrl.Result{}, err
	}
	// Deleted CAPI machines for CloudStack machines found.
	for _, machine := range machines.Items {
		for _, ref := range machine.OwnerReferences {
			if ref.Kind == "Machine" {
				machine := &clusterv1.Machine{}
				machine.Name = ref.Name
				machine.Namespace = r.ReconciliationSubject.Namespace
				if err := r.K8sClient.Delete(r.RequestCtx, machine); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}
	if len(machines.Items) > 0 {
		return r.RequeueWithMessage("FailureDomain still has machine(s) in it.")
	}
	return ctrl.Result{}, nil
}

// RemoveFinalizer just removes the finalizer from the failure domain.
func (r *CloudStackFailureDomainReconciliationRunner) RemoveFinalizer() (ctrl.Result, error) {
	controllerutil.RemoveFinalizer(r.ReconciliationSubject, infrav1.FailureDomainFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *CloudStackFailureDomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	_, err := ctrl.NewControllerManagedBy(mgr).For(&infrav1.CloudStackFailureDomain{}).Build(reconciler)
	return err
}
