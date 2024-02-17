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
	"k8s.io/utils/pointer"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

var _ = Describe("CloudStackFailureDomainReconciler", func() {
	Context("With k8s like test environment.", func() {
		BeforeEach(func() {
			dummies.SetDummyVars()
			SetupTestEnvironment()                                                                          // Must happen before setting up managers/reconcilers.
			Ω(FailureDomainReconciler.SetupWithManager(k8sManager, controller.Options{})).Should(Succeed()) // Register CloudStack FailureDomainReconciler.
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

		It("Should delete failure domain if no VM under this failure domain.", func() {
			Eventually(func() bool {
				return getFailuredomainStatus(dummies.CSFailureDomain1)
			}, timeout).WithPolling(pollInterval).Should(BeTrue())

			Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

			tempfd := &infrav1.CloudStackFailureDomain{}
			Eventually(func() bool {
				key := client.ObjectKeyFromObject(dummies.CSFailureDomain1)
				if err := k8sClient.Get(ctx, key, tempfd); err != nil {
					return errors.IsNotFound(err)
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())
		})

		DescribeTable("Should function in different replicas conditions",
			func(shouldDeleteVM bool, specReplicas, statusReplicas, statusReadyReplicas *int32, statusReady *bool, controlPlaneReady bool) {
				Eventually(func() bool {
					return getFailuredomainStatus(dummies.CSFailureDomain1)
				}, timeout).WithPolling(pollInterval).Should(BeTrue())

				setCSMachineOwnerCRD(dummies.CSMachineOwner, specReplicas, statusReplicas, statusReadyReplicas, statusReady)
				setCAPIMachineAndCSMachineCRDs(dummies.CSMachine1, dummies.CAPIMachine)
				setMachineOwnerReference(dummies.CSMachine1, dummies.CSMachineOwnerReference)
				labelMachineFailuredomain(dummies.CSMachine1, dummies.CSFailureDomain1)

				if !controlPlaneReady {
					Eventually(func() error {
						ph, err := patch.NewHelper(dummies.CAPICluster, k8sClient)
						Ω(err).ShouldNot(HaveOccurred())
						dummies.CAPICluster.Status.Conditions = []clusterv1.Condition{
							{
								Type:   "Ready",
								Status: "False",
							},
						}
						return ph.Patch(ctx, dummies.CAPICluster, patch.WithStatusObservedGeneration{})
					}, timeout).Should(Succeed())
				}

				Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

				CAPIMachine := &clusterv1.Machine{}
				if shouldDeleteVM {
					Eventually(func() bool {
						key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CAPIMachine.Name}
						if err := k8sClient.Get(ctx, key, CAPIMachine); err != nil {
							return errors.IsNotFound(err)
						}
						return false
					}, timeout).WithPolling(pollInterval).Should(BeTrue())
				} else {
					Consistently(func() bool {
						key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CAPIMachine.Name}
						if err := k8sClient.Get(ctx, key, CAPIMachine); err == nil {
							return CAPIMachine.DeletionTimestamp.IsZero()
						}
						return false
					}, timeout).WithPolling(pollInterval).Should(BeTrue())
				}

			},
			// should delete - simulate owner is kubeadmcontrolplane
			Entry("Should delete machine if spec.replicas > 1", true, pointer.Int32(2), pointer.Int32(2), pointer.Int32(2), pointer.Bool(true), true),
			// should delete - simulate owner is etcdadmcluster
			Entry("Should delete machine if status.readyReplica does not exist", true, pointer.Int32(2), pointer.Int32(2), nil, pointer.Bool(true), true),
			// should delete - simulate owner is machineset
			Entry("Should delete machine if status.ready does not exist", true, pointer.Int32(2), pointer.Int32(2), pointer.Int32(2), nil, true),
			// should not delete if condition not met
			Entry("Should not delete machine if cluster control plane not ready", false, pointer.Int32(2), pointer.Int32(2), pointer.Int32(2), pointer.Bool(true), false),
			Entry("Should not delete machine if status.replicas < spec.replicas", false, pointer.Int32(2), pointer.Int32(1), pointer.Int32(1), pointer.Bool(true), true),
			Entry("Should not delete machine if spec.replicas < 2", false, pointer.Int32(1), pointer.Int32(1), pointer.Int32(1), pointer.Bool(true), true),
			Entry("Should not delete machine if status.ready is false", false, pointer.Int32(2), pointer.Int32(2), pointer.Int32(2), pointer.Bool(false), true),
			Entry("Should not delete machine if status.readyReplicas <> status.replicas", false, pointer.Int32(2), pointer.Int32(2), pointer.Int32(1), pointer.Bool(true), true),
		)
	})
})

func getFailuredomainStatus(failureDomain *infrav1.CloudStackFailureDomain) bool {
	tempfd := &infrav1.CloudStackFailureDomain{}
	key := client.ObjectKeyFromObject(failureDomain)
	if err := k8sClient.Get(ctx, key, tempfd); err == nil {
		return tempfd.Status.Ready
	}
	return false
}
