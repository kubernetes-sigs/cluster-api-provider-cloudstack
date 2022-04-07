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
	"sigs.k8s.io/cluster-api/util"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// CloudStackClusterReconciler reconciles a CloudStackCluster object.
type CloudStackClusterReconciler struct {
	CloudStackBaseReconciler
	CloudStackZoneUser
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// TODO review whether these unnamed groups are used and if so add clarity via a comment.
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile

type CloudStackClusterReconcilerFunc func(context.Context, ctrl.Request) (ctrl.Result, error)

func (r *CloudStackClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	r.Log = r.Log.WithValues("cluster", req.Name, "namespace", req.Namespace)
	r.Log.V(1).Info("Reconcile CloudStackCluster")

	return r.runWith(ctx, req,
		r.GetBaseCRDs,
		r.CheckIfPaused,
		r.FetchZones,
		r.reconcileDelete,
		r.reconcile,
		r.patchChangesBackToAPI,
	)
}

// reconcile actually reconciles cluster.
func (r *CloudStackClusterReconciler) reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Reconcile only if deletion timestamp not present.
	if !r.CSCluster.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	r.Log.V(1).Info("Reconciling CloudStackCluster.", "clusterSpec", r.CSCluster.Spec)

	return r.runWith(ctx, req,
		r.GenerateZones,
		r.checkOwnedCRDsforReadiness,
		r.resolveClusterDetails,
	)
}

// resolveClusterDetails fetches cluster specific details like domain and aunt IDs.
func (r *CloudStackClusterReconciler) resolveClusterDetails(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Ensure that CAPI won't prematurely delete this CloudStackCluster.
	controllerutil.AddFinalizer(r.CSCluster, infrav1.ClusterFinalizer)

	// Create and or fetch cluster components.
	err := r.CS.GetOrCreateCluster(r.CSCluster)
	if err == nil {
		r.Log.Info("Fetched cluster info sussfully.")
		r.Log.V(1).Info("Post fetch cluster status.", "clusterStatus", r.CSCluster.Status)

		// Set cluster to ready to indicate readiness to CAPI.
		r.CSCluster.Status.Ready = true
	}
	return ctrl.Result{}, err
}

// reconcileDelete cleans up resources used by the cluster and finaly removes the CloudStackCluster's finalizers.
func (r *CloudStackClusterReconciler) reconcileDelete(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if r.CSCluster.DeletionTimestamp.IsZero() { // Reconcile deletion only if deletion timestamp is present.
		return ctrl.Result{}, nil
	}

	r.Log.V(1).Info("reconcileDelete CloudStackCluster...")

	if err := r.CS.DisposeClusterResources(r.CSCluster); err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(r.CSCluster, infrav1.ClusterFinalizer)
	return ctrl.Result{}, nil
}

// checkOwnedCRDsforReadiness checks that owned CRDs like Zones are ready.
func (r *CloudStackClusterReconciler) checkOwnedCRDsforReadiness(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	if len(r.CSCluster.Spec.Zones) != len(r.Zones.Items) {
		return reconcile.Result{}, errors.New("did not find all zones required for cluster reconciliation")
	}

	for _, zone := range r.Zones.Items {
		if !zone.Status.Ready {
			r.Log.Info("not all required zones are ready, requeing")
			return ctrl.Result{RequeueAfter: requeueTimeout}, nil
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
