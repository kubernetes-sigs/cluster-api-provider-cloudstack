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
	"fmt"
	"reflect"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/pkg/errors"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
)

// RBAC permissions used in all reconcilers. Events and Secrets.
// "" empty string as the api group indicates core kubernetes objects. "*" indicates all objects.
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

// RBAC permissions for CloudStackCluster.
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters/status,verbs=create;get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// CloudStackClusterReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackClusters.
// The runner does the actual reconciliation.
type CloudStackClusterReconciliationRunner struct {
	csCtrlrUtils.ReconciliationRunner
	Zones                 *infrav1.CloudStackZoneList
	ReconciliationSubject *infrav1.CloudStackCluster
}

// CloudStackClusterReconciler is the k8s controller manager's interface to reconcile a CloudStackCluster.
// This is primarily to adapt to k8s.
type CloudStackClusterReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

// Initialize a new CloudStackCluster reconciliation runner with concrete types and initialized member fields.
func NewCSClusterReconciliationRunner() *CloudStackClusterReconciliationRunner {
	// Set concrete type and init pointers.
	r := &CloudStackClusterReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackCluster{}}
	r.Zones = &infrav1.CloudStackZoneList{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	r.ReconciliationRunner = csCtrlrUtils.NewRunner(r, r.ReconciliationSubject)
	// For the CloudStackCluster, the ReconciliationSubject is the CSCluster
	// Have to do after or the setup method will overwrite the link.
	r.CSCluster = r.ReconciliationSubject

	return r
}

// Reconcile is the method k8s will call upon a reconciliation request.
func (reconciler *CloudStackClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	return NewCSClusterReconciliationRunner().
		UsingBaseReconciler(reconciler.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunBaseReconciliationStages()
}

// Reconcile actually reconciles the CloudStackCluster.
func (r *CloudStackClusterReconciliationRunner) Reconcile() (res ctrl.Result, reterr error) {
	return r.RunReconciliationStages(
		r.RequeueIfMissingBaseCRs,
		r.CreateZones(r.ReconciliationSubject.Spec.Zones),
		r.CheckOwnedCRDsForReadiness(infrav1.GroupVersion.WithKind("CloudStackZone")),
		r.GetZones(r.Zones),
		r.VerifyZoneCRDs,
		r.SetFailureDomains,
		r.SetReady)
}

// SetReady adds a finalizer and sets the cluster status to ready.
func (r *CloudStackClusterReconciliationRunner) SetReady() (ctrl.Result, error) {
	controllerutil.AddFinalizer(r.ReconciliationSubject, infrav1.ClusterFinalizer)
	r.ReconciliationSubject.Status.Ready = true
	return ctrl.Result{}, nil
}

// CheckZoneDetails verifies the Zone CRDs found match against those requested.
func (r *CloudStackClusterReconciliationRunner) VerifyZoneCRDs() (ctrl.Result, error) {
	expected := len(r.ReconciliationSubject.Spec.Zones)
	actual := len(r.Zones.Items)
	if expected != actual {
		return r.RequeueWithMessage(fmt.Sprintf("Expected %d Zones, but found %d", expected, actual))
	}
	for _, zone := range r.Zones.Items {
		if !zone.Status.Ready {
			return r.RequeueWithMessage(fmt.Sprintf("Zone %s/%s not ready, requeueing.", zone.Namespace, zone.Name))
		}
	}
	return ctrl.Result{}, nil
}

// SetFailureDomains sets failure domains to be used for CAPI machine placement.
func (r *CloudStackClusterReconciliationRunner) SetFailureDomains() (ctrl.Result, error) {
	r.ReconciliationSubject.Status.FailureDomains = capiv1.FailureDomains{}
	for _, zone := range r.Zones.Items {
		r.ReconciliationSubject.Status.FailureDomains[zone.Spec.ID] = capiv1.FailureDomainSpec{ControlPlane: true}
	}
	return ctrl.Result{}, nil
}

// ReconcileDelete cleans up resources used by the cluster and finaly removes the CloudStackCluster's finalizers.
func (r *CloudStackClusterReconciliationRunner) ReconcileDelete() (ctrl.Result, error) {
	r.Log.Info("Deleting CloudStackCluster.")
	if res, err := r.GetZones(r.Zones)(); r.ShouldReturn(res, err) {
		return res, err
	}
	if len(r.Zones.Items) > 0 {
		for idx := range r.Zones.Items {
			if err := r.K8sClient.Delete(r.RequestCtx, &r.Zones.Items[idx]); err != nil {
				return ctrl.Result{}, err
			}
		}
		return r.RequeueWithMessage("Child Zones still present, requeueing.")
	}
	controllerutil.RemoveFinalizer(r.ReconciliationSubject, infrav1.ClusterFinalizer)
	return ctrl.Result{}, nil
}

// Called in main, this registers the cluster reconciler to the CAPI controller manager.
func (reconciler *CloudStackClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	controller, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackCluster{}).
		WithEventFilter(
			predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldCluster := e.ObjectOld.(*infrav1.CloudStackCluster).DeepCopy()
					newCluster := e.ObjectNew.(*infrav1.CloudStackCluster).DeepCopy()
					// Ignore resource version because they are unique
					oldCluster.ObjectMeta.ResourceVersion = ""
					newCluster.ObjectMeta.ResourceVersion = ""
					// Ignore finalizers updates
					oldCluster.ObjectMeta.Finalizers = nil
					newCluster.ObjectMeta.Finalizers = nil
					// Ignore ManagedFields because they are mirror of ObjectMeta
					oldCluster.ManagedFields = nil
					newCluster.ManagedFields = nil
					// Ignore incremental status updates
					oldCluster.Status = infrav1.CloudStackClusterStatus{}
					newCluster.Status = infrav1.CloudStackClusterStatus{}

					return !reflect.DeepEqual(oldCluster, newCluster)
				},
			},
		).Build(reconciler)
	if err != nil {
		return errors.Wrap(err, "building CloudStackCluster controller:")
	}

	// Add a watch on CAPI Cluster objects for unpause and ready events.
	err = controller.Watch(
		&source.Kind{Type: &capiv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(
			util.ClusterToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("CloudStackCluster"))),
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldCluster := e.ObjectOld.(*capiv1.Cluster)
				newCluster := e.ObjectNew.(*capiv1.Cluster)
				return oldCluster.Spec.Paused && !newCluster.Spec.Paused
			},
			DeleteFunc: func(e event.DeleteEvent) bool { return false },
			CreateFunc: func(e event.CreateEvent) bool { return false }})
	return errors.Wrap(err, "building CloudStackCluster controller:")
}
