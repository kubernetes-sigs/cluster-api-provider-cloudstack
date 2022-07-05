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
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CloudStackAffinityGroupReconciler", func() {
	BeforeEach(func() {
		SetupTestEnvironment() // Must happen before setting up managers/reconcilers.
		dummies.SetDummyVars()
		Ω(AffinityGReconciler.SetupWithManager(k8sManager)).Should(Succeed()) // Register CloudStack AffinityGReconciler.
	})

	It("Should patch back the affinity group as ready after calling GetOrCreateAffinityGroup.", func() {
		Ω(k8sClient.Create(ctx, dummies.CSAffinityGroup)).Should(Succeed())

		mockCloudClient.EXPECT().GetOrCreateAffinityGroup(gomock.Any()).AnyTimes()

		// Test that the AffinityGroup controller sets Status.Ready to true.
		Eventually(func() bool {
			nameSpaceFilter := &client.ListOptions{Namespace: dummies.ClusterNameSpace}
			affinityGroups := &infrav1.CloudStackAffinityGroupList{}
			if err := k8sClient.List(ctx, affinityGroups, nameSpaceFilter); err == nil {
				if len(affinityGroups.Items) == 1 {
					return affinityGroups.Items[0].Status.Ready
				}
			}
			return false
		}, timeout).WithPolling(2 * time.Second).Should(BeTrue())
	})
})
