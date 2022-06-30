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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CloudStackMachineReconciler", func() {
	BeforeEach(func() {
		// Register the MachineReconciler only.
		Ω(MachineReconciler.SetupWithManager(k8sManager)).Should(Succeed())

		dummies.SetDummyVars()

		// Create the kubeadm bootstrap secret.
		dummies.CAPIMachine.Spec.Bootstrap.DataSecretName = pointer.String("asdf")
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: dummies.ClusterNameSpace,
				Name:      *dummies.CAPIMachine.Spec.Bootstrap.DataSecretName},
			Data: map[string][]byte{"value": make([]byte, 0)}}
		Ω(k8sClient.Create(ctx, secret)).Should(Succeed())

		Ω(k8sClient.Create(ctx, dummies.CSZone1)).Should(Succeed())
		setClusterReady()
		setupMachineCRDs()
	})

	It("Should call GetOrCreateVMInstance and set Status.Ready to true", func() {

		// Mock a call to GetOrCreateVMInstance and set the machine to running.
		mockCloudClient.EXPECT().GetOrCreateVMInstance(
			gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any()).Do(
			func(arg1, _, _, _, _, _ interface{}) {
				arg1.(*infrav1.CloudStackMachine).Status.InstanceState = "Running"
			}).AnyTimes()

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
