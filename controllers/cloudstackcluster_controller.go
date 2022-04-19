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
	"reflect"

	"github.com/pkg/errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
)

// CloudStackClusterReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackCluster reconciliation.
type CloudStackClusterReconciliationRunner struct {
	csCtrlrUtils.ReconciliationRunner
	Zones                 infrav1.CloudStackZoneList
	ReconciliationSubject *infrav1.CloudStackCluster
	CSUser                cloud.Client
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters/status,verbs=create;get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackzones,verbs=get;list;watch;create;delete

// TODO review whether these unnamed groups are used and if so add clarity via a comment.
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to move the current state of the cluster
// closer to the desired state.
// CloudStackClusterReconciler is the k8s controller manager's interface to reconcile a CloudStackCluster.
type CloudStackClusterReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

func (r *CloudStackClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	runner := &CloudStackClusterReconciliationRunner{}
	return runner.
		UsingBaseReconciler(r.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunReconciliationStages(
			runner.SetupLogger,
			runner.GetBaseCRDs,
			runner.SetReconciliationSubjectToCSCluster,
			runner.SetupPatcher,
			runner.Reconcile,
			runner.PatchChangesBackToAPI,
		)
}

// SetReconciliationSubjectToCSCluster set reconciliation subject to ReconciliationRunner's CloudStackCluster object
// as in this case, the subject of reconciliation is one of the CloudStack base CRDs (used in all reconciles).
func (r *CloudStackClusterReconciliationRunner) SetReconciliationSubjectToCSCluster() (res ctrl.Result, retErr error) {
	r.Log.V(1).Info("Setting CloudStack cluster as the reconciliation subject.")
	r.ReconciliationRunner.ReconciliationSubject = r.CSCluster
	r.ReconciliationSubject = r.CSCluster
	return res, nil
}

// Reconcile actually reconciles the CloudStackCluster.
func (r *CloudStackClusterReconciliationRunner) Reconcile() (res ctrl.Result, reterr error) {
	if !r.ReconciliationSubject.DeletionTimestamp.IsZero() { // Reconcile deletion if timestamp is present.
		return r.ReconcileDelete()
	}
	r.Log.V(1).Info("Reconciling CloudStackCluster.", "clusterSpec", r.ReconciliationSubject.Spec)

	return r.RunReconciliationStages(
		r.CreateZones(r.CSCluster.Spec.Zones),
		r.GetZones(&r.Zones),
		r.CheckOwnedCRDsforReadiness,
		r.ResolveClusterDetails)
}

// ReconcileDelete cleans up resources used by the cluster and finaly removes the CloudStackCluster's finalizers.
func (r *CloudStackClusterReconciliationRunner) ReconcileDelete() (ctrl.Result, error) {

	r.Log.V(1).Info("Deleting CloudStackCluster.")

	if err := r.CS.DisposeClusterResources(r.ReconciliationSubject); err != nil {
		return ctrl.Result{}, err
	}
	controllerutil.RemoveFinalizer(r.ReconciliationSubject, infrav1.ClusterFinalizer)
	return ctrl.Result{}, nil
}

// ResolveClusterDetails fetches cluster specific details like domain and account IDs.
func (r *CloudStackClusterReconciliationRunner) ResolveClusterDetails() (ctrl.Result, error) {
	// Ensure that CAPI won't prematurely delete this CloudStackCluster.
	controllerutil.AddFinalizer(r.CSCluster, infrav1.ClusterFinalizer)

	// Create and or fetch cluster components.
	err := r.CS.GetOrCreateCluster(r.CSCluster)
	if err == nil {
		r.Log.Info("Fetched cluster info sussfully.")
		r.Log.V(1).Info("Post fetch cluster status.", "clusterStatus", r.ReconciliationSubject.Status)

		// Set cluster to ready to indicate readiness to CAPI.
		r.ReconciliationSubject.Status.Ready = true
	}
	return ctrl.Result{}, err
}

// CheckOwnedCRDsforReadiness checks that owned CRDs like Zones are ready.
func (r *CloudStackClusterReconciliationRunner) CheckOwnedCRDsforReadiness() (ctrl.Result, error) {
	expected := len(r.ReconciliationSubject.Spec.Zones)
	zonesFound := len(r.Zones.Items)
	if len(r.ReconciliationSubject.Spec.Zones) != len(r.Zones.Items) {
		return reconcile.Result{}, errors.Errorf(
			"expected to find %v CloudStackZone CRDs, but found %v", expected, zonesFound)
	}
	for _, zone := range r.Zones.Items {
		if !zone.Status.Ready {
			r.Log.Info("not all required zones are ready, requeing")
			return ctrl.Result{RequeueAfter: utils.RequeueTimeout}, nil
		}
	}

	return ctrl.Result{}, nil
}

// Called in main, this registers the cluster reconciler to the CAPI controller manager.
func (r *CloudStackClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
		).Build(r)
	if err != nil {
		return err
	}
	return controller.Watch( // Add a watch on CAPI Cluster objects for unpause and ready events.
		&source.Kind{Type: &capiv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(
			util.ClusterToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("CloudStackCluster"))),
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldCluster := e.ObjectOld.(*capiv1.Cluster)
				newCluster := e.ObjectNew.(*capiv1.Cluster)
				return oldCluster.Spec.Paused && !newCluster.Spec.Paused
			},
			CreateFunc: func(e event.CreateEvent) bool {
				_, ok := e.Object.GetAnnotations()[capiv1.PausedAnnotation]
				return ok
			}},
	)
}
