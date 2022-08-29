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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"time"

	"github.com/pkg/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
)

// CloudStackFailureDomainReconciler is the k8s controller manager's interface to reconcile a CloudStackFailureDomain.
// This is primarily to adapt to k8s.
type CloudStackFailureDomainReconciler struct {
	csCtrlrUtils.ReconcilerBase
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinedeployments,verbs=get;list;watch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplates,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=etcdcluster.cluster.x-k8s.io,resources=etcdadmclusters,verbs=get;list;watch;patch

// CloudStackFailureDomainReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackFailureDomains.
// The runner does the actual reconciliation.
type CloudStackFailureDomainReconciliationRunner struct {
	*csCtrlrUtils.ReconciliationRunner
	ReconciliationSubject *infrav1.CloudStackFailureDomain
	IsoNet                *infrav1.CloudStackIsolatedNetwork
}

// Initialize a new CloudStackFailureDomain reconciliation runner with concrete types and initialized member fields.
func NewCSFailureDomainReconciliationRunner() *CloudStackFailureDomainReconciliationRunner {
	// Set concrete type and init pointers.
	r := &CloudStackFailureDomainReconciliationRunner{ReconciliationSubject: &infrav1.CloudStackFailureDomain{}}
	r.IsoNet = &infrav1.CloudStackIsolatedNetwork{}
	// Setup the base runner. Initializes pointers and links reconciliation methods.
	r.ReconciliationRunner = csCtrlrUtils.NewRunner(r, r.ReconciliationSubject, "CloudStackFailureDomain")

	return r
}

// Reconcile is the method k8s will call upon a reconciliation request.
func (reconciler *CloudStackFailureDomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	return NewCSFailureDomainReconciliationRunner().
		UsingBaseReconciler(reconciler.ReconcilerBase).
		ForRequest(req).
		WithRequestCtx(ctx).
		RunBaseReconciliationStages()
}

// Reconcile on the ReconciliationRunner actually attempts to modify or create the reconciliation subject.
func (r *CloudStackFailureDomainReconciliationRunner) Reconcile() (retRes ctrl.Result, retErr error) {
	res, err := r.AsFailureDomainUser(&r.ReconciliationSubject.Spec)()
	if r.ShouldReturn(res, err) {
		return res, err
	}
	// Prevent premature deletion.
	controllerutil.AddFinalizer(r.ReconciliationSubject, infrav1.FailureDomainFinalizer)

	// Start by purely data fetching information about the zone and specified network.
	if err := r.CSUser.ResolveZone(&r.ReconciliationSubject.Spec.Zone); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "resolving CloudStack zone information")
	}
	if err := r.CSUser.ResolveNetworkForZone(&r.ReconciliationSubject.Spec.Zone); err != nil &&
		!csCtrlrUtils.ContainsNoMatchSubstring(err) {
		return ctrl.Result{}, errors.Wrap(err, "resolving Cloudstack network information")
	}

	// Check if the passed network was an isolated network or the network was missing. In either case, create a
	// CloudStackIsolatedNetwork to manage the many intricacies and wait until CloudStackIsolatedNetwork is ready.
	if r.ReconciliationSubject.Spec.Zone.Network.ID == "" ||
		r.ReconciliationSubject.Spec.Zone.Network.Type == infrav1.NetworkTypeIsolated {
		netName := r.ReconciliationSubject.Spec.Zone.Network.Name
		if res, err := r.GenerateIsolatedNetwork(
			netName, func() string { return r.ReconciliationSubject.Spec.Name })(); r.ShouldReturn(res, err) {
			return res, err
		} else if res, err := r.GetObjectByName(r.IsoNetMetaName(netName), r.IsoNet)(); r.ShouldReturn(res, err) {
			return res, err
		}
		if r.IsoNet.Name == "" {
			return r.RequeueWithMessage("Couldn't find isolated network.")
		}
		if !r.IsoNet.Status.Ready {
			return r.RequeueWithMessage("Isolated network dependency not ready.")
		}
	}
	r.ReconciliationSubject.Status.Ready = true
	return ctrl.Result{}, nil
}

// ReconcileDelete on the ReconciliationRunner attempts to delete the reconciliation subject.
func (r *CloudStackFailureDomainReconciliationRunner) ReconcileDelete() (ctrl.Result, error) {
	r.Log.Info("Deleting CloudStackFailureDomain")

	return r.RunReconciliationStages(
		r.ClearMachines,
		r.DeleteOwnedObjects(
			infrav1.GroupVersion.WithKind("CloudStackAffinityGroup"),
			infrav1.GroupVersion.WithKind("CloudStackIsolatedNetwork")),
		r.CheckOwnedObjectsDeleted(
			infrav1.GroupVersion.WithKind("CloudStackAffinityGroup"),
			infrav1.GroupVersion.WithKind("CloudStackIsolatedNetwork")),
		r.RemoveFinalizer,
	)
}

// ClearMachines checks for any machines in failure domain, deletes the CAPI machine for any still in FailureDomain,
// and requeus until all CloudStack machines are cleared from the FailureDomain.
func (r *CloudStackFailureDomainReconciliationRunner) ClearMachines() (ctrl.Result, error) {
	machines := &infrav1.CloudStackMachineList{}
	if err := r.K8sClient.List(r.RequestCtx, machines, client.MatchingLabels{infrav1.FailureDomainLabelName: r.ReconciliationSubject.Name}); err != nil {
		return ctrl.Result{}, err
	}
	r.Log.Info(fmt.Sprintf("Clear machine: %d found.", len(machines.Items)))
	if len(machines.Items) > 0 {
		if result, err := checkClusterReady(r); err != nil {
			return result, err
		}
	}

	result, err := triggerEtcdClusterRollout(machines.Items, r)
	if err != nil {
		return result, err
	}

	result, err = triggerControlPlaneRollout(machines.Items, r)
	if err != nil {
		return result, err
	}

	result, err = triggerMachineDeploymentRollout(machines.Items, r)
	if err != nil {
		return result, err
	}

	if len(machines.Items) > 0 {
		return r.RequeueWithMessage("FailureDomain still has machine(s) in it.")
	}
	return ctrl.Result{}, nil
}

func triggerEtcdClusterRollout(machines []infrav1.CloudStackMachine, r *CloudStackFailureDomainReconciliationRunner) (ctrl.Result, error) {
	var etcdMachine infrav1.CloudStackMachine
	etcdMachineFound := false
	for _, machine := range machines {
		for _, ref := range machine.OwnerReferences {
			if ref.Kind == "EtcdadmCluster" {
				etcdMachineFound = true
				etcdMachine = machine
			}
		}
	}
	if !etcdMachineFound {
		r.Log.Info("Clear machine: no etcd VM found.")
		return ctrl.Result{}, nil
	}
	for _, ref := range etcdMachine.OwnerReferences {
		if ref.Kind == "EtcdadmCluster" {
			r.Log.Info("Clear machine: getting etcdadmcluster ...")
			etcdadmCluster := &unstructured.Unstructured{}
			etcdadmCluster.SetGroupVersionKind(schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind))
			if err := r.K8sClient.Get(r.RequestCtx, client.ObjectKey{Namespace: etcdMachine.Namespace, Name: ref.Name}, etcdadmCluster); err != nil {
				return ctrl.Result{}, err
			}
			r.Log.Info("Clear machine: retrieved etcdadmcluster.")
			csMachineTemplateName, ok := etcdMachine.Annotations["cluster.x-k8s.io/cloned-from-name"]
			if !ok {
				result, err := r.RequeueWithMessage("annotations['cluster.x-k8s.io/cloned-from-name'] not found.", "cloudstackmachine", etcdMachine.Name)
				return result, err
			}
			r.Log.Info(fmt.Sprintf("Clear machine: machine template name %s decided", csMachineTemplateName))
			csMachineTemplateNameInEtcdadmCluster, found, err := unstructured.NestedString(etcdadmCluster.Object, "spec", "infrastructureTemplate", "name")
			if err != nil || !found {
				return ctrl.Result{}, errors.New("etcdadmcluster spec.infrastructureTemplate.name not found or not string")
			}
			if csMachineTemplateName != csMachineTemplateNameInEtcdadmCluster {
				return ctrl.Result{}, errors.Errorf("cloudstackmachinetemplate in machine %s and etcdadmcluster %s are different", etcdMachine.Name, ref.Name)
			}
			csMachineTemplate := &infrav1.CloudStackMachineTemplate{}
			if err := r.K8sClient.Get(r.RequestCtx, client.ObjectKey{Namespace: etcdMachine.Namespace, Name: csMachineTemplateName}, csMachineTemplate); err != nil {
				return ctrl.Result{}, err
			}
			r.Log.Info(fmt.Sprintf("Clear machine: machine template name %s retrieved", csMachineTemplateName))
			csMachineTemplate.Name = fmt.Sprintf("%s-template-%d", ref.Name, time.Now().UnixNano()/int64(time.Millisecond))
			csMachineTemplate.SetAnnotations(map[string]string{})
			csMachineTemplate.SetResourceVersion("")
			if err := r.K8sClient.Create(r.RequestCtx, csMachineTemplate); err != nil {
				return ctrl.Result{}, err
			}
			r.Log.Info(fmt.Sprintf("Clear machine: new machine template name %s created", csMachineTemplate.Name))

			if _, err := controllerutil.CreateOrPatch(r.RequestCtx, r.K8sClient, etcdadmCluster,
				func() error {
					return unstructured.SetNestedField(
						etcdadmCluster.Object,
						csMachineTemplate.Name,
						"spec", "infrastructureTemplate", "name",
					)
				}); err != nil {
				return ctrl.Result{}, err
			}
			r.Log.Info(fmt.Sprintf("Clear machine: etcdadmcluster infrastructureTemplate cloudstackmachinetemplate name %s patched", csMachineTemplate.Name))
		}
	}
	return ctrl.Result{}, nil
}

func triggerMachineDeploymentRollout(machines []infrav1.CloudStackMachine, r *CloudStackFailureDomainReconciliationRunner) (ctrl.Result, error) {
	var workerMachine infrav1.CloudStackMachine
	workerMachineFound := false
	for _, machine := range machines {
		for _, ref := range machine.OwnerReferences {
			if ref.Kind == "MachineSet" {
				workerMachineFound = true
				workerMachine = machine
			}
		}
	}
	if !workerMachineFound {
		r.Log.Info("Clear machine: no worker VM found.")
		return ctrl.Result{}, nil
	}
	for _, ref := range workerMachine.OwnerReferences {
		if ref.Kind == "MachineSet" {
			r.Log.Info("Clear machine: getting machine deployment...")
			md := &clusterv1.MachineDeployment{}
			mdName, ok := workerMachine.Labels[clusterv1.MachineDeploymentLabelName]
			if !ok {
				result, err := r.RequeueWithMessage("cloudstack machine misses label: " + clusterv1.MachineDeploymentLabelName, "cloudstackmachine", r.ReconciliationSubject.Name)
				return result, err
			}
			r.Log.Info(fmt.Sprintf("Clear machine: machine deployment name %s decided", mdName))
			if err := r.K8sClient.Get(r.RequestCtx, client.ObjectKey{Namespace: workerMachine.Namespace, Name: mdName}, md); err != nil {
				return ctrl.Result{}, err
			}
			r.Log.Info(fmt.Sprintf("Clear machine: machine deployment name %s retrieved", mdName))
			_, ok = md.Spec.Template.Annotations["cluster.x-k8s.io/restartedAt"]
			if !ok {
				now, err := metav1.Time{Time: time.Now()}.MarshalJSON()
				if err != nil {
					return ctrl.Result{}, err
				}
				md.Spec.Template.Annotations["cluster.x-k8s.io/restartedAt"] = string(now)
				patcher, err := patch.NewHelper(md, r.K8sClient)
				if err != nil {
					return ctrl.Result{}, err
				}
				if err := patcher.Patch(r.RequestCtx, md); err != nil {
					return ctrl.Result{}, err
				}
				r.Log.Info(fmt.Sprintf("Clear machine: machine deployment name %s add annotation restartedAt %s", mdName, string(now)))
			} else {
				r.Log.Info(fmt.Sprintf("Clear machine: machine deployment name %s already has restartedAt annotation, skip", mdName))
			}
		}
	}
	return ctrl.Result{}, nil
}

func triggerControlPlaneRollout(machines []infrav1.CloudStackMachine, r *CloudStackFailureDomainReconciliationRunner) (ctrl.Result, error) {
	var cpMachine infrav1.CloudStackMachine
	cpMachineFound := false
	for _, machine := range machines {
		for _, ref := range machine.OwnerReferences {
			if ref.Kind == "KubeadmControlPlane" {
				cpMachineFound = true
				cpMachine = machine
			}
			// etcdadmcluster rollout will trigger control plane rollout automatically.
			// if etcd VM exists, trigger etcdadmcluster rollout is enough, no need to trigger control plane VM rollout.
			if ref.Kind == "EtcdadmCluster" {
				return ctrl.Result{}, nil
			}
		}
	}
	if !cpMachineFound {
		r.Log.Info("Clear machine: no control plane VM found.")
		return ctrl.Result{}, nil
	}
	for _, ref := range cpMachine.OwnerReferences {
		if ref.Kind == "KubeadmControlPlane" {
			r.Log.Info("Clear machine: getting kubeadmControlPlane...")
			kcp := &controlplanev1.KubeadmControlPlane{}
			if err := r.K8sClient.Get(r.RequestCtx, client.ObjectKey{Namespace: cpMachine.Namespace, Name: ref.Name}, kcp); err != nil {
				return ctrl.Result{}, err
			}
			r.Log.Info(fmt.Sprintf("Clear machine: kubeadmControlPlane %s retrieved", ref.Name))
			if kcp.Spec.RolloutAfter != nil {
				kcp.Spec.RolloutAfter = &metav1.Time{Time: time.Now()}
				patcher, err := patch.NewHelper(kcp, r.K8sClient)
				if err != nil {
					return ctrl.Result{}, err
				}
				if err := patcher.Patch(r.RequestCtx, kcp); err != nil {
					return ctrl.Result{}, err
				}
				r.Log.Info(fmt.Sprintf("Clear machine: kubeadmControlPlane %s rolloutAfter spec set %v", ref.Name, *kcp.Spec.RolloutAfter))
			} else {
				r.Log.Info(fmt.Sprintf("Clear machine: kubeadmControlPlane %s rolloutAfter already set, skip", ref.Name))
			}
		}
	}
	return ctrl.Result{}, nil
}

func checkClusterReady(r *CloudStackFailureDomainReconciliationRunner) (ctrl.Result, error) {
	for _, condition := range r.CAPICluster.Status.Conditions {
		if condition.Type == "ControlPlaneReady" && condition.Status == "False" {
			return ctrl.Result{}, errors.New("cluster control plane not ready")
		}
		if condition.Type == "ManagedEtcdReady" && condition.Status == "False" {
			return ctrl.Result{}, errors.New("cluster managed etcd not ready")
		}
	}
	machineDeployments := &clusterv1.MachineDeploymentList{}
	if err := r.K8sClient.List(
		r.RequestCtx,
		machineDeployments,
		client.InNamespace(r.ReconciliationSubject.Namespace),
		client.MatchingLabels{clusterv1.ClusterLabelName: r.ReconciliationSubject.Labels[clusterv1.ClusterLabelName]},
	); err != nil {
		return ctrl.Result{}, errors.New("cluster machine deployment not found")
	}
	for _, md := range machineDeployments.Items {
		for _, condition := range md.Status.Conditions {
			if condition.Type == "Ready" && condition.Status != "True" {
				return ctrl.Result{}, errors.New("cluster machine deployment not ready")
			}
		}
	}
	return ctrl.Result{}, nil
}

// RemoveFinalizer just removes the finalizer from the failure domain.
func (r *CloudStackFailureDomainReconciliationRunner) RemoveFinalizer() (ctrl.Result, error) {
	controllerutil.RemoveFinalizer(r.ReconciliationSubject, infrav1.FailureDomainFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *CloudStackFailureDomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	_, err := ctrl.NewControllerManagedBy(mgr).For(&infrav1.CloudStackFailureDomain{}).Build(reconciler)
	return err
}
