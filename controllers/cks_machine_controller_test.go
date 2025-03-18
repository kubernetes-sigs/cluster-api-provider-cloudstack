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
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = ginkgo.Describe("CksCloudStackMachineReconciler", func() {
	ginkgo.Context("With machine controller running.", func() {
		ginkgo.BeforeEach(func() {
			dummies.SetDummyVars()
			dummies.CSCluster.Spec.SyncWithACS = ptr.To(true)
			dummies.CSCluster.Spec.FailureDomains = dummies.CSCluster.Spec.FailureDomains[:1]
			dummies.CSCluster.Spec.FailureDomains[0].Name = dummies.CSFailureDomain1.Spec.Name
			dummies.CSCluster.Status.CloudStackClusterID = "cluster-id-123"

			SetupTestEnvironment()                                                                                       // Must happen before setting up managers/reconcilers.
			gomega.Ω(MachineReconciler.SetupWithManager(ctx, k8sManager, controller.Options{})).Should(gomega.Succeed()) // Register the CloudStack MachineReconciler.
			gomega.Ω(CksClusterReconciler.SetupWithManager(k8sManager)).Should(gomega.Succeed())                         // Register the CloudStack MachineReconciler.
			gomega.Ω(CksMachineReconciler.SetupWithManager(k8sManager)).Should(gomega.Succeed())                         // Register the CloudStack MachineReconciler.

			mockCloudClient.EXPECT().GetOrCreateCksCluster(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_, arg1, _ interface{}) {
				arg1.(*infrav1.CloudStackCluster).Status.CloudStackClusterID = "cluster-id-123"
			}).MinTimes(1).Return(nil)
			// Point CAPI machine Bootstrap secret ref to dummy bootstrap secret.
			dummies.CAPIMachine.Spec.Bootstrap.DataSecretName = &dummies.BootstrapSecret.Name
			gomega.Ω(k8sClient.Create(ctx, dummies.BootstrapSecret)).Should(gomega.Succeed())

			// Setup a failure domain for the machine reconciler to find.
			gomega.Ω(k8sClient.Create(ctx, dummies.CSFailureDomain1)).Should(gomega.Succeed())
			setClusterReady(k8sClient)
		})

		ginkgo.It("Should call AddVMToCksCluster", func() {
			// Mock a call to GetOrCreateVMInstance and set the machine to running.
			mockCloudClient.EXPECT().GetOrCreateVMInstance(
				gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Do(
				func(arg1, _, _, _, _, _ interface{}) {
					arg1.(*infrav1.CloudStackMachine).Status.InstanceState = "Running"
				}).AnyTimes()

			mockCloudClient.EXPECT().AddVMToCksCluster(
				gomock.Any(), gomock.Any()).MinTimes(1).Return(nil)
			// Have to do this here or the reconcile call to GetOrCreateVMInstance may happen too early.
			setupMachineCRDs()

			// Eventually the machine should set ready to true.
			gomega.Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err == nil {
					if tempMachine.Status.Ready == true {
						return len(tempMachine.ObjectMeta.Finalizers) > 1
					}
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Should call RemoveVMFromCksCluster when CS machine deleted", func() {
			// Mock a call to GetOrCreateVMInstance and set the machine to running.
			mockCloudClient.EXPECT().GetOrCreateVMInstance(
				gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Do(
				func(arg1, _, _, _, _, _ interface{}) {
					arg1.(*infrav1.CloudStackMachine).Status.InstanceState = "Running"
					controllerutil.AddFinalizer(arg1.(*infrav1.CloudStackMachine), infrav1.MachineFinalizer)
				}).AnyTimes()

			mockCloudClient.EXPECT().AddVMToCksCluster(gomock.Any(), gomock.Any()).MinTimes(1).Return(nil)

			mockCloudClient.EXPECT().DestroyVMInstance(gomock.Any()).MinTimes(1).Return(nil)
			mockCloudClient.EXPECT().RemoveVMFromCksCluster(
				gomock.Any(), gomock.Any()).MinTimes(1).Return(nil)
			// Have to do this here or the reconcile call to GetOrCreateVMInstance may happen too early.
			setupMachineCRDs()

			// Eventually the machine should set ready to true.
			gomega.Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err == nil {
					if tempMachine.Status.Ready == true {
						return len(tempMachine.ObjectMeta.Finalizers) > 1
					}
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())

			gomega.Ω(k8sClient.Delete(ctx, dummies.CSMachine1)).Should(gomega.Succeed())

			gomega.Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err != nil {
					return errors.IsNotFound(err)
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Without a k8s test environment.", func() {
		ginkgo.It("Should create a reconciliation runner with a Cloudstack Machine as the reconciliation subject.", func() {
			reconRunner := controllers.NewCksMachineReconciliationRunner()
			gomega.Ω(reconRunner.ReconciliationSubject).ShouldNot(gomega.BeNil())
		})
	})
})
