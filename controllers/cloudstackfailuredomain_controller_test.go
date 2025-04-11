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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

var _ = ginkgo.Describe("CloudStackFailureDomainReconciler", func() {
	ginkgo.Context("With k8s like test environment.", func() {
		ginkgo.BeforeEach(func() {
			dummies.SetDummyVars()
			SetupTestEnvironment()                                                                                                                        // Must happen before setting up managers/reconcilers.
			gomega.Ω(FailureDomainReconciler.SetupWithManager(k8sManager, controller.Options{SkipNameValidation: ptr.To(true)})).Should(gomega.Succeed()) // Register CloudStack FailureDomainReconciler.
			// Modify failure domain name the same way the cluster controller would.
			dummies.CSFailureDomain1.Name = dummies.CSFailureDomain1.Name + "-" + dummies.CSCluster.Name

			gomega.Ω(k8sClient.Create(ctx, dummies.ACSEndpointSecret1))
			gomega.Ω(k8sClient.Create(ctx, dummies.CSFailureDomain1))

			mockCloudClient.EXPECT().ResolveZone(gomock.Any()).MinTimes(1)

			mockCloudClient.EXPECT().ResolveNetworkForZone(gomock.Any()).AnyTimes().Do(
				func(arg1 interface{}) {
					arg1.(*infrav1.CloudStackZoneSpec).Network.ID = "SomeID"
					arg1.(*infrav1.CloudStackZoneSpec).Network.Type = cloud.NetworkTypeShared
				}).MinTimes(1)
		})

		ginkgo.It("Should delete failure domain if no VM under this failure domain.", func() {
			gomega.Eventually(func() bool {
				return getFailuredomainStatus(dummies.CSFailureDomain1)
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())

			gomega.Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

			tempfd := &infrav1.CloudStackFailureDomain{}
			gomega.Eventually(func() bool {
				key := client.ObjectKeyFromObject(dummies.CSFailureDomain1)
				if err := k8sClient.Get(ctx, key, tempfd); err != nil {
					return errors.IsNotFound(err)
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())
		})

		ginkgo.DescribeTable("Should function in different replicas conditions",
			func(shouldDeleteVM bool, specReplicas, statusReplicas, statusReadyReplicas *int32, statusReady *bool, controlPlaneReady bool) {
				gomega.Eventually(func() bool {
					return getFailuredomainStatus(dummies.CSFailureDomain1)
				}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())

				setCSMachineOwnerCRD(dummies.CSMachineOwner, specReplicas, statusReplicas, statusReadyReplicas, statusReady)
				setCAPIMachineAndCSMachineCRDs(dummies.CSMachine1, dummies.CAPIMachine)
				setMachineOwnerReference(dummies.CSMachine1, dummies.CSMachineOwnerReference)
				labelMachineFailuredomain(dummies.CSMachine1, dummies.CSFailureDomain1)

				if !controlPlaneReady {
					gomega.Eventually(func() error {
						ph, err := patch.NewHelper(dummies.CAPICluster, k8sClient)
						gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
						dummies.CAPICluster.Status.Conditions = []clusterv1.Condition{
							{
								Type:   "Ready",
								Status: "False",
							},
						}
						return ph.Patch(ctx, dummies.CAPICluster, patch.WithStatusObservedGeneration{})
					}, timeout).Should(gomega.Succeed())
				}

				gomega.Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

				CAPIMachine := &clusterv1.Machine{}
				if shouldDeleteVM {
					gomega.Eventually(func() bool {
						key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CAPIMachine.Name}
						if err := k8sClient.Get(ctx, key, CAPIMachine); err != nil {
							return errors.IsNotFound(err)
						}
						return false
					}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())
				} else {
					gomega.Consistently(func() bool {
						key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CAPIMachine.Name}
						if err := k8sClient.Get(ctx, key, CAPIMachine); err == nil {
							return CAPIMachine.DeletionTimestamp.IsZero()
						}
						return false
					}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())
				}

			},
			// should delete - simulate owner is kubeadmcontrolplane
			ginkgo.Entry("Should delete machine if spec.replicas > 1", true, ptr.To(int32(2)), ptr.To(int32(2)), ptr.To(int32(2)), ptr.To(true), true),
			// should delete - simulate owner is etcdadmcluster
			ginkgo.Entry("Should delete machine if status.readyReplica does not exist", true, ptr.To(int32(2)), ptr.To(int32(2)), nil, ptr.To(true), true),
			// should delete - simulate owner is machineset
			ginkgo.Entry("Should delete machine if status.ready does not exist", true, ptr.To(int32(2)), ptr.To(int32(2)), ptr.To(int32(2)), nil, true),
			// should not delete if condition not met
			ginkgo.Entry("Should not delete machine if cluster control plane not ready", false, ptr.To(int32(2)), ptr.To(int32(2)), ptr.To(int32(2)), ptr.To(true), false),
			ginkgo.Entry("Should not delete machine if status.replicas < spec.replicas", false, ptr.To(int32(2)), ptr.To(int32(1)), ptr.To(int32(1)), ptr.To(true), true),
			ginkgo.Entry("Should not delete machine if spec.replicas < 2", false, ptr.To(int32(1)), ptr.To(int32(1)), ptr.To(int32(1)), ptr.To(true), true),
			ginkgo.Entry("Should not delete machine if status.ready is false", false, ptr.To(int32(2)), ptr.To(int32(2)), ptr.To(int32(2)), ptr.To(false), true),
			ginkgo.Entry("Should not delete machine if status.readyReplicas <> status.replicas", false, ptr.To(int32(2)), ptr.To(int32(2)), ptr.To(int32(1)), ptr.To(true), true),
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
