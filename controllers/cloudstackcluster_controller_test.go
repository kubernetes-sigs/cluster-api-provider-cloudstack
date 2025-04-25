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
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

var _ = ginkgo.Describe("CloudStackClusterReconciler", func() {
	ginkgo.Context("With k8s like test environment.", func() {
		ginkgo.BeforeEach(func() {
			SetupTestEnvironment()                                                                                                                        // Must happen before setting up managers/reconcilers.
			gomega.Ω(ClusterReconciler.SetupWithManager(ctx, k8sManager, controller.Options{SkipNameValidation: ptr.To(true)})).Should(gomega.Succeed())  // Register CloudStack ClusterReconciler.
			gomega.Ω(FailureDomainReconciler.SetupWithManager(k8sManager, controller.Options{SkipNameValidation: ptr.To(true)})).Should(gomega.Succeed()) // Register CloudStack FailureDomainReconciler.
		})

		ginkgo.It("Should create a CloudStackFailureDomain.", func() {
			tempfd := &infrav1.CloudStackFailureDomain{}
			mockCloudClient.EXPECT().ResolveZone(gomock.Any()).AnyTimes()
			gomega.Eventually(func() bool {
				key := client.ObjectKeyFromObject(dummies.CSFailureDomain1)
				key.Name = key.Name + "-" + dummies.CSCluster.Name
				if err := k8sClient.Get(ctx, key, tempfd); err != nil {
					return true
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Without a k8s test environment.", func() {
		ginkgo.It("Should create a reconciliation runner with a Cloudstack Cluster as the reconciliation subject.", func() {
			reconRunenr := controllers.NewCSClusterReconciliationRunner()
			gomega.Ω(reconRunenr.ReconciliationSubject).ShouldNot(gomega.BeNil())
		})
	})
})
