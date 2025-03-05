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

package v1beta3_test

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
)

var _ = ginkgo.Describe("CloudStackMachineTemplate webhook", func() {
	var ctx context.Context
	forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"
	requiredRegex := "admission webhook.*denied the request.*Required value\\: %s"

	ginkgo.BeforeEach(func() { // Reset test vars to initial state.
		dummies.SetDummyVars()
		ctx = context.Background()
		_ = k8sClient.Delete(ctx, dummies.CSMachineTemplate1) // Delete any remnants.
	})

	ginkgo.Context("When creating a CloudStackMachineTemplate", func() {
		ginkgo.It("Should accept a CloudStackMachineTemplate with all attributes present", func() {
			gomega.Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).Should(gomega.Succeed())
		})

		ginkgo.It("Should accept a CloudStackMachineTemplate when missing the VM Disk Offering attribute", func() {
			dummies.CSMachineTemplate1.Spec.Template.Spec.DiskOffering = infrav1.CloudStackResourceDiskOffering{
				CloudStackResourceIdentifier: infrav1.CloudStackResourceIdentifier{Name: "", ID: ""},
			}
			gomega.Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).Should(gomega.Succeed())
		})

		ginkgo.It("Should reject a CloudStackMachineTemplate when missing the VM Offering attribute", func() {
			dummies.CSMachineTemplate1.Spec.Template.Spec.Offering = infrav1.CloudStackResourceIdentifier{Name: "", ID: ""}
			gomega.Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).
				Should(gomega.MatchError(gomega.MatchRegexp(requiredRegex, "Offering")))
		})

		ginkgo.It("Should reject a CloudStackMachineTemplate when missing the VM Template attribute", func() {
			dummies.CSMachineTemplate1.Spec.Template.Spec.Template = infrav1.CloudStackResourceIdentifier{Name: "", ID: ""}
			gomega.Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).
				Should(gomega.MatchError(gomega.MatchRegexp(requiredRegex, "Template")))
		})
	})

	ginkgo.Context("When updating a CloudStackMachineTemplate", func() {
		ginkgo.BeforeEach(func() { // Reset test vars to initial state.
			gomega.Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).Should(gomega.Succeed())
		})

		ginkgo.It("should reject VM template updates to the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Template.Spec.Template = infrav1.CloudStackResourceIdentifier{Name: "ArbitraryUpdateTemplate"}
			gomega.Expect(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).
				Should(gomega.MatchError(gomega.MatchRegexp(forbiddenRegex, "template")))
		})

		ginkgo.It("should reject VM disk offering updates to the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Template.Spec.DiskOffering = infrav1.CloudStackResourceDiskOffering{
				CloudStackResourceIdentifier: infrav1.CloudStackResourceIdentifier{Name: "DiskOffering2"}}
			gomega.Expect(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).
				Should(gomega.MatchError(gomega.MatchRegexp(forbiddenRegex, "diskOffering")))
		})

		ginkgo.It("should reject VM offering updates to the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Template.Spec.Offering = infrav1.CloudStackResourceIdentifier{Name: "Offering2"}
			gomega.Expect(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).
				Should(gomega.MatchError(gomega.MatchRegexp(forbiddenRegex, "offering")))
		})

		ginkgo.It("should reject updates to VM details of the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Template.Spec.Details = map[string]string{"memoryOvercommitRatio": "1.5"}
			gomega.Expect(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).
				Should(gomega.MatchError(gomega.MatchRegexp(forbiddenRegex, "details")))
		})

		ginkgo.It("should reject updates to the list of AffinityGroupIDs of the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Template.Spec.AffinityGroupIDs = []string{"28b907b8-75a7-4214-bd3d-6c61961fc2ag"}
			gomega.Expect(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).ShouldNot(gomega.Succeed())
		})
	})
})
