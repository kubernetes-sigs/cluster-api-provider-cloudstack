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
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
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

type ClusterContext struct {
	CAPICluster *capiv1.Cluster
	CSCluster   *infrav1.CloudStackCluster
	Zones       *infrav1.CloudStackZoneList
}

// CloudStackClusterReconciler reconciles a CloudStackCluster object.
type CloudStackClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	CS     cloud.Client
	CC     *ClusterContext
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
		r.fetchRelatedCRDs,
		r.checkIfPaused,
		r.reconcileDelete,
		r.reconcile,
		r.patchCSCluster,
	)
}

// Fetches all CRDs relavent to reconciling a CloudStackCluster.
func (r *CloudStackClusterReconciler) fetchRelatedCRDs(
	ctx context.Context, req ctrl.Request) (retRes ctrl.Result, reterr error) {

	r.CC = &ClusterContext{
		CSCluster:   &infrav1.CloudStackCluster{},
		CAPICluster: &capiv1.Cluster{},
		Zones:       &infrav1.CloudStackZoneList{}}

	// Get CloudStack cluster.
	if reterr = r.Client.Get(ctx, req.NamespacedName, r.CC.CSCluster); reterr != nil {
		if client.IgnoreNotFound(reterr) == nil {
			r.Log.Info("Cluster not found.")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrap(reterr, "error encountered while fetching CloudStackCluster crd")
	}

	// Get CAPI cluster.
	r.CC.CAPICluster, reterr = util.GetOwnerCluster(ctx, r.Client, r.CC.CSCluster.ObjectMeta)
	if reterr != nil {
		return ctrl.Result{}, errors.Wrap(reterr, "error encountered while fetching CAPI Cluster crd")
	} else if r.CC.CAPICluster == nil {
		return ctrl.Result{}, errors.New("CAPI Cluster not found")
	}

	// Get CloudStackZones.
	_, reterr = r.fetchZones(ctx, req)
	if reterr != nil {
		return reconcile.Result{}, errors.Wrap(reterr, "error encountered fetching CloudStackZone(s)")
	}
	return ctrl.Result{}, nil
}

// checkIfPaused returns with reque later set if paused.
func (r *CloudStackClusterReconciler) checkIfPaused(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if annotations.IsPaused(r.CC.CAPICluster, r.CC.CSCluster) {
		r.Log.Info("Cluster is paused. Refusing to reconcile.")
		return reconcile.Result{RequeueAfter: requeueTimeout}, nil
	}
	return reconcile.Result{}, nil
}

// patchCSCluster patches the changes made to the Reconciler local copy of the cluster to the API stored version.
func (r *CloudStackClusterReconciler) patchCSCluster(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Setup patcher. This ensures modifications to the csCluster copy fetched above are patched into the origin.
	patchHelper, err := patch.NewHelper(r.CC.CSCluster, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err = patchHelper.Patch(ctx, r.CC.CSCluster); err != nil {
		err = errors.Wrapf(err, "error patching CloudStackCluster %s/%s", r.CC.CSCluster.Namespace, r.CC.CSCluster.Name)
		//retErr = multierror.Append(err, err)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// reconcile actually reconciles the cluster.
func (r *CloudStackClusterReconciler) reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// Reconcile only if deletion timestamp not present.
	if !r.CC.CSCluster.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	r.Log.V(1).Info("Reconciling CloudStackCluster.", "clusterSpec", r.CC.CSCluster.Spec)

	return r.runWith(ctx, req,
		r.generateZones,
		r.checkOwnedCRDsforReadiness,
		r.resolveClusterDetails,
	)
}

// resolveClusterDetails fetches cluster specific details like domain and account IDs.
func (r *CloudStackClusterReconciler) resolveClusterDetails(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Ensure that CAPI won't prematurely delete this CloudStackCluster.
	controllerutil.AddFinalizer(r.CC.CSCluster, infrav1.ClusterFinalizer)

	// Create and or fetch cluster components.
	err := r.CS.GetOrCreateCluster(r.CC.CSCluster)
	if err == nil {
		r.Log.Info("Fetched cluster info successfully.")
		r.Log.V(1).Info("Post fetch cluster status.", "clusterStatus", r.CC.CSCluster.Status)

		// Set cluster to ready to indicate readiness to CAPI.
		r.CC.CSCluster.Status.Ready = true
	}
	return ctrl.Result{}, err
}

// reconcileDelete cleans up resources used by the cluster and finaly removes the CloudStackCluster's finalizers.
func (r *CloudStackClusterReconciler) reconcileDelete(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if r.CC.CSCluster.DeletionTimestamp.IsZero() { // Reconcile deletion only if deletion timestamp is present.
		return ctrl.Result{}, nil
	}

	r.Log.V(1).Info("reconcileDelete CloudStackCluster...")

	if err := r.CS.DisposeClusterResources(r.CC.CSCluster); err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(r.CC.CSCluster, infrav1.ClusterFinalizer)
	return ctrl.Result{}, nil
}

// checkOwnedCRDsforReadiness checks that owned CRDs like Zones are ready.
func (r *CloudStackClusterReconciler) checkOwnedCRDsforReadiness(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	if len(r.CC.CSCluster.Spec.Zones) != len(r.CC.Zones.Items) {
		return reconcile.Result{}, errors.New("did not find all zones required for cluster reconciliation")
	}

	for _, zone := range r.CC.Zones.Items {
		if !zone.Status.Ready {
			r.Log.Info("not all required zones are ready, requeing")
			return ctrl.Result{RequeueAfter: requeueTimeout}, nil
		}
	}

	return ctrl.Result{}, nil
}

// generateZones generates a CloudStackClusterZone CRD for each of the CloudStackCluster's spec Zones.
func (r *CloudStackClusterReconciler) generateZones(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	for _, zone := range r.CC.CSCluster.Spec.Zones {
		if err := r.generateZone(ctx, zone); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
				return reconcile.Result{}, errors.Wrap(err, "error encountered when creating CloudStackZone")
			}
		}
	}
	return ctrl.Result{}, nil
}

// generateZone generates a specified CloudStackZone CRD owned by the CloudStackCluster.
func (r *CloudStackClusterReconciler) generateZone(ctx context.Context, zoneSpec infrav1.Zone) error {
	csZone := &infrav1.CloudStackZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:        strings.ToLower(zoneSpec.Name),
			Namespace:   r.CC.CSCluster.Namespace,
			Labels:      map[string]string{"OwnedBy": r.CC.CSCluster.Name},
			Annotations: map[string]string{},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.CC.CSCluster, controlplanev1.GroupVersion.WithKind("CloudStackCluster")),
			},
		},
		Spec: infrav1.CloudStackZoneSpec{Name: zoneSpec.Name},
	}

	if err := r.Client.Create(ctx, csZone); err != nil {
		return errors.Wrap(err, "failed to create zone")
	}
	return nil
}

// fetchZones fetches CloudStackZones owned by a CloudStackCluster via an ownership label.
func (r *CloudStackClusterReconciler) fetchZones(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	labels := map[string]string{"OwnedBy": r.CC.CSCluster.Name}
	if err := r.Client.List(
		ctx,
		r.CC.Zones,
		client.InNamespace(r.CC.CSCluster.Namespace),
		client.MatchingLabels(labels),
	); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to list zones")
	}
	return ctrl.Result{}, nil
}

// runWith runs all given CloudStackClusterReconcilerFunc with the given context and request object and returns on
// either the first encountered error, or after all successfully complete.
func (r *CloudStackClusterReconciler) runWith(
	ctx context.Context, req ctrl.Request, fns ...CloudStackClusterReconcilerFunc) (ctrl.Result, error) {
	for _, fn := range fns {
		if rslt, err := fn(ctx, req); err != nil || rslt.Requeue == true || rslt.RequeueAfter != 0 {
			return rslt, err
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
