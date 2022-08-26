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

package v1beta2_test

import (
	"context"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloudStackMachineTemplate webhook", func() {
	var ctx context.Context
	requiredRegex := "admission webhook.*denied the request.*Required value\\: %s"

	BeforeEach(func() { // Reset test vars to initial state.
		dummies.SetDummyVars()
		ctx = context.Background()
		_ = k8sClient.Delete(ctx, dummies.CSMachineTemplate1) // Delete any remnants.
	})

	Context("When creating a CloudStackMachineTemplate", func() {
		It("Should accept a CloudStackMachineTemplate with all attributes present", func() {
			Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).Should(Succeed())
		})

		It("Should accept a CloudStackMachineTemplate when missing the VM Disk Offering attribute", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.DiskOffering = infrav1.CloudStackResourceDiskOffering{
				CloudStackResourceIdentifier: infrav1.CloudStackResourceIdentifier{Name: "", ID: ""},
			}
			Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).Should(Succeed())
		})

		It("Should reject a CloudStackMachineTemplate when missing the VM Offering attribute", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.Offering = infrav1.CloudStackResourceIdentifier{Name: "", ID: ""}
			Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).
				Should(MatchError(MatchRegexp(requiredRegex, "Offering")))
		})

		It("Should reject a CloudStackMachineTemplate when missing the VM Template attribute", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.Template = infrav1.CloudStackResourceIdentifier{Name: "", ID: ""}
			Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).
				Should(MatchError(MatchRegexp(requiredRegex, "Template")))
		})
	})

})
