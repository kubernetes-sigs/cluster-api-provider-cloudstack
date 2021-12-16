/*
Copyright 2021.

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
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	infrav1 "cluster.x-k8s.io/cluster-api-provider-capc/api/v1alpha3"
	"cluster.x-k8s.io/cluster-api-provider-capc/pkg/cloud"
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

	// Fetch the CloudStackMachine.
	csMachine := &infrav1.CloudStackMachine{}
	if retErr = r.Client.Get(ctx, req.NamespacedName, csMachine); retErr != nil {
		if client.IgnoreNotFound(retErr) == nil {
			log.Info("Machine not found.")
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

	// Fetch the CloudStack cluster associated with this machine.
	csCluster := &infrav1.CloudStackCluster{}
	if retErr := r.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: csMachine.Namespace,
			Name:      cluster.Spec.InfrastructureRef.Name},
		csCluster); retErr != nil {
		if client.IgnoreNotFound(retErr) == nil {
			return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
		} else {
			return ctrl.Result{}, retErr
		}
	} else if csCluster.Status.ZoneID == "" {
		log.Info("Cluster not found. Likely not ready.")
		return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
	}

	// Delete VM instance if deletion timestamp present.
	if !csMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(log, csMachine)
	}

	// Otherwise reconcile a VM instance.
	return r.reconcile(log, csCluster, csMachine, machine)
}

// Actually reconcile/Create a VM instance.
func (r *CloudStackMachineReconciler) reconcile(
	log logr.Logger,
	csCluster *infrav1.CloudStackCluster,
	csMachine *infrav1.CloudStackMachine,
	machine *capiv1.Machine) (ctrl.Result, error) {

	// Make sure bootstrap data is available in CAPI machine.
	if machine.Spec.Bootstrap.DataSecretName == nil {
		log.Info("Bootstrap DataSecretName not yet available.")
		return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
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
	userData := base64.StdEncoding.EncodeToString(value)

	// Create machine (or Fetch if present). Will set ready to true.
	if err := r.CS.GetOrCreateVMInstance(csMachine, machine, csCluster, userData); err == nil {
		if !controllerutil.ContainsFinalizer(csMachine, infrav1.MachineFinalizer) { // Fetched or Created?
			log.Info("Machine Created", "instanceStatus", csMachine.Status, "instanceSpec", csMachine.Spec)
			controllerutil.AddFinalizer(csMachine, infrav1.MachineFinalizer)
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	if util.IsControlPlaneMachine(machine) && csCluster.Status.NetworkType != cloud.NetworkTypeShared {
		log.Info("Assigning VM to load balancer rule.")
		err := r.CS.AssignVMToLoadBalancerRule(csCluster, *csMachine.Spec.InstanceID)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if csMachine.Status.InstanceState == "Running" {
		log.Info("Machine instance is Running...")
		csMachine.Status.Ready = true
	} else {
		log.Info(fmt.Sprintf("Instance not ready, is %s.", csMachine.Status.InstanceState))
		return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
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
	return ctrl.NewControllerManagedBy(mgr).
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
		Complete(r)
}
