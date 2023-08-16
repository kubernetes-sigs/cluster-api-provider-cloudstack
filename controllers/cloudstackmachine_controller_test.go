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
	"fmt"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("CloudStackMachineReconciler", func() {
	Context("With machine controller running.", func() {
		BeforeEach(func() {
			dummies.SetDummyVars()
			dummies.CSCluster.Spec.FailureDomains = dummies.CSCluster.Spec.FailureDomains[:1]
			dummies.CSCluster.Spec.FailureDomains[0].Name = dummies.CSFailureDomain1.Spec.Name

			SetupTestEnvironment()                                                                    // Must happen before setting up managers/reconcilers.
			Ω(MachineReconciler.SetupWithManager(k8sManager, controller.Options{})).Should(Succeed()) // Register the CloudStack MachineReconciler.

			// Point CAPI machine Bootstrap secret ref to dummy bootstrap secret.
			dummies.CAPIMachine.Spec.Bootstrap.DataSecretName = &dummies.BootstrapSecret.Name
			Ω(k8sClient.Create(ctx, dummies.BootstrapSecret)).Should(Succeed())

			// Setup a failure domain for the machine reconciler to find.
			Ω(k8sClient.Create(ctx, dummies.CSFailureDomain1)).Should(Succeed())
			setClusterReady(k8sClient)

			mockCloudClient.EXPECT().GetOrCreateUnmanagedCluster(gomock.Any(), gomock.Any(), gomock.Any()).Do(
				func(arg1, _, _ interface{}) {
					arg1.(*infrav1.CloudStackCluster).Status.CloudStackClusterID = "cluster-id-123"
				}).AnyTimes().Return(nil)

			mockCloudClient.EXPECT().AddVMToUnmanagedCluster(
				gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			mockCloudClient.EXPECT().RemoveVMFromUnmanagedCluster(
				gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		})

		It("Should call GetOrCreateVMInstance and set Status.Ready to true", func() {
			// Mock a call to GetOrCreateVMInstance and set the machine to running.
			mockCloudClient.EXPECT().GetOrCreateVMInstance(
				gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Do(
				func(arg1, _, _, _, _, _ interface{}) {
					arg1.(*infrav1.CloudStackMachine).Status.InstanceState = "Running"
				}).AnyTimes()

			// Have to do this here or the reconcile call to GetOrCreateVMInstance may happen too early.
			setupMachineCRDs()

			// Eventually the machine should set ready to true.
			Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err == nil {
					if tempMachine.Status.Ready == true {
						return len(tempMachine.ObjectMeta.Finalizers) > 0
					}
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())
		})

		It("Should call DestroyVMInstance when CS machine deleted", func() {
			// Mock a call to GetOrCreateVMInstance and set the machine to running.
			mockCloudClient.EXPECT().GetOrCreateVMInstance(
				gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Do(
				func(arg1, _, _, _, _, _ interface{}) {
					arg1.(*infrav1.CloudStackMachine).Status.InstanceState = "Running"
					controllerutil.AddFinalizer(arg1.(*infrav1.CloudStackMachine), infrav1.MachineFinalizer)
				}).AnyTimes()

			mockCloudClient.EXPECT().DestroyVMInstance(gomock.Any()).Times(1).Return(nil)
			// Have to do this here or the reconcile call to GetOrCreateVMInstance may happen too early.
			setupMachineCRDs()

			// Eventually the machine should set ready to true.
			Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err == nil {
					if tempMachine.Status.Ready == true {
						return true
					}
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())

			Ω(k8sClient.Delete(ctx, dummies.CSMachine1)).Should(Succeed())

			Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err != nil {
					return errors.IsNotFound(err)
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())

		})

		It("Should call ResolveVMInstanceDetails when CS machine without instanceID deleted", func() {
			instanceID := pointer.String("instance-id-123")
			// Mock a call to GetOrCreateVMInstance and set the machine to running.
			mockCloudClient.EXPECT().GetOrCreateVMInstance(
				gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Do(
				func(arg1, _, _, _, _, _ interface{}) {
					arg1.(*infrav1.CloudStackMachine).Status.InstanceState = "Running"
					controllerutil.AddFinalizer(arg1.(*infrav1.CloudStackMachine), infrav1.MachineFinalizer)
				}).AnyTimes()

			mockCloudClient.EXPECT().ResolveVMInstanceDetails(gomock.Any()).Do(
				func(arg1 interface{}) {
					arg1.(*infrav1.CloudStackMachine).Spec.InstanceID = instanceID
				}).AnyTimes().Return(nil)

			mockCloudClient.EXPECT().DestroyVMInstance(gomock.Any()).Times(1).Return(nil)
			// Have to do this here or the reconcile call to GetOrCreateVMInstance may happen too early.
			setupMachineCRDs()

			// Eventually the machine should set ready to true.
			Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err == nil {
					if tempMachine.Status.Ready == true {
						return true
					}
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())

			// remove instanceID from CS machine
			ph, err := patch.NewHelper(dummies.CSMachine1, k8sClient)
			Ω(err).ShouldNot(HaveOccurred())
			dummies.CSMachine1.Spec.InstanceID = nil
			Ω(ph.Patch(ctx, dummies.CSMachine1)).Should(Succeed())
			Ω(k8sClient.Delete(ctx, dummies.CSMachine1)).Should(Succeed())

			Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err != nil {
					return errors.IsNotFound(err)
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())

		})

		It("Should replace ds.meta_data.xxx with proper values.", func() {
			// Mock a call to GetOrCreateVMInstance and set the machine to running.
			mockCloudClient.EXPECT().GetOrCreateVMInstance(
				gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Do(
				func(arg1, _, _, _, _, userdata interface{}) {
					expectedUserdata := fmt.Sprintf("%s{{%s}}", dummies.CAPIMachine.Name, dummies.CSMachine1.Spec.FailureDomainName)
					Ω(userdata == expectedUserdata).Should(BeTrue())
					arg1.(*infrav1.CloudStackMachine).Status.InstanceState = "Running"
				}).AnyTimes()

			// Have to do this here or the reconcile call to GetOrCreateVMInstance may happen too early.
			setupMachineCRDs()

			// Eventually the machine should set ready to true.
			Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err == nil {
					if tempMachine.Status.Ready == true {
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
			dummies.CSCluster.Spec.FailureDomains = dummies.CSCluster.Spec.FailureDomains[:1]
			dummies.CSCluster.Spec.FailureDomains[0].Name = dummies.CSFailureDomain1.Spec.Name
		})

		It("Should exit having not found a failure domain to place the machine in.", func() {
			key := client.ObjectKeyFromObject(dummies.CSCluster)
			dummies.CAPIMachine.Name = "someMachine"
			dummies.CSMachine1.OwnerReferences = append(dummies.CSMachine1.OwnerReferences, metav1.OwnerReference{
				Kind:       "Machine",
				APIVersion: clusterv1.GroupVersion.String(),
				Name:       dummies.CAPIMachine.Name,
				UID:        "uniqueness",
			})
			Ω(fakeCtrlClient.Get(ctx, key, dummies.CSCluster)).Should(Succeed())
			Ω(fakeCtrlClient.Create(ctx, dummies.CAPIMachine)).Should(Succeed())
			Ω(fakeCtrlClient.Create(ctx, dummies.CSMachine1)).Should(Succeed())

			requestNamespacedName := types.NamespacedName{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
			res, err := MachineReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: requestNamespacedName})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(res.RequeueAfter).ShouldNot(BeZero())
		})

		It("Should create event Machine instance is Running", func() {
			key := client.ObjectKeyFromObject(dummies.CSCluster)
			dummies.CAPIMachine.Name = "someMachine"
			dummies.CAPIMachine.Spec.Bootstrap.DataSecretName = &dummies.BootstrapSecret.Name
			dummies.CSMachine1.OwnerReferences = append(dummies.CSMachine1.OwnerReferences, metav1.OwnerReference{
				Kind:       "Machine",
				APIVersion: clusterv1.GroupVersion.String(),
				Name:       dummies.CAPIMachine.Name,
				UID:        "uniqueness",
			})
			mockCloudClient.EXPECT().GetOrCreateVMInstance(
				gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Do(
				func(arg1, _, _, _, _, _ interface{}) {
					arg1.(*infrav1.CloudStackMachine).Status.InstanceState = "Running"
				}).AnyTimes()
			mockCloudClient.EXPECT().AddVMToUnmanagedCluster(
				gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			Ω(fakeCtrlClient.Get(ctx, key, dummies.CSCluster)).Should(Succeed())
			Ω(fakeCtrlClient.Create(ctx, dummies.CAPIMachine)).Should(Succeed())
			Ω(fakeCtrlClient.Create(ctx, dummies.CSMachine1)).Should(Succeed())
			Ω(fakeCtrlClient.Create(ctx, dummies.CSFailureDomain1)).Should(Succeed())
			Ω(fakeCtrlClient.Create(ctx, dummies.ACSEndpointSecret1)).Should(Succeed())
			Ω(fakeCtrlClient.Create(ctx, dummies.BootstrapSecret)).Should(Succeed())

			setClusterReady(fakeCtrlClient)

			requestNamespacedName := types.NamespacedName{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
			MachineReconciler.AsFailureDomainUser(&dummies.CSFailureDomain1.Spec)
			res, err := MachineReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: requestNamespacedName})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(res.RequeueAfter).Should(BeZero())

			Eventually(func() bool {
				for event := range fakeRecorder.Events {
					return strings.Contains(event, "Normal Created CloudStack instance Created") ||
						strings.Contains(event, "Normal Running Machine instance is Running...") ||
						strings.Contains(event, "Normal Machine State Checker CloudStackMachineStateChecker created")
				}
				return false
			}, timeout).Should(BeTrue())
		})
	})
})
