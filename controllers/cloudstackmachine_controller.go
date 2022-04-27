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
	"math/rand"
	"reflect"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinesets,verbs=get;list;watch
// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubeadmcontrolplanes,verbs=get;list;watch

// CloudStackMachineReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackCluster reconciliation.
type CloudStackMachineReconciliationRunner struct {
	csCtrlrUtils.ReconciliationRunner
	ReconciliationSubject *infrav1.CloudStackMachine
	CAPIMachine           *capiv1.Machine
	CSUser                cloud.Client
	Zones                 *infrav1.CloudStackZoneList
	FailureDomain         *infrav1.CloudStackZone
	IsoNet                *infrav1.CloudStackIsolatedNetwork
}

// CloudStackMachineReconciler reconciles a CloudStackMachine object
type CloudStackMachineReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

// Initialize a new CloudStackMachine reconciliation runner with concrete types and initialized member fields.
func NewCSMachineReconciliationRunner() *CloudStackMachineReconciliationRunner {
	// Set concrete type and init pointers.
	runner := &CloudStackMachineReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackMachine{}}
	runner.CAPIMachine = &capiv1.Machine{}
	runner.Zones = &infrav1.CloudStackZoneList{}
	runner.IsoNet = &infrav1.CloudStackIsolatedNetwork{}
	runner.FailureDomain = &infrav1.CloudStackZone{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	runner.ReconciliationRunner = csCtrlrUtils.NewRunner(runner, runner.ReconciliationSubject)
	return runner
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (reconciler *CloudStackMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	return NewCSMachineReconciliationRunner().
		UsingBaseReconciler(reconciler.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunBaseReconciliationStages()
}

func (r *CloudStackMachineReconciliationRunner) Reconcile() (retRes ctrl.Result, reterr error) {
	return r.RunReconciliationStages(
		r.GetZones(r.Zones),
		r.GetParent(r.ReconciliationSubject, r.CAPIMachine),
		r.RequeueIfCloudStackClusterNotReady,
		r.SetFailureDomainOnCSMachine,
		r.GetObjectByName("placeholder", r.IsoNet, func() string { return r.FailureDomain.Spec.Network.Name }),
		// This can move to IsoNet controller with a nice watch someday.
		r.GetOrCreateVMInstance,
		r.RequeueIfInstanceNotRunning,
		r.AddToLBIfNeeded)
}

// SetFailureDomainOnCSMachine sets the failure domain the machine should launch in.
func (runner *CloudStackMachineReconciliationRunner) SetFailureDomainOnCSMachine() (retRes ctrl.Result, reterr error) {
	// Set ZoneID on csMachine.
	if util.IsControlPlaneMachine(runner.CAPIMachine) { // Use failure domain zone.
		runner.ReconciliationSubject.Status.ZoneID = *runner.CAPIMachine.Spec.FailureDomain
	} else { // Specified by Machine Template or Random zone.
		if runner.ReconciliationSubject.Spec.ZoneID != "" {
			if zone, foundZone := runner.CSCluster.Status.Zones[runner.ReconciliationSubject.Spec.ZoneID]; foundZone { // ZoneID Specified.
				runner.ReconciliationSubject.Status.ZoneID = zone.ID
			} else {
				return ctrl.Result{}, errors.Errorf("could not find zone by zoneID: %s", runner.ReconciliationSubject.Spec.ZoneID)
			}
		} else if runner.ReconciliationSubject.Spec.ZoneName != "" {
			if zone := runner.CSCluster.Status.Zones.GetByName(runner.ReconciliationSubject.Spec.ZoneName); zone != nil { // ZoneName Specified.
				runner.ReconciliationSubject.Status.ZoneID = zone.ID
			} else {
				return ctrl.Result{}, errors.Errorf("could not find zone by zoneName: %s", runner.ReconciliationSubject.Spec.ZoneName)
			}
		} else { // No Zone Specified, pick a Random Zone.
			randNum := (rand.Int() % len(runner.Zones.Items)) // #nosec G404 -- weak crypt rand doesn't matter here.
			runner.ReconciliationSubject.Status.ZoneID = runner.Zones.Items[randNum].Spec.ID
		}
	}
	for _, zone := range runner.Zones.Items {
		runner.FailureDomain = &zone
	}
	fmt.Printf("%+v\n", runner.FailureDomain.Spec.Network)
	return ctrl.Result{}, nil
}

// GetOrCreateVMInstance creates or gets a VM instance.
// Implicitly it also fetches its bootstrap secret in order to create said instance.
func (runner *CloudStackMachineReconciliationRunner) GetOrCreateVMInstance() (retRes ctrl.Result, reterr error) {
	if runner.CAPIMachine.Spec.Bootstrap.DataSecretName == nil {
		return runner.RequeueWithMessage("Bootstrap DataSecretName not yet available.")
	}
	runner.Log.Info("Got Bootstrap DataSecretName.")

	// Get the CloudStackZone for the Machine.
	var machineZone infrav1.CloudStackZone
	for _, zone := range runner.Zones.Items {
		machineZone = zone
		if zone.Spec.ID == runner.ReconciliationSubject.Status.ZoneID {
			break
		}
	}

	// Get the kubeadm bootstrap secret for this machine.
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: runner.CAPIMachine.Namespace, Name: *runner.CAPIMachine.Spec.Bootstrap.DataSecretName}
	if err := runner.Client.Get(context.TODO(), key, secret); err != nil {
		return ctrl.Result{}, err
	}
	if data, isPresent := secret.Data["value"]; isPresent {
		if err := runner.CS.GetOrCreateVMInstance(
			runner.ReconciliationSubject, runner.CAPIMachine, runner.CSCluster, &machineZone, string(data)); err == nil {
			if !controllerutil.ContainsFinalizer(runner.ReconciliationSubject, infrav1.MachineFinalizer) { // Fetched or Created?
				runner.Log.Info("CloudStack instance Created", "instanceStatus", runner.ReconciliationSubject.Status)
				controllerutil.AddFinalizer(runner.ReconciliationSubject, infrav1.MachineFinalizer)
			}
		} else if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		return ctrl.Result{}, errors.New("bootstrap secret data not yet set")
	}
	return ctrl.Result{}, nil
}

// ConfirmVMStatus checks the Instance's status for running state and requeues otherwise.
func (runner *CloudStackMachineReconciliationRunner) RequeueIfInstanceNotRunning() (retRes ctrl.Result, reterr error) {
	if runner.ReconciliationSubject.Status.InstanceState == "Running" {
		runner.Log.Info("Machine instance is Running...")
		runner.ReconciliationSubject.Status.Ready = true
	} else if runner.ReconciliationSubject.Status.InstanceState == "Error" {
		runner.Log.Info("CloudStackMachine VM in error state. Deleting associated Machine.", "csMachine", runner.ReconciliationSubject)
		if err := runner.Client.Delete(runner.RequestCtx, runner.CAPIMachine); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: utils.RequeueTimeout}, nil
	} else {
		runner.Log.Info(fmt.Sprintf("Instance not ready, is %s.", runner.ReconciliationSubject.Status.InstanceState))
		return ctrl.Result{RequeueAfter: utils.RequeueTimeout}, nil
	}
	return ctrl.Result{}, nil
}

// AddToLBIfNeeded adds instance to load balancer if it is a control plane in an isolated network.
func (runner *CloudStackMachineReconciliationRunner) AddToLBIfNeeded() (retRes ctrl.Result, reterr error) {
	if util.IsControlPlaneMachine(runner.CAPIMachine) && runner.FailureDomain.Spec.Network.Type == cloud.NetworkTypeIsolated {
		runner.Log.Info("Assigning VM to load balancer rule.")
		if runner.IsoNet.Spec.Name == "" {
			return runner.RequeueWithMessage("Could not get required Isolated Network for VM, requeueing.")
		}
		err := runner.CS.AssignVMToLoadBalancerRule(runner.IsoNet, *runner.ReconciliationSubject.Spec.InstanceID)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (runner *CloudStackMachineReconciliationRunner) ReconcileDelete() (retRes ctrl.Result, reterr error) {
	runner.Log.Info("Deleting instance", "instance-id", *&runner.ReconciliationSubject.Spec.InstanceID)
	if err := runner.CS.DestroyVMInstance(runner.ReconciliationSubject); err != nil {
		if err.Error() == "VM deletion in progress" {
			runner.Log.Info(err.Error())
			return ctrl.Result{RequeueAfter: utils.DestoryVMRequeueInterval}, nil
		}
		return ctrl.Result{}, err
	}
	controllerutil.RemoveFinalizer(runner.ReconciliationSubject, infrav1.MachineFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager registers the machine reconciler to the CAPI controller manager.
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

	// Used below, this maps CAPI clusters to CAPC machines
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
