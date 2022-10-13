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
	"k8s.io/apimachinery/pkg/api/errors"
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
			assertFailureDomainCreated()
		})
		It("Should delete failure domain if no VM under this failure domain.", func() {
			assertFailureDomainCreated()
			Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

			assertFailureDomainNotExisted()
		})
		DescribeTable("Should function in different replicas conditions",
			func(shouldDeleteVM bool, specReplicas, statusReplicas, statusReadyReplicas *int32, statusReady *bool) {
				assertFailureDomainCreated()
				setCSMachineOwnerCRD(dummies.CSMachineOwner, specReplicas, statusReplicas, statusReadyReplicas, statusReady)
				setCAPIMachineAndCSMachineCRDs(dummies.CSMachine1, dummies.CAPIMachine)
				setMachineOwnerReference(dummies.CSMachine1, dummies.CSMachineOwnerReference)
				labelMachineFailuredomain(dummies.CSMachine1, dummies.CSFailureDomain1)

				Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

				CAPIMachine := &clusterv1.Machine{}
				Eventually(func() bool {
					key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CAPIMachine.Name}
					if shouldDeleteVM {
						if err := k8sClient.Get(ctx, key, CAPIMachine); err != nil {
							return errors.IsNotFound(err)
						}
					} else {
						if err := k8sClient.Get(ctx, key, CAPIMachine); err == nil {
							return CAPIMachine.DeletionTimestamp.IsZero()
						}
					}
					return false
				}, timeout).WithPolling(pollInterval).Should(BeTrue())
			},
			// should delete - simulate owner is kubeadmcontrolplane
			Entry("Should delete machine if spec.replicas > 1", true, int32Pointer(2), int32Pointer(2), int32Pointer(2), boolPointer(true)),
			// should delete - simulate owner is etcdadmcluster
			Entry("Should delete machine if status.readyReplica does not exist", true, int32Pointer(2), int32Pointer(2), nil, boolPointer(true)),
			// should delete - simulate owner is machineset
			Entry("Should delete machine if status.ready does not exist", true, int32Pointer(2), int32Pointer(2), int32Pointer(2), nil),
			// should not delete if condition not met
			Entry("Should return error if status.replicas < spec.replicas", false, int32Pointer(2), int32Pointer(1), int32Pointer(1), boolPointer(true)),
			Entry("Should return error if spec.replicas < 2", false, int32Pointer(1), int32Pointer(1), int32Pointer(1), boolPointer(true)),
			Entry("Should return error if status.ready is false", false, int32Pointer(2), int32Pointer(2), int32Pointer(2), boolPointer(false)),
			Entry("Should return error if status.readyReplicas <> status.replicas", false, int32Pointer(2), int32Pointer(2), int32Pointer(1), boolPointer(true)),
		)
	})
})

func assertFailureDomainCreated() {
	tempfd := &infrav1.CloudStackFailureDomain{}
	Eventually(func() bool {
		key := client.ObjectKeyFromObject(dummies.CSFailureDomain1)
		if err := k8sClient.Get(ctx, key, tempfd); err == nil {
			return tempfd.Status.Ready
		}
		return false
	}, timeout).WithPolling(pollInterval).Should(BeTrue())
}

func assertFailureDomainNotExisted() {
	tempfd := &infrav1.CloudStackFailureDomain{}
	Eventually(func() bool {
		key := client.ObjectKeyFromObject(dummies.CSFailureDomain1)
		if err := k8sClient.Get(ctx, key, tempfd); err != nil {
			return true
		}
		return false
	}, timeout).WithPolling(pollInterval).Should(BeTrue())
}

func boolPointer(b bool) *bool {
	return &b
}

func int32Pointer(b int32) *int32 {
	return &b
}
