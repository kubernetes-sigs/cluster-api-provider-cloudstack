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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CloudStackClusterReconciler", func() {
	Context("With k8s like test environment.", func() {
		BeforeEach(func() {
			SetupTestEnvironment()                                              // Must happen before setting up managers/reconcilers.
			Ω(ClusterReconciler.SetupWithManager(k8sManager)).Should(Succeed()) // Register CloudStack ClusterReconciler.
		})

		It("Should create a CloudStackFailureDomain.", func() {
			tempfd := &infrav1.CloudStackFailureDomain{}
			Eventually(func() bool {
				key := client.ObjectKeyFromObject(dummies.CSFailureDomain1)
				if err := k8sClient.Get(ctx, key, tempfd); err != nil {
					return true
				}
				return false
			}, timeout).Should(BeTrue())
		})
	})

	Context("Without a k8s test environment.", func() {
		It("Should create a reconciliation runner with a Cloudstack Cluster as the reconciliation subject.", func() {
			reconRunenr := controllers.NewCSClusterReconciliationRunner()
			Ω(reconRunenr.ReconciliationSubject).ShouldNot(BeNil())
		})
	})
})
