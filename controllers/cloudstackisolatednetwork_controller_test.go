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

package controllers_test

import (
	g "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CloudStackIsolatedNetworkReconciler", func() {
	Context("With k8s like test environment.", func() {
		BeforeEach(func() {
			SetupTestEnvironment()                                             // Must happen before setting up managers/reconcilers.
			Ω(IsoNetReconciler.SetupWithManager(k8sManager)).Should(Succeed()) // Register CloudStack IsoNetReconciler.
		})

		It("Should set itself to ready if there are no errors in calls to CloudStack methods.", func() {
			mockCloudClient.EXPECT().GetOrCreateIsolatedNetwork(g.Any(), g.Any(), g.Any()).AnyTimes()
			mockCloudClient.EXPECT().AddClusterTag(g.Any(), g.Any(), g.Any()).AnyTimes()

			Ω(k8sClient.Create(ctx, dummies.CSISONet1)).Should(Succeed())
			Eventually(func() bool {
				tempIsoNet := &infrav1.CloudStackIsolatedNetwork{}
				key := client.ObjectKeyFromObject(dummies.CSISONet1)
				if err := k8sClient.Get(ctx, key, tempIsoNet); err == nil {
					return true
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())
		})
	})
})
