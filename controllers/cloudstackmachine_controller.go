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
	"regexp"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var (
	hostnameMatcher      = regexp.MustCompile(`\{\{\s*ds\.meta_data\.hostname\s*\}\}`)
	failuredomainMatcher = regexp.MustCompile(`\{\{\s*ds\.meta_data\.failuredomain\s*\}\}`)
)

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinesets,verbs=get;list;watch
// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubeadmcontrolplanes,verbs=get;list;watch

// CloudStackMachineReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStack machine reconciliation.
type CloudStackMachineReconciliationRunner struct {
	*utils.ReconciliationRunner
	ReconciliationSubject *infrav1.CloudStackMachine
	CAPIMachine           *clusterv1.Machine
	StateChecker          *infrav1.CloudStackMachineStateChecker
	FailureDomain         *infrav1.CloudStackFailureDomain
	IsoNet                *infrav1.CloudStackIsolatedNetwork
	AffinityGroup         *infrav1.CloudStackAffinityGroup
}

// CloudStackMachineReconciler reconciles a CloudStackMachine object
type CloudStackMachineReconciler struct {
	utils.ReconcilerBase
}

// Initialize a new CloudStackMachine reconciliation runner with concrete types and initialized member fields.
func NewCSMachineReconciliationRunner() *CloudStackMachineReconciliationRunner {
	// Set concrete type and init pointers.
	r := &CloudStackMachineReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackMachine{}}
	r.CAPIMachine = &clusterv1.Machine{}
	r.StateChecker = &infrav1.CloudStackMachineStateChecker{}
	r.IsoNet = &infrav1.CloudStackIsolatedNetwork{}
	r.AffinityGroup = &infrav1.CloudStackAffinityGroup{}
	r.FailureDomain = &infrav1.CloudStackFailureDomain{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	r.ReconciliationRunner = utils.NewRunner(r, r.ReconciliationSubject, "CloudStackMachine")
	return r
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (reconciler *CloudStackMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	r := NewCSMachineReconciliationRunner()
	r.UsingBaseReconciler(reconciler.ReconcilerBase).ForRequest(req).WithRequestCtx(ctx)
	r.WithAdditionalCommonStages(
		r.GetParent(r.ReconciliationSubject, r.CAPIMachine),
		r.RequeueIfCloudStackClusterNotReady,
		r.SetFailureDomainOnCSMachine,
		r.GetFailureDomainByName(func() string { return r.ReconciliationSubject.Spec.FailureDomainName }, r.FailureDomain),
		r.AsFailureDomainUser(&r.FailureDomain.Spec))
	return r.RunBaseReconciliationStages()
}

func (r *CloudStackMachineReconciliationRunner) Reconcile() (retRes ctrl.Result, reterr error) {
	return r.RunReconciliationStages(
		r.GetObjectByName("placeholder", r.IsoNet,
			func() string { return r.IsoNetMetaName(r.FailureDomain.Spec.Zone.Network.Name) }),
		r.RunIf(func() bool { return r.FailureDomain.Spec.Zone.Network.Type == cloud.NetworkTypeIsolated },
			r.CheckPresent(map[string]client.Object{"CloudStackIsolatedNetwork": r.IsoNet})),
		r.ConsiderAffinity,
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

	agName, err := utils.GenerateAffinityGroupName(*r.ReconciliationSubject, r.CAPIMachine)
	if err != nil {
		r.Log.Info("getting affinity group name", err)
	}

	// Set failure domain name and owners.
	r.AffinityGroup.Spec.FailureDomainName = r.ReconciliationSubject.Spec.FailureDomainName
	if res, err := r.GetOrCreateAffinityGroup(
		agName, r.ReconciliationSubject.Spec.Affinity, r.AffinityGroup, r.FailureDomain)(); r.ShouldReturn(res, err) {
		return res, err
	}
	if !r.AffinityGroup.Status.Ready {
		return r.RequeueWithMessage("Required affinity group not ready.")
	}

	return ctrl.Result{}, nil
}

// SetFailureDomainOnCSMachine sets the failure domain the machine should launch in.
func (r *CloudStackMachineReconciliationRunner) SetFailureDomainOnCSMachine() (retRes ctrl.Result, reterr error) {
	if r.ReconciliationSubject.Spec.FailureDomainName == "" {
		var name string
		if r.CAPIMachine.Spec.FailureDomain != nil &&
			(util.IsControlPlaneMachine(r.CAPIMachine) || // Is control plane machine -- CAPI will specify.
				*r.CAPIMachine.Spec.FailureDomain != "") { // Or potentially another machine controller specified.
			name = *r.CAPIMachine.Spec.FailureDomain
			r.ReconciliationSubject.Spec.FailureDomainName = *r.CAPIMachine.Spec.FailureDomain
		} else { // Not a control plane machine. Place randomly.
			randNum := (rand.Int() % len(r.CSCluster.Spec.FailureDomains)) // #nosec G404 -- weak crypt rand doesn't matter here.
			name = r.CSCluster.Spec.FailureDomains[randNum].Name
		}
		r.ReconciliationSubject.Spec.FailureDomainName = name
		r.ReconciliationSubject.Labels[infrav1.FailureDomainLabelName] = infrav1.FailureDomainHashedMetaName(name, r.CAPICluster.Name)
	}
	return ctrl.Result{}, nil
}

// GetOrCreateVMInstance gets or creates a VM instance.
// Implicitly it also fetches its bootstrap secret in order to create said instance.
func (r *CloudStackMachineReconciliationRunner) GetOrCreateVMInstance() (retRes ctrl.Result, reterr error) {
	if r.CAPIMachine.Spec.Bootstrap.DataSecretName == nil {
		return r.RequeueWithMessage("Bootstrap DataSecretName not yet available.")
	}
	r.Log.Info("Got Bootstrap DataSecretName.")

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

	userData := processUserData(data, r)
	err := r.CSUser.GetOrCreateVMInstance(r.ReconciliationSubject, r.CAPIMachine, r.CSCluster, r.FailureDomain, r.AffinityGroup, userData)

	if err == nil && !controllerutil.ContainsFinalizer(r.ReconciliationSubject, infrav1.MachineFinalizer) { // Fetched or Created?
		r.Log.Info("CloudStack instance Created", "instanceStatus", r.ReconciliationSubject.Status)
	}
	// Always add the finalizer regardless. It can't be added twice anyway.
	controllerutil.AddFinalizer(r.ReconciliationSubject, infrav1.MachineFinalizer)
	return ctrl.Result{}, err
}

func processUserData(data []byte, r *CloudStackMachineReconciliationRunner) string {
	// since cloudstack metadata does not allow custom data added into meta_data, following line is a workaround to specify a hostname name
	// {{ ds.meta_data.hostname }} is expected to be used as a name when kubelet register a node
	// if more custom data needed to injected, this can be refactored into a method -- processCustomMetadata()
	userData := hostnameMatcher.ReplaceAllString(string(data), r.CAPIMachine.Name)
	userData = failuredomainMatcher.ReplaceAllString(userData, r.FailureDomain.Spec.Name)
	return userData
}

// ConfirmVMStatus checks the Instance's status for running state and requeues otherwise.
func (r *CloudStackMachineReconciliationRunner) RequeueIfInstanceNotRunning() (retRes ctrl.Result, reterr error) {
	if r.ReconciliationSubject.Status.InstanceState == "Running" {
		r.Log.Info("Machine instance is Running...")
		r.ReconciliationSubject.Status.Ready = true
	} else if r.ReconciliationSubject.Status.InstanceState == "Error" {
		r.Log.Info("CloudStackMachine VM in error state. Deleting associated Machine.", "csMachine", r.ReconciliationSubject.GetName())
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
	if util.IsControlPlaneMachine(r.CAPIMachine) && r.FailureDomain.Spec.Zone.Network.Type == cloud.NetworkTypeIsolated {
		r.Log.Info("Assigning VM to load balancer rule.")
		if r.IsoNet.Spec.Name == "" {
			return r.RequeueWithMessage("Could not get required Isolated Network for VM, requeueing.")
		}
		err := r.CSUser.AssignVMToLoadBalancerRule(r.IsoNet, *r.ReconciliationSubject.Spec.InstanceID)
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

	if err := r.K8sClient.Create(r.RequestCtx, csMachineStateChecker); err != nil && !utils.ContainsAlreadyExistsSubstring(err) {
		return r.ReturnWrappedError(err, "error encountered when creating CloudStackMachineStateChecker")
	}

	return r.GetObjectByName(*checkerName, r.StateChecker)()
}

func (r *CloudStackMachineReconciliationRunner) ReconcileDelete() (retRes ctrl.Result, reterr error) {
	if r.ReconciliationSubject.Spec.InstanceID != nil {
		r.Log.Info("Deleting instance", "instance-id", r.ReconciliationSubject.Spec.InstanceID)
		// Use CSClient instead of CSUser here to expunge as admin.
		// The CloudStack-Go API does not return an error, but the VM won't delete with Expunge set if requested by
		// non-domain admin user.
		if err := r.CSClient.DestroyVMInstance(r.ReconciliationSubject); err != nil {
			if err.Error() == "VM deletion in progress" {
				r.Log.Info(err.Error())
				return ctrl.Result{RequeueAfter: utils.DestoryVMRequeueInterval}, nil
			}
			return ctrl.Result{}, err
		}
	}
	r.Log.Info("VM Deleted")
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
		&source.Kind{Type: &clusterv1.Machine{}},
		handler.EnqueueRequestsFromMapFunc(
			util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("CloudStackMachine"))),
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldMachine := e.ObjectOld.(*clusterv1.Machine)
				newMachine := e.ObjectNew.(*clusterv1.Machine)

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
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(csMachineMapper),
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldCluster := e.ObjectOld.(*clusterv1.Cluster)
				newCluster := e.ObjectNew.(*clusterv1.Cluster)
				return oldCluster.Spec.Paused && !newCluster.Spec.Paused
			},
			CreateFunc: func(e event.CreateEvent) bool {
				_, ok := e.Object.GetAnnotations()[clusterv1.PausedAnnotation]
				return ok
			},
		},
	)
}
