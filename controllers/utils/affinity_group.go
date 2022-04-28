package utils

import (
	"fmt"
	"strings"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/pkg/errors"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GenerateIsolatedNetwork of the passed name that's owned by the ReconciliationSubject.
func (r *ReconciliationRunner) GetOrCreateAffinityGroup(name string, affinityType string, ag *infrav1.CloudStackAffinityGroup) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		// Start by attempting a fetch.
		lowerName := strings.ToLower(name)
		namespace := r.ReconciliationSubject.GetNamespace()
		objKey := client.ObjectKey{Namespace: namespace, Name: lowerName}
		if err := r.Client.Get(r.RequestCtx, objKey, ag); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		} else if ag.Name != "" {
			return ctrl.Result{}, nil
		} // Didn't find a group.

		if affinityType == infrav1.ProAffinity {
			ag.Spec.Type = "host affinity"
		} else {
			ag.Spec.Type = "host anti-affinity"
		}
		ag.Spec.Name = name

		for _, ref := range r.ReconciliationSubject.GetOwnerReferences() {
			if ref.Kind == "EtcdadmCluster" || ref.Kind == "KubeadmControlPlane" || ref.Kind == "MachineSet" {
				ag.OwnerReferences = append(ag.OwnerReferences, ref)
				break
			}
		}

		// Create if fetch didn't work.
		ag.Name = name
		ag.Spec.Name = name
		ag.ObjectMeta = r.NewChildObjectMeta(lowerName)
		if err := r.Client.Create(r.RequestCtx, ag); err != nil && !ContainsAlreadyExistsSubstring(err) {
			return r.ReturnWrappedError(err, "error encountered when creating affinity group CRD")
		}
		return ctrl.Result{}, nil
	}
}

// The computed affinity group name relevant to this machine.
func AffinityGroupName(csm infrav1.CloudStackMachine, capiMachine *capiv1.Machine) (string, error) {
	managerOwnerRef := GetManagementOwnerRef(capiMachine)
	if managerOwnerRef == nil {
		return "", errors.Errorf("could not find owner UID for %s/%s", csm.Namespace, csm.Name)
	}
	return fmt.Sprintf("%sAffinity-%s-%s", strings.Title(csm.Spec.Affinity), managerOwnerRef.Name, managerOwnerRef.UID), nil
}
