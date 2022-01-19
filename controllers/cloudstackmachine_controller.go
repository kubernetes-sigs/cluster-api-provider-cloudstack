/*
Copyright 2022.

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
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
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

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1alpha3"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	capiv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

// CloudStackMachineReconciler reconciles a CloudStackMachine object
type CloudStackMachineReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	CS     cloud.Client
}

const RequeueTimeout = 5 * time.Second

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CloudStackMachineReconciler) Reconcile(req ctrl.Request) (retRes ctrl.Result, retErr error) {
	ctx := context.Background()
	log := r.Log.WithValues("machine", req.Name, "namespace", req.Namespace)
	log.V(1).Info("Reconcile CloudStackMachine")

	// Fetch the CloudStackMachine.
	csMachine := &infrav1.CloudStackMachine{}
	if retErr = r.Client.Get(ctx, req.NamespacedName, csMachine); retErr != nil {
		if client.IgnoreNotFound(retErr) == nil {
			log.Info("CloudStackMachine not found.")
		}
		return ctrl.Result{}, client.IgnoreNotFound(retErr)
	}

	// Setup patcher. This ensures modifications to the csMachine copy fetched above are patched into the origin.
	if patchHelper, retErr := patch.NewHelper(csMachine, r.Client); retErr != nil {
		return ctrl.Result{}, retErr
	} else {
		defer func() { // If there was no error on return, but the patch fails, set the error accordingly.
			if err := patchHelper.Patch(ctx, csMachine); retErr == nil && err != nil {
				retErr = err
			}
		}()
	}

	// Fetch the CAPI Machine.
	machine, retErr := util.GetOwnerMachine(ctx, r.Client, csMachine.ObjectMeta)
	if retErr != nil {
		return ctrl.Result{}, retErr
	} else if machine == nil {
		log.Info("Waiting for CAPI cluster controller to set owner reference on CloudStack machine.")
		return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
	}

	// Fetch the CAPI Cluster.
	cluster, retErr := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if retErr != nil {
		log.Info("Machine is missing cluster label or cluster does not exist.")
		return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
	}

	// Check the machine is not paused.
	if annotations.IsPaused(cluster, csMachine) {
		log.Info("CloudStackMachine or linked Cluster is paused. Requeuing reconcile.")
		return reconcile.Result{}, nil
	}

	// Delete VM instance if deletion timestamp present.
	if !csMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(log, csMachine)
	}

	// Fetch the CloudStack cluster associated with this machine.
	csCluster := &infrav1.CloudStackCluster{}
	if retErr := r.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: csMachine.Namespace,
			Name:      cluster.Spec.InfrastructureRef.Name},
		csCluster); retErr != nil {
		if client.IgnoreNotFound(retErr) == nil {
			log.Info("CloudStackCluster not found.")
			return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
		} else {
			return ctrl.Result{}, retErr
		}
	} else if csCluster.Status.ZoneID == "" {
		log.Info("CloudStackCluster ZoneId not initialized. Likely not ready.")
		return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
	}

	// Reconcile a VM instance for creates/updates
	return r.reconcile(log, ctx, csCluster, csMachine, machine)
}

// Actually reconcile/Create a VM instance.
func (r *CloudStackMachineReconciler) reconcile(
	log logr.Logger,
	ctx context.Context,
	csCluster *infrav1.CloudStackCluster,
	csMachine *infrav1.CloudStackMachine,
	machine *capiv1.Machine) (ctrl.Result, error) {

	log.V(1).Info("reconcile CloudStackMachine")

	// Make sure bootstrap data is available in CAPI machine.
	if machine.Spec.Bootstrap.DataSecretName == nil {
		log.Info("Bootstrap DataSecretName not yet available.")
		return ctrl.Result{}, nil
	}
	log.Info("Got Bootstrap DataSecretName: " + *machine.Spec.Bootstrap.DataSecretName)

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: machine.Namespace, Name: *machine.Spec.Bootstrap.DataSecretName}
	if err := r.Client.Get(context.TODO(), key, secret); err != nil {
		return ctrl.Result{}, err
	}

	value, ok := secret.Data["value"]
	if !ok {
		return ctrl.Result{}, errors.New("Bootstrap secret data not ok.")
	}

	buf := &bytes.Buffer{}
	err := cloud.CompressData(value, buf)
	if err != nil {
		log.Error(err, "Failed to compress userData")
		return ctrl.Result{}, err
	}
	compressedValue := buf.Bytes()
	userData := base64.StdEncoding.EncodeToString(compressedValue)

	// Create VM (or Fetch if present). Will set ready to true.
	if err := r.CS.GetOrCreateVMInstance(csMachine, machine, csCluster, userData); err == nil {
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
		if err := r.Client.Delete(ctx, machine); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
	} else {
		log.Info(fmt.Sprintf("Instance not ready, is %s.", csMachine.Status.InstanceState))
		return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
	}

	if util.IsControlPlaneMachine(machine) && csCluster.Status.NetworkType != cloud.NetworkTypeShared {
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
	log logr.Logger,
	csMachine *infrav1.CloudStackMachine) (ctrl.Result, error) {

	log.Info("Deleting instance", "instance-id", *csMachine.Spec.InstanceID)
	if err := r.CS.DestroyVMInstance(csMachine); err != nil {
		return ctrl.Result{}, err
	}
	controllerutil.RemoveFinalizer(csMachine, infrav1.MachineFinalizer)
	return ctrl.Result{}, nil
}

// Called in main, this registers the machine reconciler to the CAPI controller manager.
func (r *CloudStackMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {

	controller, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackMachine{}).
		WithEventFilter(
			predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldMachine := e.ObjectOld.(*infrav1.CloudStackMachine).DeepCopy()
					newMachine := e.ObjectNew.(*infrav1.CloudStackMachine).DeepCopy()
					// Ignore resource version because they are unique
					oldMachine.ObjectMeta.ResourceVersion = ""
					newMachine.ObjectMeta.ResourceVersion = ""
					// Ignore generation because it's not used in reconcile
					oldMachine.ObjectMeta.Generation = 0
					newMachine.ObjectMeta.Generation = 0
					// Ignore finalizers updates
					oldMachine.ObjectMeta.Finalizers = nil
					newMachine.ObjectMeta.Finalizers = nil
					// Ignore ManagedFields because they are mirror of ObjectMeta
					oldMachine.ManagedFields = nil
					newMachine.ManagedFields = nil
					// Ignore incremental status updates
					oldMachine.Status = infrav1.CloudStackMachineStatus{}
					newMachine.Status = infrav1.CloudStackMachineStatus{}
					// Ignore provide ID
					oldMachine.Spec.ProviderID = nil
					newMachine.Spec.ProviderID = nil
					// Ignore instance ID
					oldMachine.Spec.InstanceID = nil
					newMachine.Spec.InstanceID = nil

					return !reflect.DeepEqual(oldMachine, newMachine)
				},
			},
		).
		Build(r)

	if err != nil {
		return err
	}

	// Watch CAPI machines for changes.
	// Queues a reconcile request for owned CloudStackMachine on change.
	// Used to update when bootstrap data becomes available.
	if err = controller.Watch(
		&source.Kind{Type: &capiv1.Machine{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("CloudStackMachine")),
		},
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

	csMachineMapper, err := util.ClusterToObjectsMapper(r.Client, &infrav1.CloudStackMachineList{}, mgr.GetScheme())
	if err != nil {
		return err
	}

	// Add a watch on CAPI Cluster objects for unpause and ready events.
	return controller.Watch(
		&source.Kind{Type: &capiv1.Cluster{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: csMachineMapper},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldCluster := e.ObjectOld.(*capiv1.Cluster)
				newCluster := e.ObjectNew.(*capiv1.Cluster)
				return oldCluster.Spec.Paused && !newCluster.Spec.Paused
			},
			CreateFunc: func(e event.CreateEvent) bool {
				_, ok := e.Meta.GetAnnotations()[capiv1.PausedAnnotation]
				return ok
			},
		},
	)
}
