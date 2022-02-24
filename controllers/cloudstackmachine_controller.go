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
	"time"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// CloudStackMachineReconciler reconciles a CloudStackMachine object.
type CloudStackMachineReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	CS     cloud.Client
}

const requeueTimeout = 5 * time.Second
const destoryVMRequeueInterval = 10 * time.Second

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinesets,verbs=get;list;watch
// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubeadmcontrolplanes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CloudStackMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	log := r.Log.WithValues("machine", req.Name, "namespace", req.Namespace)
	log.V(1).Info("Reconcile CloudStackMachine")

	// Fetch the CloudStackMachine.
	csMachine := &infrav1.CloudStackMachine{}
	if err := r.Client.Get(ctx, req.NamespacedName, csMachine); err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Info("CloudStackMachine not found.")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Setup patcher. This ensures modifications to the csMachine copy fetched above are patched into the origin.
	patchHelper, err := patch.NewHelper(csMachine, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() { // If there was no error on return, but the patch fails, set the error accordingly.
		if err = patchHelper.Patch(ctx, csMachine); err != nil {
			msg := "error patching CloudStackMachine %s/%s"
			err = errors.Wrapf(err, msg, csMachine.Namespace, csMachine.Name)
			retErr = multierror.Append(retErr, err)
		}
	}()

	// Fetch the CAPI Machine.
	capiMachine, err := util.GetOwnerMachine(ctx, r.Client, csMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	} else if capiMachine == nil {
		log.Info("Waiting for CAPI cluster controller to set owner reference on CloudStack machine.")
		return ctrl.Result{RequeueAfter: requeueTimeout}, nil
	}

	// Fetch the CAPI Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, capiMachine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist.")
		return ctrl.Result{RequeueAfter: requeueTimeout}, nil
	}

	// Check if the machine is paused.
	if annotations.IsPaused(cluster, csMachine) {
		log.Info("CloudStackMachine or linked Cluster is paused. Requeuing reconcile.")
		return reconcile.Result{}, nil
	}

	// Delete VM instance if deletion timestamp present.
	if !csMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, csMachine, capiMachine)
	}

	// Fetch the CloudStack cluster associated with this machine.
	csCluster := &infrav1.CloudStackCluster{}
	if err := r.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: csMachine.Namespace,
			Name:      cluster.Spec.InfrastructureRef.Name},
		csCluster); err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Info("CloudStackCluster not found.")
			return ctrl.Result{RequeueAfter: requeueTimeout}, nil
		}
		return ctrl.Result{}, err
	} else if csCluster.Status.ZoneID == "" {
		log.Info("CloudStackCluster ZoneId not initialized. Likely not ready.")
		return ctrl.Result{RequeueAfter: requeueTimeout}, nil
	}

	// Reconcile a VM instance for creates/updates.
	return r.reconcile(ctx, log, csMachine, capiMachine, csCluster)
}

// Actually reconcile/Create a VM instance.
func (r *CloudStackMachineReconciler) reconcile(
	ctx context.Context,
	log logr.Logger,
	csMachine *infrav1.CloudStackMachine,
	capiMachine *capiv1.Machine,
	csCluster *infrav1.CloudStackCluster) (ctrl.Result, error) {

	log.V(1).Info("reconcile CloudStackMachine")

	// Make sure bootstrap data is available in CAPI machine.
	if capiMachine.Spec.Bootstrap.DataSecretName == nil {
		log.Info("Bootstrap DataSecretName not yet available.")
		return ctrl.Result{}, nil
	}
	log.Info("Got Bootstrap DataSecretName.")

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: capiMachine.Namespace, Name: *capiMachine.Spec.Bootstrap.DataSecretName}
	if err := r.Client.Get(context.TODO(), key, secret); err != nil {
		return ctrl.Result{}, err
	}

	value, ok := secret.Data["value"]
	if !ok {
		return ctrl.Result{}, errors.New("bootstrap secret data not ok")
	}

	// Create VM (or Fetch if present). Will set ready to true.
	if err := r.CS.GetOrCreateVMInstance(csMachine, capiMachine, csCluster, string(value)); err == nil {
		if !controllerutil.ContainsFinalizer(csMachine, infrav1.MachineFinalizer) { // Fetched or Created?
			log.Info("CloudStack instance Created", "instanceStatus", csMachine.Status, "instanceSpec", csMachine.Spec)
			controllerutil.AddFinalizer(csMachine, infrav1.MachineFinalizer)
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	if csMachine.Status.InstanceState == "Running" {
		log.Info("Machine instance is Running...")
		csMachine.Status.Ready = true
	} else if csMachine.Status.InstanceState == "Error" {
		log.Info("CloudStackMachine VM in error state.  Deleting associated Machine.", "csMachine", csMachine)
		if err := r.Client.Delete(ctx, capiMachine); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: requeueTimeout}, nil
	} else {
		log.Info(fmt.Sprintf("Instance not ready, is %s.", csMachine.Status.InstanceState))
		return ctrl.Result{RequeueAfter: requeueTimeout}, nil
	}

	if util.IsControlPlaneMachine(capiMachine) && csCluster.Status.NetworkType != cloud.NetworkTypeShared {
		log.Info("Assigning VM to load balancer rule.")
		err := r.CS.AssignVMToLoadBalancerRule(csCluster, *csMachine.Spec.InstanceID)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// Reconcile/Destroy a deleted VM instance.
func (r *CloudStackMachineReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	csMachine *infrav1.CloudStackMachine,
	capiMachine *capiv1.Machine,
) (ctrl.Result, error) {

	// Remove any CAPC managed Affinity groups if owner references a deleted object.
	if deleted, err := csCtrlrUtils.IsOwnerDeleted(ctx, r.Client, capiMachine); err != nil {
		return ctrl.Result{}, err
	} else if deleted {
		if err := r.RemoveManagedAffinity(log, capiMachine, csMachine); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "Error encountered when removing affinity group.")
		}
	}

	log.Info("Deleting instance", "instance-id", *csMachine.Spec.InstanceID)
	if err := r.CS.DestroyVMInstance(csMachine); err != nil {
		if err.Error() == "VM deletion in progress" {
			log.Info(err.Error())
			return ctrl.Result{RequeueAfter: destoryVMRequeueInterval}, nil
		}
		return ctrl.Result{}, err
	}
	controllerutil.RemoveFinalizer(csMachine, infrav1.MachineFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager is called in main, this registers the machine reconciler to the CAPI controller manager.
func (r *CloudStackMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {

	controller, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackMachine{}).
		WithEventFilter(
			predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldMachine := e.ObjectOld.(*infrav1.CloudStackMachine).DeepCopy()
					newMachine := e.ObjectNew.(*infrav1.CloudStackMachine).DeepCopy()
					// Ignore resource version because they are unique.
					oldMachine.ObjectMeta.ResourceVersion = ""
					newMachine.ObjectMeta.ResourceVersion = ""
					// Ignore generation because it's not used in reconcile.
					oldMachine.ObjectMeta.Generation = 0
					newMachine.ObjectMeta.Generation = 0
					// Ignore finalizers updates.
					oldMachine.ObjectMeta.Finalizers = nil
					newMachine.ObjectMeta.Finalizers = nil
					// Ignore ManagedFields because they are mirror of ObjectMeta.
					oldMachine.ManagedFields = nil
					newMachine.ManagedFields = nil
					// Ignore incremental status updates.
					oldMachine.Status = infrav1.CloudStackMachineStatus{}
					newMachine.Status = infrav1.CloudStackMachineStatus{}
					// Ignore provide ID.
					oldMachine.Spec.ProviderID = nil
					newMachine.Spec.ProviderID = nil
					// Ignore instance ID.
					oldMachine.Spec.InstanceID = nil
					newMachine.Spec.InstanceID = nil

					return !reflect.DeepEqual(oldMachine, newMachine)
				},
			},
		).Build(r)
	if err != nil {
		return err
	}

	// Watch CAPI machines for changes.
	// Queues a reconcile request for owned CloudStackMachine on change.
	// Used to update when bootstrap data becomes available.
	if err = controller.Watch(
		&source.Kind{Type: &capiv1.Machine{}},
		handler.EnqueueRequestsFromMapFunc(
			util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("CloudStackMachine"))),
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldMachine := e.ObjectOld.(*capiv1.Machine)
				newMachine := e.ObjectNew.(*capiv1.Machine)

				return oldMachine.Spec.Bootstrap.DataSecretName == nil && newMachine.Spec.Bootstrap.DataSecretName != nil
			},
		},
	); err != nil {
		return err
	}

	// Used below, this maps CAPI clusters to CAPC machines.
	csMachineMapper, err := util.ClusterToObjectsMapper(r.Client, &infrav1.CloudStackMachineList{}, mgr.GetScheme())
	if err != nil {
		return err
	}

	// Add a watch on CAPI Cluster objects for unpause and ready events.
	return controller.Watch(
		&source.Kind{Type: &capiv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(csMachineMapper),
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldCluster := e.ObjectOld.(*capiv1.Cluster)
				newCluster := e.ObjectNew.(*capiv1.Cluster)
				return oldCluster.Spec.Paused && !newCluster.Spec.Paused
			},
			CreateFunc: func(e event.CreateEvent) bool {
				_, ok := e.Object.GetAnnotations()[capiv1.PausedAnnotation]
				return ok
			},
		},
	)
}

// RemoveManagedAffinity considers a machine's affinity management strategy and removes the created affinity group.
// if it exists.
func (r *CloudStackMachineReconciler) RemoveManagedAffinity(
	log logr.Logger,
	capiMachine *capiv1.Machine,
	csMachine *infrav1.CloudStackMachine,
) error {

	ownerRef := csCtrlrUtils.GetManagementOwnerRef(capiMachine)
	if ownerRef == nil {
		return errors.Errorf("Could not find management owner reference for %s/%s", csMachine.Namespace, csMachine.Name)
	}
	name, err := csMachine.AffinityGroupName(capiMachine)
	if err != nil {
		return err
	}
	group := &cloud.AffinityGroup{Name: name}
	_ = r.CS.FetchAffinityGroup(group)
	if group.ID == "" { // Affinity group not found, must have been deleted.
		return nil
	}

	log.Info(fmt.Sprintf("Deleting affinity group '%s'", name))

	return r.CS.DeleteAffinityGroup(group)
}
