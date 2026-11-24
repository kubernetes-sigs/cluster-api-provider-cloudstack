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
	"sync/atomic"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = ginkgo.Describe("CloudStackMachineReconciler", func() {
	ginkgo.Context("With machine controller running.", func() {
		ginkgo.BeforeEach(func() {
			dummies.SetDummyVars()
			dummies.CSCluster.Spec.FailureDomains = dummies.CSCluster.Spec.FailureDomains[:1]
			dummies.CSCluster.Spec.FailureDomains[0].Name = dummies.CSFailureDomain1.Spec.Name

			SetupTestEnvironment()                                                                                                                        // Must happen before setting up managers/reconcilers.
			gomega.Expect(MachineReconciler.SetupWithManager(ctx, k8sManager, controller.Options{SkipNameValidation: ptr.To(true)})).To(gomega.Succeed()) // Register the CloudStack MachineReconciler.

			// Point CAPI machine Bootstrap secret ref to dummy bootstrap secret.
			dummies.CAPIMachine.Spec.Bootstrap.DataSecretName = &dummies.BootstrapSecret.Name
			gomega.Expect(k8sClient.Create(ctx, dummies.BootstrapSecret)).To(gomega.Succeed())

			// Setup a failure domain for the machine reconciler to find.
			gomega.Expect(k8sClient.Create(ctx, dummies.CSFailureDomain1)).To(gomega.Succeed())
			setClusterReady(k8sClient)
		})

		ginkgo.It("Should call GetOrCreateVMInstance and set Status.Ready to true", func() {
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
			gomega.Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err == nil {
					if tempMachine.Status.Ready == true {
						return len(tempMachine.ObjectMeta.Finalizers) > 0
					}
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Should call DestroyVMInstance when CS machine deleted", func() {
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
			gomega.Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err == nil {
					if tempMachine.Status.Ready == true {
						return true
					}
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())

			gomega.Expect(k8sClient.Delete(ctx, dummies.CSMachine1)).To(gomega.Succeed())

			gomega.Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err != nil {
					return errors.IsNotFound(err)
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())

		})

		ginkgo.It("Should call ResolveVMInstanceDetails when CS machine without instanceID deleted", func() {
			instanceID := ptr.To("instance-id-123")
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
			gomega.Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err == nil {
					if tempMachine.Status.Ready == true {
						return true
					}
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())

			// remove instanceID from CS machine
			ph, err := patch.NewHelper(dummies.CSMachine1, k8sClient)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			dummies.CSMachine1.Spec.InstanceID = nil
			gomega.Expect(ph.Patch(ctx, dummies.CSMachine1)).To(gomega.Succeed())
			gomega.Expect(k8sClient.Delete(ctx, dummies.CSMachine1)).To(gomega.Succeed())

			gomega.Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err != nil {
					return errors.IsNotFound(err)
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())

		})

		ginkgo.It("Should replace ds.meta_data.xxx with proper values.", func() {
			// Mock a call to GetOrCreateVMInstance and set the machine to running.
			mockCloudClient.EXPECT().GetOrCreateVMInstance(
				gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Do(
				func(arg1, _, _, _, _, userdata interface{}) {
					expectedUserdata := fmt.Sprintf("%s{{%s}}", dummies.CAPIMachine.Name, dummies.CSMachine1.Spec.FailureDomainName)
					gomega.Expect(userdata == expectedUserdata).To(gomega.BeTrue())
					arg1.(*infrav1.CloudStackMachine).Status.InstanceState = "Running"
				}).AnyTimes()

			// Have to do this here or the reconcile call to GetOrCreateVMInstance may happen too early.
			setupMachineCRDs()

			// Eventually the machine should set ready to true.
			gomega.Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err == nil {
					if tempMachine.Status.Ready == true {
						return true
					}
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Should remove a control plane VM from the LB rule once its CAPI Machine is marked for deletion", func() {
			fd := &infrav1.CloudStackFailureDomain{}
			gomega.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dummies.CSFailureDomain1), fd)).To(gomega.Succeed())
			fd.Spec.Zone.Network = dummies.ISONet1
			gomega.Expect(k8sClient.Update(ctx, fd)).To(gomega.Succeed())

			dummies.CSISONet1.Name = dummies.ClusterName + "-" + dummies.ISONet1.Name
			dummies.CSISONet1.Spec.FailureDomainName = dummies.CSFailureDomain1.Spec.Name
			gomega.Expect(k8sClient.Create(ctx, dummies.CSISONet1)).To(gomega.Succeed())

			otherCPMachine := &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{Name: "other-cp-machine", Namespace: dummies.ClusterNameSpace, Labels: controlPlaneLabels()},
				Spec:       clusterv1.MachineSpec{ClusterName: dummies.ClusterName},
			}
			gomega.Expect(k8sClient.Create(ctx, otherCPMachine)).To(gomega.Succeed())

			dummies.CAPIMachine.Labels = controlPlaneLabels()
			dummies.CAPIMachine.Finalizers = []string{clusterv1.MachineFinalizer}
			dummies.CAPIMachine.Spec.InfrastructureRef = corev1.ObjectReference{
				APIVersion: infrav1.GroupVersion.String(),
				Kind:       "CloudStackMachine",
				Name:       dummies.CSMachine1.Name,
				Namespace:  dummies.ClusterNameSpace,
			}

			mockCloudClient.EXPECT().GetOrCreateVMInstance(
				gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Do(
				func(arg1, _, _, _, _, _ interface{}) {
					arg1.(*infrav1.CloudStackMachine).Status.InstanceState = "Running"
					controllerutil.AddFinalizer(arg1.(*infrav1.CloudStackMachine), infrav1.MachineFinalizer)
				}).AnyTimes()
			mockCloudClient.EXPECT().AssignVMToLoadBalancerRule(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			var removed atomic.Bool
			mockCloudClient.EXPECT().RemoveVMFromLoadBalancerRule(gomock.Any(), *dummies.CSMachine1.Spec.InstanceID).Do(
				func(_, _ interface{}) { removed.Store(true) }).AnyTimes().Return(nil)

			setupMachineCRDs()

			gomega.Eventually(func() bool {
				tempMachine := &infrav1.CloudStackMachine{}
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
				if err := k8sClient.Get(ctx, key, tempMachine); err == nil {
					return tempMachine.Status.Ready
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())
			gomega.Expect(removed.Load()).To(gomega.BeFalse())

			gomega.Expect(k8sClient.Delete(ctx, dummies.CAPIMachine)).To(gomega.Succeed())

			gomega.Eventually(removed.Load, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("API server LB membership with a fake ctrlRuntimeClient.", func() {
		var requestNamespacedName types.NamespacedName

		ginkgo.BeforeEach(func() {
			setupFakeTestClient()
			requestNamespacedName = types.NamespacedName{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
			dummies.CSFailureDomain1.Spec.Zone.Network = dummies.ISONet1
			dummies.CSISONet1.Name = dummies.ClusterName + "-" + dummies.ISONet1.Name

			dummies.CAPIMachine.Name = "cp-machine"
			dummies.CAPIMachine.Labels = controlPlaneLabels()
			dummies.CAPIMachine.Finalizers = []string{clusterv1.MachineFinalizer}
			dummies.CAPIMachine.Spec.Bootstrap.DataSecretName = &dummies.BootstrapSecret.Name
			dummies.CSMachine1.Labels = controlPlaneLabels()
			dummies.CSMachine1.Finalizers = []string{infrav1.MachineFinalizer}
			dummies.CSMachine1.OwnerReferences = append(dummies.CSMachine1.OwnerReferences, metav1.OwnerReference{
				Kind:       "Machine",
				APIVersion: clusterv1.GroupVersion.String(),
				Name:       dummies.CAPIMachine.Name,
				UID:        "uniqueness",
			})

			for _, obj := range []client.Object{dummies.CAPIMachine, dummies.CSMachine1, dummies.CSFailureDomain1,
				dummies.ACSEndpointSecret1, dummies.BootstrapSecret, dummies.CSISONet1} {
				gomega.Expect(fakeCtrlClient.Create(ctx, obj)).To(gomega.Succeed())
			}
			setClusterReady(fakeCtrlClient)

			mockCloudClient.EXPECT().GetOrCreateVMInstance(
				gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Do(
				func(arg1, _, _, _, _, _ interface{}) {
					arg1.(*infrav1.CloudStackMachine).Status.InstanceState = "Running"
				}).AnyTimes()
		})

		ginkgo.It("Should remove the VM from the LB rule when its CAPI Machine is being deleted", func() {
			gomega.Expect(fakeCtrlClient.Create(ctx, &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{Name: "other-cp-machine", Namespace: dummies.ClusterNameSpace, Labels: controlPlaneLabels()},
				Spec:       clusterv1.MachineSpec{ClusterName: dummies.ClusterName},
			})).To(gomega.Succeed())
			gomega.Expect(fakeCtrlClient.Delete(ctx, dummies.CAPIMachine)).To(gomega.Succeed())

			mockCloudClient.EXPECT().RemoveVMFromLoadBalancerRule(gomock.Any(), *dummies.CSMachine1.Spec.InstanceID).Times(1).Return(nil)
			mockCloudClient.EXPECT().AssignVMToLoadBalancerRule(gomock.Any(), gomock.Any()).Times(0)

			_, err := MachineReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: requestNamespacedName})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("Should keep the VM in the LB rule when it is the last active control plane Machine", func() {
			gomega.Expect(fakeCtrlClient.Delete(ctx, dummies.CAPIMachine)).To(gomega.Succeed())

			mockCloudClient.EXPECT().RemoveVMFromLoadBalancerRule(gomock.Any(), gomock.Any()).Times(0)
			mockCloudClient.EXPECT().AssignVMToLoadBalancerRule(gomock.Any(), gomock.Any()).Times(0)

			_, err := MachineReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: requestNamespacedName})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("Should remove the VM from the LB rule before destroying it when the CloudStackMachine is deleted", func() {
			gomega.Expect(fakeCtrlClient.Delete(ctx, dummies.CSMachine1)).To(gomega.Succeed())

			gomock.InOrder(
				mockCloudClient.EXPECT().RemoveVMFromLoadBalancerRule(gomock.Any(), *dummies.CSMachine1.Spec.InstanceID).Times(1).Return(nil),
				mockCloudClient.EXPECT().DestroyVMInstance(gomock.Any()).Times(1).Return(nil),
			)

			_, err := MachineReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: requestNamespacedName})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("With a fake ctrlRuntimeClient and no test Env at all.", func() {
		ginkgo.BeforeEach(func() {
			setupFakeTestClient()
			dummies.CSCluster.Spec.FailureDomains = dummies.CSCluster.Spec.FailureDomains[:1]
			dummies.CSCluster.Spec.FailureDomains[0].Name = dummies.CSFailureDomain1.Spec.Name
		})

		ginkgo.It("Should exit having not found a failure domain to place the machine in.", func() {
			key := client.ObjectKeyFromObject(dummies.CSCluster)
			dummies.CAPIMachine.Name = "someMachine"
			dummies.CSMachine1.OwnerReferences = append(dummies.CSMachine1.OwnerReferences, metav1.OwnerReference{
				Kind:       "Machine",
				APIVersion: clusterv1.GroupVersion.String(),
				Name:       dummies.CAPIMachine.Name,
				UID:        "uniqueness",
			})
			gomega.Expect(fakeCtrlClient.Get(ctx, key, dummies.CSCluster)).To(gomega.Succeed())
			gomega.Expect(fakeCtrlClient.Create(ctx, dummies.CAPIMachine)).To(gomega.Succeed())
			gomega.Expect(fakeCtrlClient.Create(ctx, dummies.CSMachine1)).To(gomega.Succeed())

			requestNamespacedName := types.NamespacedName{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
			res, err := MachineReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: requestNamespacedName})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(res.RequeueAfter).ShouldNot(gomega.BeZero())
		})

		ginkgo.It("Should create event Machine instance is Running", func() {
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
			gomega.Expect(fakeCtrlClient.Get(ctx, key, dummies.CSCluster)).To(gomega.Succeed())
			gomega.Expect(fakeCtrlClient.Create(ctx, dummies.CAPIMachine)).To(gomega.Succeed())
			gomega.Expect(fakeCtrlClient.Create(ctx, dummies.CSMachine1)).To(gomega.Succeed())
			gomega.Expect(fakeCtrlClient.Create(ctx, dummies.CSFailureDomain1)).To(gomega.Succeed())
			gomega.Expect(fakeCtrlClient.Create(ctx, dummies.ACSEndpointSecret1)).To(gomega.Succeed())
			gomega.Expect(fakeCtrlClient.Create(ctx, dummies.BootstrapSecret)).To(gomega.Succeed())

			setClusterReady(fakeCtrlClient)

			requestNamespacedName := types.NamespacedName{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
			MachineReconciler.AsFailureDomainUser(&dummies.CSFailureDomain1.Spec)
			res, err := MachineReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: requestNamespacedName})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(res.RequeueAfter).Should(gomega.BeZero())

			gomega.Eventually(func() bool {
				for event := range fakeRecorder.Events {
					return strings.Contains(event, "Normal Created CloudStack instance Created") ||
						strings.Contains(event, "Normal Running Machine instance is Running...") ||
						strings.Contains(event, "Normal Machine State Checker CloudStackMachineStateChecker created")
				}
				return false
			}, timeout).Should(gomega.BeTrue())
		})
	})
})

func controlPlaneLabels() map[string]string {
	return map[string]string{
		clusterv1.ClusterNameLabel:         dummies.ClusterName,
		clusterv1.MachineControlPlaneLabel: "",
	}
}
