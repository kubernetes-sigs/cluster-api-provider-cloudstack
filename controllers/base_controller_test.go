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
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
)

var _ = Describe("Base Reconciler Methods", func() {
	Context("Fake enve", func() {
		BeforeEach(func() {
			dummies.SetDummyVars()
			setupFakeTestClient()
		})

		It("Should kick some major butt.", func() {
			Ω(fakeCtrlClient.Create(ctx, dummies.ACSEndpointSecret1)).Should(Succeed())
			reconRunner := controllers.NewCSClusterReconciliationRunner().UsingBaseReconciler(ClusterReconciler.ReconcilerBase)
			Ω(reconRunner.ReconciliationSubject).ShouldNot(BeNil())
			_, err := reconRunner.AsFailureDomainUser(dummies.CSFailureDomain1.Spec)()
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

})
