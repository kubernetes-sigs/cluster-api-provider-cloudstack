package utils

import (
	"strings"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// GenerateIsolatedNetwork of the passed name that's owned by the ReconciliationSubject.
func (r *ReconciliationRunner) GenerateIsolatedNetwork(name string) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		lowerName := strings.ToLower(name)
		csIsoNet := &infrav1.CloudStackIsolatedNetwork{}
		csIsoNet.ObjectMeta = r.NewChildObjectMeta(lowerName)
		csIsoNet.Spec.Name = strings.ToLower(lowerName)
		csIsoNet.Spec.ControlPlaneEndpoint.Host = r.CSCluster.Spec.ControlPlaneEndpoint.Host
		csIsoNet.Spec.ControlPlaneEndpoint.Port = r.CSCluster.Spec.ControlPlaneEndpoint.Port

		if err := r.Client.Create(r.RequestCtx, csIsoNet); err != nil && !ContainsAlreadyExistsSubstring(err) {
			return r.ReturnWrappedError(err, "error encountered when creating isolated network CRD")
		}
		return ctrl.Result{}, nil
	}
}
