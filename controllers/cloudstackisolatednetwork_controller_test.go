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
	"k8s.io/apimachinery/pkg/types"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CloudStackIsolatedNetworkReconciler", func() {
	Context("With k8s like test environment.", func() {
		BeforeEach(func() {
			SetupTestEnvironment() // Must happen before setting up managers/reconcilers.
			dummies.SetDummyVars()
			Ω(IsoNetReconciler.SetupWithManager(k8sManager)).Should(Succeed()) // Register CloudStack IsoNetReconciler.
		})

		It("Should set itself to ready if there are no errors in calls to CloudStack methods.", func() {
			mockCloudClient.EXPECT().GetOrCreateIsolatedNetwork(g.Any(), g.Any(), g.Any()).AnyTimes()
			mockCloudClient.EXPECT().AddClusterTag(g.Any(), g.Any(), g.Any()).AnyTimes()

			// We use CSFailureDomain2 here because CSFailureDomain1 has an empty Spec.Zone.ID
			dummies.CSISONet1.Spec.FailureDomainName = dummies.CSFailureDomain2.Spec.Name
			Ω(k8sClient.Create(ctx, dummies.CSFailureDomain2)).Should(Succeed())
			Ω(k8sClient.Create(ctx, dummies.CSISONet1)).Should(Succeed())

			Eventually(func() bool {
				tempIsoNet := &infrav1.CloudStackIsolatedNetwork{}
				key := client.ObjectKeyFromObject(dummies.CSISONet1)
				if err := k8sClient.Get(ctx, key, tempIsoNet); err == nil {
					if tempIsoNet.Status.Ready == true {
						return true
					}
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())
		})
	})

	Context("With a fake ctrlRuntimeClient and no test Env at all.", func() {
		BeforeEach(func() {
			setupFakeTestClient()
		})

		It("Should requeue in case the zone ID is not resolved yet.", func() {
			// We use CSFailureDomain1 here because it has an empty Spec.Zone.ID
			dummies.CSISONet1.Spec.FailureDomainName = dummies.CSFailureDomain1.Spec.Name
			Ω(fakeCtrlClient.Create(ctx, dummies.CSFailureDomain1)).Should(Succeed())
			Ω(fakeCtrlClient.Create(ctx, dummies.CSISONet1)).Should(Succeed())

			requestNamespacedName := types.NamespacedName{Namespace: dummies.ClusterNameSpace, Name: dummies.CSISONet1.Name}
			res, err := IsoNetReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: requestNamespacedName})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(res.RequeueAfter).ShouldNot(BeZero())
		})
	})
})
