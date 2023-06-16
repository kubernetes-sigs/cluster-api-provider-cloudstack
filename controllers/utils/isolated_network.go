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
	"fmt"
	"strings"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *ReconciliationRunner) IsoNetMetaName(name string) string {
	return fmt.Sprintf("%s-%s", r.CSCluster.Name, strings.ToLower(name))
}

// GenerateIsolatedNetwork of the passed name that's owned by the ReconciliationSubject.
func (r *ReconciliationRunner) GenerateIsolatedNetwork(name string, fdNameFunc func() string) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		lowerName := strings.ToLower(name)
		metaName := fmt.Sprintf("%s-%s", r.CSCluster.Name, lowerName)
		csIsoNet := &infrav1.CloudStackIsolatedNetwork{}
		csIsoNet.ObjectMeta = r.NewChildObjectMeta(metaName)
		csIsoNet.Spec.Name = lowerName
		csIsoNet.Spec.FailureDomainName = fdNameFunc()
		csIsoNet.Spec.ControlPlaneEndpoint.Host = r.CSCluster.Spec.ControlPlaneEndpoint.Host
		csIsoNet.Spec.ControlPlaneEndpoint.Port = r.CSCluster.Spec.ControlPlaneEndpoint.Port

		if err := r.K8sClient.Create(r.RequestCtx, csIsoNet); err != nil && !ContainsAlreadyExistsSubstring(err) {
			return r.ReturnWrappedError(err, "creating isolated network CRD")
		}
		return ctrl.Result{}, nil
	}
}
