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
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CloudStackMachineReconciler", func() {
	Context("With machine controller running.", func() {
		BeforeEach(func() {
			SetupTestEnvironment()                                              // Must happen before setting up managers/reconcilers.
			Ω(MachineReconciler.SetupWithManager(k8sManager)).Should(Succeed()) // Register the CloudStack MachineReconciler.

			dummies.SetDummyVars()

			// Point CAPI machine Bootstrap secret ref to dummy bootstrap secret.
			dummies.CAPIMachine.Spec.Bootstrap.DataSecretName = &dummies.BootstrapSecret.Name
			Ω(k8sClient.Create(ctx, dummies.BootstrapSecret)).Should(Succeed())

			// Setup a zone for the machine reconciler to find.
			Ω(k8sClient.Create(ctx, dummies.CSZone1)).Should(Succeed())
			setClusterReady()
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
						return true
					}
				}
				return false
			}, timeout).WithPolling(2 * time.Second).Should(BeTrue())
		})
	})

	Context("With a fake ctrlRuntimeClient and no test Env at all.", func() {
		BeforeEach(func() {
			dummies.SetDummyVars()
			setupFakeTestClient()
		})

		It("Should exit having not found a zone to place the machine in.", func() {
			key := client.ObjectKeyFromObject(dummies.CSCluster)
			Ω(fakeCtrlClient.Get(ctx, key, dummies.CSCluster)).Should(Succeed())
			Ω(fakeCtrlClient.Create(ctx, dummies.CSMachine1)).Should(Succeed())

			requestNamespacedName := types.NamespacedName{Namespace: dummies.ClusterNameSpace, Name: dummies.CSMachine1.Name}
			res, err := MachineReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: requestNamespacedName})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(res.RequeueAfter).ShouldNot(BeZero())
		})
	})
})
