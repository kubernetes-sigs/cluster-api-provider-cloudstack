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

const (
	conditionTypeManagedEtcdReady = "ManagedEtcdReady"
	conditionStatusTrue           = "True"
	conditionStatusFalse          = "False"
	kindKubeadmControlPlane       = "KubeadmControlPlane"
	kindMachineSet                = "MachineSet"
	kindEtcdadmCluster            = "EtcdadmCluster"
	kindCloudStackAffinityGroup   = "CloudStackAffinityGroup"
	kindCloudStackIsolatedNetwork = "CloudStackIsolatedNetwork"
)

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackfailuredomains/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinedeployments,verbs=get;list;watch;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplates,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=etcdcluster.cluster.x-k8s.io,resources=etcdadmclusters,verbs=get;list;watch;patch
//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubeadmcontrolplanes,verbs=get;list;watch;patch

// CloudStackFailureDomainReconciliationRunner is a ReconciliationRunner with extensions specific to CloudStackFailureDomains.
// The runner does the actual reconciliation.
type CloudStackFailureDomainReconciliationRunner struct {
	*csCtrlrUtils.ReconciliationRunner
	ReconciliationSubject *infrav1.CloudStackFailureDomain
	IsoNet                *infrav1.CloudStackIsolatedNetwork
	Machines              *infrav1.CloudStackMachineList
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
		r.CheckClusterReady,
		r.GetCSMachine,
		r.TriggerEtcdClusterRollout,
		r.TriggerControlPlaneRollout,
		r.TriggerMachineDeploymentRollout,
		r.DeleteOwnedObjects(
			infrav1.GroupVersion.WithKind(kindCloudStackAffinityGroup),
			infrav1.GroupVersion.WithKind(kindCloudStackIsolatedNetwork)),
		r.CheckOwnedObjectsDeleted(
			infrav1.GroupVersion.WithKind(kindCloudStackAffinityGroup),
			infrav1.GroupVersion.WithKind(kindCloudStackIsolatedNetwork)),
		r.RemoveFinalizer,
	)
}

// GetCSMachine fetch all cloudstack machines under this failuredomain by checking label of each cloudstack machines.
func (r *CloudStackFailureDomainReconciliationRunner) GetCSMachine() (ctrl.Result, error) {
	machines := &infrav1.CloudStackMachineList{}
	if err := r.K8sClient.List(r.RequestCtx, machines, client.MatchingLabels{infrav1.FailureDomainLabelName: r.ReconciliationSubject.Name}); err != nil {
		return ctrl.Result{}, err
	}
	r.Machines = machines
	return ctrl.Result{}, nil
}

// TriggerEtcdClusterRollout triggers an etcdadmcluster rolling upgrade if any etcd cloudstack machine found,
// and requeue until all CloudStack machines are cleared from the FailureDomain.
func (r *CloudStackFailureDomainReconciliationRunner) TriggerEtcdClusterRollout() (ctrl.Result, error) {
	for _, machine := range r.Machines.Items {
		for _, ref := range machine.OwnerReferences {
			if ref.Kind == kindEtcdadmCluster {
				// try to get etcdadmcluster by using machine's etcdadmcluster ownerReference
				etcdadmCluster := &unstructured.Unstructured{}
				etcdadmCluster.SetGroupVersionKind(schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind))
				if err := r.K8sClient.Get(r.RequestCtx, client.ObjectKey{Namespace: machine.Namespace, Name: ref.Name}, etcdadmCluster); err != nil {
					return ctrl.Result{}, err
				}
				csMachineTemplateNameInEtcdadmCluster, found, err := unstructured.NestedString(etcdadmCluster.Object, "spec", "infrastructureTemplate", "name")
				if err != nil {
					return ctrl.Result{}, err
				}
				if !found {
					return ctrl.Result{}, errors.Errorf("name not found in %s spec.infrastructureTemplate", ref.Name)
				}

				csMachineTemplateName, err := getTemplateNameIfMatch(machine, csMachineTemplateNameInEtcdadmCluster)
				if err != nil {
					return ctrl.Result{}, err
				}

				newCsMachineTemplateName, err := cloneCsMachineTemplateWithNewName(r, machine.Namespace, csMachineTemplateName, ref.Name)
				if err != nil {
					return ctrl.Result{}, err
				}

				// patch etcdadmcluster to refer newly created cloudstack machine template, which will trigger an etcdadmcluster rolling upgrade.
				if _, err = controllerutil.CreateOrPatch(r.RequestCtx, r.K8sClient, etcdadmCluster,
					func() error {
						return unstructured.SetNestedField(
							etcdadmCluster.Object,
							newCsMachineTemplateName,
							"spec", "infrastructureTemplate", "name",
						)
					}); err != nil {
					return ctrl.Result{}, err
				}
				return r.RequeueWithMessage("failuredomain delete: etcdadmcluster infrastructureTemplate cloudstackmachinetemplate name patched,", "etcdadmcluster",
					ref.Name, "cloudstackmachinetemplatename", newCsMachineTemplateName)
			}
		}
	}

	return ctrl.Result{}, nil
}

// TriggerMachineDeploymentRollout triggers a machine deployment rolling upgrade if any worker cloudstack machine found,
// and requeue until all CloudStack machines are cleared from the FailureDomain.
func (r *CloudStackFailureDomainReconciliationRunner) TriggerMachineDeploymentRollout() (ctrl.Result, error) {
	for _, machine := range r.Machines.Items {
		for _, ref := range machine.OwnerReferences {
			if ref.Kind == kindMachineSet {
				// get machine deployment by using cloudstack machine's label info
				md := &clusterv1.MachineDeployment{}
				mdName, ok := machine.Labels[clusterv1.MachineDeploymentLabelName]
				if !ok {
					return ctrl.Result{}, errors.Errorf("cloudstack machine %s lacks expected label: %s", machine.Name, clusterv1.MachineDeploymentLabelName)
				}

				if err := r.K8sClient.Get(r.RequestCtx, client.ObjectKey{Namespace: machine.Namespace, Name: mdName}, md); err != nil {
					return ctrl.Result{}, err
				}

				// add an annotation restartedAt in machine deployment if such one not already added
				// this will trigger an immediate machine deployment rollout
				patcher, err := patch.NewHelper(md, r.K8sClient)
				if err != nil {
					return ctrl.Result{}, err
				}
				if md.Spec.Template.Annotations == nil {
					md.Spec.Template.Annotations = map[string]string{}
				}
				timeNowStr := time.Now().Format(time.RFC3339)
				md.Spec.Template.Annotations["cluster.x-k8s.io/restartedAt"] = timeNowStr
				if err = patcher.Patch(r.RequestCtx, md); err != nil {
					return ctrl.Result{}, err
				}
				return r.RequeueWithMessage("failuredomain delete: machine deployment annotation restartedAt added,", "restartedAt", timeNowStr)
			}
		}
	}
	return ctrl.Result{}, nil
}

// TriggerControlPlaneRollout triggers a kubeadm control plane rolling upgrade if any control plane cloudstack machine found,
// and requeue until all CloudStack machines are cleared from the FailureDomain.
func (r *CloudStackFailureDomainReconciliationRunner) TriggerControlPlaneRollout() (ctrl.Result, error) {
	for _, machine := range r.Machines.Items {
		for _, ref := range machine.OwnerReferences {
			if ref.Kind == kindKubeadmControlPlane {
				// get kcp (kubeadmcontrolplane) by using cloudstack machine's kubeadmControlPlane ownerReference
				kcp := &controlplanev1.KubeadmControlPlane{}
				if err := r.K8sClient.Get(r.RequestCtx, client.ObjectKey{Namespace: machine.Namespace, Name: ref.Name}, kcp); err != nil {
					return ctrl.Result{}, err
				}

				// set kcp spec.RolloutAfter, this will trigger control plane rollout immediately
				if kcp.Spec.RolloutAfter == nil {
					patcher, err := patch.NewHelper(kcp, r.K8sClient)
					if err != nil {
						return ctrl.Result{}, err
					}
					kcp.Spec.RolloutAfter = &metav1.Time{Time: time.Now()}
					if err = patcher.Patch(r.RequestCtx, kcp); err != nil {
						return ctrl.Result{}, err
					}
				}
				return r.RequeueWithMessage("failuredomain delete: kubeadm control plane RolloutAfter added", "RolloutAfter,", kcp.Spec.RolloutAfter)
			}
		}
	}

	return ctrl.Result{}, nil
}

// CheckClusterReady check cluster to see if there is any rolling update going on.
// etcdadmcluster, kubeadmControlPlane, or machine deployment rolling update.
func (r *CloudStackFailureDomainReconciliationRunner) CheckClusterReady() (ctrl.Result, error) {
	for _, condition := range r.CAPICluster.Status.Conditions {
		if condition.Type == clusterv1.ControlPlaneReadyCondition && condition.Status == conditionStatusFalse {
			return r.RequeueWithMessage("cluster control plane not ready,")
		}
		if condition.Type == conditionTypeManagedEtcdReady && condition.Status == conditionStatusFalse {
			return r.RequeueWithMessage("cluster managed etcd not ready,")
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
			if condition.Type == clusterv1.ReadyCondition && condition.Status != conditionStatusTrue {
				return r.RequeueWithMessage("cluster machine deployment not ready,")
			}
		}
	}
	return ctrl.Result{}, nil
}

func getTemplateNameIfMatch(machine infrav1.CloudStackMachine, templateName string) (string, error) {
	// get cloudstack machine template by using machine's annotation info
	csMachineTemplateName, ok := machine.Annotations["cluster.x-k8s.io/cloned-from-name"]
	if !ok {
		return "", errors.Errorf("annotations['cluster.x-k8s.io/cloned-from-name'] not found in cloudstack machine %s", machine.Name)
	}

	if csMachineTemplateName != templateName {
		return "", errors.Errorf("cloudstackmachinetemplate %s in machine %s annotation and cloudstackmachinetemplate %s in etcdadmcluster are different",
			csMachineTemplateName, machine.Name, templateName)
	}
	return csMachineTemplateName, nil
}

func cloneCsMachineTemplateWithNewName(r *CloudStackFailureDomainReconciliationRunner, namespace string, cloudstackmachineTemplateName string, etcdadmClusterName string) (string, error) {
	// get cloudstack machine template from k8s
	csMachineTemplate := &infrav1.CloudStackMachineTemplate{}
	if err := r.K8sClient.Get(r.RequestCtx, client.ObjectKey{Namespace: namespace, Name: cloudstackmachineTemplateName}, csMachineTemplate); err != nil {
		return "", err
	}
	// create a new cloudstack machine template, which will be referred by etcdadmcluster spec.infrastructureTemplate
	csMachineTemplate.Name = fmt.Sprintf("%s-template-%d", etcdadmClusterName, time.Now().UnixNano()/int64(time.Millisecond))
	csMachineTemplate.SetAnnotations(map[string]string{})
	csMachineTemplate.SetResourceVersion("")
	if err := r.K8sClient.Create(r.RequestCtx, csMachineTemplate); err != nil {
		return "", err
	}
	return csMachineTemplate.Name, nil

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
