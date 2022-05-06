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

// CloudStackMachineReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStack machine reconciliation.
type CloudStackMachineReconciliationRunner struct {
	csCtrlrUtils.ReconciliationRunner
	ReconciliationSubject *infrav1.CloudStackMachine
	CAPIMachine           *capiv1.Machine
	StateChecker          *infrav1.CloudStackMachineStateChecker
	CSUser                cloud.Client
	Zones                 *infrav1.CloudStackZoneList
	FailureDomain         *infrav1.CloudStackZone
	IsoNet                *infrav1.CloudStackIsolatedNetwork
	AffinityGroup         *infrav1.CloudStackAffinityGroup
}

// CloudStackMachineReconciler reconciles a CloudStackMachine object
type CloudStackMachineReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

// Initialize a new CloudStackMachine reconciliation runner with concrete types and initialized member fields.
func NewCSMachineReconciliationRunner() *CloudStackMachineReconciliationRunner {
	// Set concrete type and init pointers.
	r := &CloudStackMachineReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackMachine{}}
	r.CAPIMachine = &capiv1.Machine{}
	r.StateChecker = &infrav1.CloudStackMachineStateChecker{}
	r.Zones = &infrav1.CloudStackZoneList{}
	r.IsoNet = &infrav1.CloudStackIsolatedNetwork{}
	r.AffinityGroup = &infrav1.CloudStackAffinityGroup{}
	r.FailureDomain = &infrav1.CloudStackZone{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	r.ReconciliationRunner = csCtrlrUtils.NewRunner(r, r.ReconciliationSubject)
	return r
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
		r.ConsiderAffinity,
		r.SetFailureDomainOnCSMachine,
		r.GetObjectByName("placeholder", r.IsoNet, func() string { return r.FailureDomain.Spec.Network.Name }),
		r.GetOrCreateVMInstance,
		r.RequeueIfInstanceNotRunning,
		r.AddToLBIfNeeded,
		r.GetOrCreateMachineStateChecker,
	)
}

// ConsiderAffinity sets machine affinity if needed. It also creates or gets an affinity group CRD if required and
// checks it for readiness.
func (r *CloudStackMachineReconciliationRunner) ConsiderAffinity() (ctrl.Result, error) {
	if r.ReconciliationSubject.Spec.Affinity == infrav1.NoAffinity ||
		r.ReconciliationSubject.Spec.Affinity == "" { // No managed affinity.
		return ctrl.Result{}, nil
	}

	agName, err := csCtrlrUtils.GenerateAffinityGroupName(*r.ReconciliationSubject, r.CAPIMachine)
	if err != nil {
		r.Log.Info("getting affinity group name:", err)
	}

	if res, err := r.GetOrCreateAffinityGroup(agName, r.ReconciliationSubject.Spec.Affinity, r.AffinityGroup)(); r.ShouldReturn(res, err) {
		return res, err
	}
	if !r.AffinityGroup.Status.Ready {
		return r.RequeueWithMessage("Required afinity group not ready.")
	}
	return ctrl.Result{}, nil
}

// SetFailureDomainOnCSMachine sets the failure domain the machine should launch in.
func (r *CloudStackMachineReconciliationRunner) SetFailureDomainOnCSMachine() (retRes ctrl.Result, reterr error) {
	// Set ZoneID on csMachine.
	if util.IsControlPlaneMachine(r.CAPIMachine) { // Use failure domain zone.
		r.ReconciliationSubject.Status.ZoneID = *r.CAPIMachine.Spec.FailureDomain
	} else { // Specified by Machine Template or Random zone.
		if r.ReconciliationSubject.Spec.ZoneID != "" {
			if zone, foundZone := r.CSCluster.Status.Zones[r.ReconciliationSubject.Spec.ZoneID]; foundZone { // ZoneID Specified.
				r.ReconciliationSubject.Status.ZoneID = zone.ID
			} else {
				return ctrl.Result{}, errors.Errorf("could not find zone by zoneID: %s", r.ReconciliationSubject.Spec.ZoneID)
			}
		} else if r.ReconciliationSubject.Spec.ZoneName != "" {
			for _, zone := range r.Zones.Items {
				if zone.Spec.Name == r.ReconciliationSubject.Spec.ZoneName {
					r.ReconciliationSubject.Status.ZoneID = zone.Spec.ID
					break
				}
			}
			if r.ReconciliationSubject.Status.ZoneID == "" {
				return ctrl.Result{}, errors.Errorf("could not find zone by zoneName: %s", r.ReconciliationSubject.Spec.ZoneName)
			}
		} else { // No Zone Specified, pick a Random Zone.
			randNum := (rand.Int() % len(r.Zones.Items)) // #nosec G404 -- weak crypt rand doesn't matter here.
			r.ReconciliationSubject.Status.ZoneID = r.Zones.Items[randNum].Spec.ID
		}
	}
	for idx, zone := range r.Zones.Items {
		if zone.Spec.ID == r.ReconciliationSubject.Status.ZoneID {
			r.FailureDomain = &r.Zones.Items[idx]
			break
		}
	}
	return ctrl.Result{}, nil
}

// GetOrCreateVMInstance creates or gets a VM instance.
// Implicitly it also fetches its bootstrap secret in order to create said instance.
func (r *CloudStackMachineReconciliationRunner) GetOrCreateVMInstance() (retRes ctrl.Result, reterr error) {
	if r.CAPIMachine.Spec.Bootstrap.DataSecretName == nil {
		return r.RequeueWithMessage("Bootstrap DataSecretName not yet available.")
	}
	r.Log.Info("Got Bootstrap DataSecretName.")

	// Get the CloudStackZone for the Machine.
	var machineZone infrav1.CloudStackZone
	for _, zone := range r.Zones.Items {
		machineZone = zone
		if zone.Spec.ID == r.ReconciliationSubject.Status.ZoneID {
			break
		}
	}

	// Get the kubeadm bootstrap secret for this machine.
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: r.CAPIMachine.Namespace, Name: *r.CAPIMachine.Spec.Bootstrap.DataSecretName}
	if err := r.K8sClient.Get(context.TODO(), key, secret); err != nil {
		return ctrl.Result{}, err
	}
	data, present := secret.Data["value"]
	if !present {
		return ctrl.Result{}, errors.New("bootstrap secret data not yet set")
	}

	err := r.CSClient.GetOrCreateVMInstance(r.ReconciliationSubject, r.CAPIMachine, r.CSCluster, &machineZone, r.AffinityGroup, string(data))

	if err == nil && !controllerutil.ContainsFinalizer(r.ReconciliationSubject, infrav1.MachineFinalizer) { // Fetched or Created?
		r.Log.Info("CloudStack instance Created", "instanceStatus", r.ReconciliationSubject.Status)
		controllerutil.AddFinalizer(r.ReconciliationSubject, infrav1.MachineFinalizer)
	}
	return ctrl.Result{}, err
}

// ConfirmVMStatus checks the Instance's status for running state and requeues otherwise.
func (r *CloudStackMachineReconciliationRunner) RequeueIfInstanceNotRunning() (retRes ctrl.Result, reterr error) {
	if r.ReconciliationSubject.Status.InstanceState == "Running" {
		r.Log.Info("Machine instance is Running...")
		r.ReconciliationSubject.Status.Ready = true
	} else if r.ReconciliationSubject.Status.InstanceState == "Error" {
		r.Log.Info("CloudStackMachine VM in error state. Deleting associated Machine.", "csMachine", r.ReconciliationSubject)
		if err := r.K8sClient.Delete(r.RequestCtx, r.CAPIMachine); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: utils.RequeueTimeout}, nil
	} else {
		r.Log.Info(fmt.Sprintf("Instance not ready, is %s.", r.ReconciliationSubject.Status.InstanceState))
		return ctrl.Result{RequeueAfter: utils.RequeueTimeout}, nil
	}
	return ctrl.Result{}, nil
}

// AddToLBIfNeeded adds instance to load balancer if it is a control plane in an isolated network.
func (r *CloudStackMachineReconciliationRunner) AddToLBIfNeeded() (retRes ctrl.Result, reterr error) {
	if util.IsControlPlaneMachine(r.CAPIMachine) && r.FailureDomain.Spec.Network.Type == cloud.NetworkTypeIsolated {
		r.Log.Info("Assigning VM to load balancer rule.")
		if r.IsoNet.Spec.Name == "" {
			return r.RequeueWithMessage("Could not get required Isolated Network for VM, requeueing.")
		}
		err := r.CSClient.AssignVMToLoadBalancerRule(r.IsoNet, *r.ReconciliationSubject.Spec.InstanceID)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// GetOrCreateMachineStateChecker creates or gets CloudStackMachineStateChecker object.
func (r *CloudStackMachineReconciliationRunner) GetOrCreateMachineStateChecker() (retRes ctrl.Result, reterr error) {
	checkerName := r.ReconciliationSubject.Spec.InstanceID
	csMachineStateChecker := &infrav1.CloudStackMachineStateChecker{
		ObjectMeta: r.NewChildObjectMeta(*checkerName),
		Spec:       infrav1.CloudStackMachineStateCheckerSpec{InstanceID: *checkerName},
		Status:     infrav1.CloudStackMachineStateCheckerStatus{Ready: false},
	}

	if err := r.K8sClient.Create(r.RequestCtx, csMachineStateChecker); err != nil && !csCtrlrUtils.ContainsAlreadyExistsSubstring(err) {
		return r.ReturnWrappedError(err, "error encountered when creating CloudStackMachineStateChecker")
	}

	return r.GetObjectByName(*checkerName, r.StateChecker)()
}

func (r *CloudStackMachineReconciliationRunner) ReconcileDelete() (retRes ctrl.Result, reterr error) {
	r.Log.Info("Deleting instance", "instance-id", r.ReconciliationSubject.Spec.InstanceID)
	if err := r.CSClient.DestroyVMInstance(r.ReconciliationSubject); err != nil {
		if err.Error() == "VM deletion in progress" {
			r.Log.Info(err.Error())
			return ctrl.Result{RequeueAfter: utils.DestoryVMRequeueInterval}, nil
		}
		return ctrl.Result{}, err
	}
	controllerutil.RemoveFinalizer(r.ReconciliationSubject, infrav1.MachineFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager registers the machine reconciler to the CAPI controller manager.
func (reconciler *CloudStackMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {

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
		).Build(reconciler)
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
	csMachineMapper, err := util.ClusterToObjectsMapper(reconciler.K8sClient, &infrav1.CloudStackMachineList{}, mgr.GetScheme())
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
