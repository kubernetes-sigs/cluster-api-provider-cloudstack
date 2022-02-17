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

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/cluster-api/util"

	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// CloudStackClusterReconciler reconciles a CloudStackCluster object.
type CloudStackClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	CS     cloud.Client
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
func (r *CloudStackClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	log := r.Log.WithValues("cluster", req.Name, "namespace", req.Namespace)
	log.V(1).Info("Reconcile CloudStackCluster")

	// Get CloudStack cluster.
	csCluster := &infrav1.CloudStackCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, csCluster); err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Info("Cluster not found.")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get CAPI cluster.
	capiCluster, err := util.GetOwnerCluster(ctx, r.Client, csCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	} else if capiCluster == nil {
		log.Info("Waiting for CAPI Cluster controller to set owner reference on CloudStack cluster.")
		return reconcile.Result{}, nil
	}

	// Check the cluster is not paused.
	if annotations.IsPaused(capiCluster, csCluster) {
		log.Info("Cluster is paused. Refusing to reconcile.")
		return reconcile.Result{}, nil
	}

	// Setup patcher. This ensures modifications to the csCluster copy fetched above are patched into the origin.
	if patchHelper, err := patch.NewHelper(csCluster, r.Client); err != nil {
		return ctrl.Result{}, err
	} else {
		defer func() {
			if err = patchHelper.Patch(ctx, csCluster); err != nil {
				msg := "error patching CloudStackCluster %s/%s"
				err = errors.Wrapf(err, msg, csCluster.Namespace, csCluster.Name)
				retErr = multierror.Append(retErr, err)
			}
		}()
	}

	// Delete Cluster Resources if deletion timestamp present.
	if !csCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, csCluster)
	}

	// Reconcile remaining clusters.
	return r.reconcile(ctx, log, csCluster)
}

// Actually reconcile cluster.
func (r *CloudStackClusterReconciler) reconcile(
	ctx context.Context,
	log logr.Logger,
	csCluster *infrav1.CloudStackCluster,
) (ctrl.Result, error) {

	log.V(1).Info("reconcile CloudStackCluster")

	// Prevent premature deletion of the csCluster construct from CAPI.
	controllerutil.AddFinalizer(csCluster, infrav1.ClusterFinalizer)

	// Create and or fetch cluster components -- sets cluster to ready if no errors.
	err := r.CS.GetOrCreateCluster(csCluster)
	if err == nil {
		log.Info("Fetched cluster info successfully.", "clusterSpec", csCluster.Spec,
			"clusterStatus", csCluster.Status)
	}
	return ctrl.Result{}, err
}

// Delete a cluster.
func (r *CloudStackClusterReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	csCluster *infrav1.CloudStackCluster,
) (ctrl.Result, error) {

	log.V(1).Info("reconcileDelete CloudStackCluster...")

	// TODO Decide what resources to remove w/Cluster if any.
	// cloud.DestroyCluster(r.CS, csStackCluster)

	controllerutil.RemoveFinalizer(csCluster, infrav1.ClusterFinalizer)
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
