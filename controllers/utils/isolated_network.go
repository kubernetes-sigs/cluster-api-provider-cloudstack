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
		csIsoNet.Spec.Name = lowerName
		csIsoNet.Spec.ControlPlaneEndpoint.Host = r.CSCluster.Spec.ControlPlaneEndpoint.Host
		csIsoNet.Spec.ControlPlaneEndpoint.Port = r.CSCluster.Spec.ControlPlaneEndpoint.Port

		if err := r.Client.Create(r.RequestCtx, csIsoNet); err != nil && !ContainsAlreadyExistsSubstring(err) {
			return r.ReturnWrappedError(err, "error encountered when creating isolated network CRD")
		}
		return ctrl.Result{}, nil
	}
}
