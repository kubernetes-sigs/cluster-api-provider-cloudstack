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
	"sort"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
)

const (
	conditionTypeReady   = "Ready"
	conditionStatusFalse = "False"
)

// CloudStackFailureDomainReconciler is the k8s controller manager's interface to reconcile a CloudStackFailureDomain.
// This is primarily to adapt to k8s.
type CloudStackFailureDomainReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinesets,verbs=get;list;watch
//+kubebuilder:rbac:groups=etcdcluster.cluster.x-k8s.io,resources=etcdadmclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubeadmcontrolplanes,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinesets/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=etcdcluster.cluster.x-k8s.io,resources=etcdadmclusters/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubeadmcontrolplanes/status,verbs=get;list;watch

// CloudStackFailureDomainReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackFailureDomains.
// The runner does the actual reconciliation.
type CloudStackFailureDomainReconciliationRunner struct {
	*csCtrlrUtils.ReconciliationRunner
	ReconciliationSubject *infrav1.CloudStackFailureDomain
	IsoNet                *infrav1.CloudStackIsolatedNetwork
	Machines              []infrav1.CloudStackMachine
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

// ReconcileDelete on the ReconciliationRunner attempts to delete the reconciliation subject.
func (r *CloudStackFailureDomainReconciliationRunner) ReconcileDelete() (ctrl.Result, error) {
	r.Log.Info("Deleting CloudStackFailureDomain")

	return r.RunReconciliationStages(
		r.GetAllMachinesInFailureDomain,
		r.RequeueIfClusterNotReady,
		r.RequeueIfMachineCannotBeRemoved,
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

// GetAllMachinesInFailureDomain returns all cloudstackmachines deployed in this failure domain sorted by name.
func (r *CloudStackFailureDomainReconciliationRunner) GetAllMachinesInFailureDomain() (ctrl.Result, error) {
	machines := &infrav1.CloudStackMachineList{}
	if err := r.K8sClient.List(r.RequestCtx, machines, client.MatchingLabels{infrav1.FailureDomainLabelName: r.ReconciliationSubject.Name}); err != nil {
		return ctrl.Result{}, err
	}
	items := machines.Items
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	r.Machines = items
	return ctrl.Result{}, nil
}

// RequeueIfClusterNotReady check cluster to see if there is any rolling update going on.
func (r *CloudStackFailureDomainReconciliationRunner) RequeueIfClusterNotReady() (ctrl.Result, error) {
	if len(r.Machines) > 0 {
		if !r.CAPICluster.DeletionTimestamp.IsZero() {
			return ctrl.Result{}, nil
		}
		for _, condition := range r.CAPICluster.Status.Conditions {
			if condition.Type == conditionTypeReady && condition.Status == conditionStatusFalse {
				return r.RequeueWithMessage("cluster status not ready,")
			}
		}
	}
	return ctrl.Result{}, nil
}

// RequeueIfMachineCannotBeRemoved checks for each machine to confirm it is not risky to remove it.
func (r *CloudStackFailureDomainReconciliationRunner) RequeueIfMachineCannotBeRemoved() (ctrl.Result, error) {
	// check CAPI machines for CloudStack machines found.
	for _, machine := range r.Machines {
		for _, ref := range machine.OwnerReferences {
			if ref.Kind != "Machine" {
				owner := &unstructured.Unstructured{}
				owner.SetGroupVersionKind(schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind))
				if err := r.K8sClient.Get(r.RequestCtx, client.ObjectKey{Namespace: machine.Namespace, Name: ref.Name}, owner); err != nil {
					return ctrl.Result{}, err
				}
				specReplicas, statusReplicas, err := getSpecAndStatusReplicas(owner, ref.Name, machine.Name)
				if err != nil {
					return ctrl.Result{}, err
				}
				if specReplicas != statusReplicas {
					return r.RequeueWithMessage("spec.replicas <> status.replicas, ", "owner", ref.Name, "spec.replicas", specReplicas, "status.replicas", statusReplicas)
				}

				statusReady, found, err := unstructured.NestedBool(owner.Object, "status", "ready")
				if found && err != nil {
					return ctrl.Result{}, err
				}
				if found && !statusReady {
					return r.RequeueWithMessage("status.ready not true, ", "owner", ref.Name)
				}

				statusReadyReplicas, found, err := unstructured.NestedInt64(owner.Object, "status", "readyReplicas")
				if found && err != nil {
					return ctrl.Result{}, err
				}
				if found && statusReadyReplicas != statusReplicas {
					return r.RequeueWithMessage("status.replicas <> status.readyReplicas, ", "owner", ref.Name, "status.replicas", statusReplicas, "status.readyReplicas", statusReadyReplicas)
				}
			}
		}
	}
	return ctrl.Result{}, nil
}

func getSpecAndStatusReplicas(owner *unstructured.Unstructured, ownerName, machineName string) (int64, int64, error) {
	specReplicas, found, err := unstructured.NestedInt64(owner.Object, "spec", "replicas")
	if err != nil {
		return 0, 0, err
	}
	if !found {
		return 0, 0, errors.Errorf("spec.replicas not found in %s", ownerName)
	}

	statusReplicas, found, err := unstructured.NestedInt64(owner.Object, "status", "replicas")
	if err != nil {
		return specReplicas, 0, err
	}
	if !found {
		return specReplicas, 0, errors.Errorf("status.replicas not found in %s", ownerName)
	}

	if specReplicas < 2 {
		return specReplicas, statusReplicas, errors.Errorf("spec.replicas < 2 in %s, %s cannot be moved away from failure domain", ownerName, machineName)
	}

	return specReplicas, statusReplicas, nil
}

// ClearMachines deletes the CAPI machine in FailureDomain.
func (r *CloudStackFailureDomainReconciliationRunner) ClearMachines() (ctrl.Result, error) {
	// pick first machine to delete
	if len(r.Machines) > 0 {
		for _, ref := range r.Machines[0].OwnerReferences {
			if ref.Kind == "Machine" {
				machine := &clusterv1.Machine{}
				if err := r.K8sClient.Get(r.RequestCtx, client.ObjectKey{Namespace: r.ReconciliationSubject.Namespace, Name: ref.Name}, machine); err != nil {
					return ctrl.Result{}, err
				}
				if !machine.DeletionTimestamp.IsZero() {
					return r.RequeueWithMessage("machine is being deleted, ", "machine", machine.Name)
				}
				if err := r.K8sClient.Delete(r.RequestCtx, machine); err != nil {
					return ctrl.Result{}, err
				}
				return r.RequeueWithMessage("start to delete machine, ", "machine", machine.Name)
			}
		}
	}
	return ctrl.Result{}, nil
}

// RemoveFinalizer just removes the finalizer from the failure domain.
func (r *CloudStackFailureDomainReconciliationRunner) RemoveFinalizer() (ctrl.Result, error) {
	controllerutil.RemoveFinalizer(r.ReconciliationSubject, infrav1.FailureDomainFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *CloudStackFailureDomainReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackFailureDomain{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(ctrl.LoggerFrom(ctx), reconciler.WatchFilterValue)).
		Complete(reconciler)
}
