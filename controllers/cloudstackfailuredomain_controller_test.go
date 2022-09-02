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
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CloudStackFailureDomainReconciler", func() {
	Context("With k8s like test environment.", func() {
		BeforeEach(func() {
			dummies.SetDummyVars()
			SetupTestEnvironment()                                                    // Must happen before setting up managers/reconcilers.
			Ω(FailureDomainReconciler.SetupWithManager(k8sManager)).Should(Succeed()) // Register CloudStack FailureDomainReconciler.
			// Modify failure domain name the same way the cluster controller would.
			dummies.CSFailureDomain1.Name = dummies.CSFailureDomain1.Name + "-" + dummies.CSCluster.Name

			Ω(k8sClient.Create(ctx, dummies.ACSEndpointSecret1))
			Ω(k8sClient.Create(ctx, dummies.CSFailureDomain1))

			mockCloudClient.EXPECT().ResolveZone(gomock.Any()).MinTimes(1)

			mockCloudClient.EXPECT().ResolveNetworkForZone(gomock.Any()).AnyTimes().Do(
				func(arg1 interface{}) {
					arg1.(*infrav1.CloudStackZoneSpec).Network.ID = "SomeID"
					arg1.(*infrav1.CloudStackZoneSpec).Network.Type = cloud.NetworkTypeShared
				}).MinTimes(1)

		})

		It("Should set failure domain Status.Ready to true.", func() {
			assertFailuredomainCreated()
		})
		It("Should delete failure domain if no VM under this failure domain.", func() {
			assertFailuredomainCreated()
			Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

			assertFailuredomainNotExisted()
		})
		It("Should trigger machine deployment rollout when only worker node in failure domain.", func() {
			assertFailuredomainCreated()
			setupCAPIMachineDeploymentCRDs(dummies.CAPIMachineDeployment)
			setupCAPIMachineAndCSMachineCRDs(dummies.CSMachine1, dummies.CAPIMachine1)
			setMachineOwnerReference(dummies.CSMachine1, dummies.MachineSetOwnerRef)
			labelMachineFailuredomain(dummies.CSMachine1, dummies.CSFailureDomain1)
			labelMachineDeploymentName(dummies.CSMachine1, dummies.CAPIMachineDeployment.Name)
			Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

			tempMd := &clusterv1.MachineDeployment{}
			Eventually(func() bool {
				key := client.ObjectKeyFromObject(dummies.CAPIMachineDeployment)
				if err := k8sClient.Get(ctx, key, tempMd); err == nil {
					_, ok := tempMd.Spec.Template.Annotations["cluster.x-k8s.io/restartedAt"]
					return ok
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())
		})
	})
})

func assertFailuredomainCreated() {
	tempfd := &infrav1.CloudStackFailureDomain{}
	Eventually(func() bool {
		key := client.ObjectKeyFromObject(dummies.CSFailureDomain1)
		if err := k8sClient.Get(ctx, key, tempfd); err == nil {
			return tempfd.Status.Ready
		}
		return false
	}, timeout).WithPolling(pollInterval).Should(BeTrue())
}

func assertFailuredomainNotExisted() {
	tempfd := &infrav1.CloudStackFailureDomain{}
	Eventually(func() bool {
		key := client.ObjectKeyFromObject(dummies.CSFailureDomain1)
		if err := k8sClient.Get(ctx, key, tempfd); err != nil {
			return true
		}
		return false
	}, timeout).WithPolling(pollInterval).Should(BeTrue())
}
