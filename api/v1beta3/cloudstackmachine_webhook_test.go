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

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"

	"github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("CloudStackMachine webhook", func() {
	var ctx context.Context
	forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"
	requiredRegex := "admission webhook.*denied the request.*Required value\\: %s"

	ginkgo.BeforeEach(func() { // Reset test vars to initial state.
		dummies.SetDummyVars()
		ctx = context.Background()
		_ = k8sClient.Delete(ctx, dummies.CSMachine1) // Clear out any remaining machines. Ignore errors.
	})

	ginkgo.Context("When creating a CloudStackMachine", func() {
		ginkgo.It("should accept CloudStackMachine with all attributes", func() {
			gomega.Expect(k8sClient.Create(ctx, dummies.CSMachine1)).Should(gomega.Succeed())
		})

		ginkgo.It("should accept a CloudStackMachine with disk Offering attribute", func() {
			dummies.CSMachine1.Spec.DiskOffering = dummies.DiskOffering
			gomega.Expect(k8sClient.Create(ctx, dummies.CSMachine1)).Should(gomega.Succeed())
		})

		ginkgo.It("should accept a CloudStackMachine with positive disk Offering size attribute", func() {
			dummies.CSMachine1.Spec.DiskOffering = dummies.DiskOffering
			dummies.CSMachine1.Spec.DiskOffering.CustomSize = 1
			gomega.Expect(k8sClient.Create(ctx, dummies.CSMachine1)).Should(gomega.Succeed())
		})

		ginkgo.It("should not accept a CloudStackMachine with negative disk Offering size attribute", func() {
			dummies.CSMachine1.Spec.DiskOffering = dummies.DiskOffering
			dummies.CSMachine1.Spec.DiskOffering.CustomSize = -1
			gomega.Expect(k8sClient.Create(ctx, dummies.CSMachine1)).Should(gomega.MatchError(gomega.MatchRegexp(forbiddenRegex, "customSizeInGB")))
		})

		ginkgo.It("should reject a CloudStackMachine with missing Offering attribute", func() {
			dummies.CSMachine1.Spec.Offering = infrav1.CloudStackResourceIdentifier{ID: "", Name: ""}
			gomega.Expect(k8sClient.Create(ctx, dummies.CSMachine1)).
				Should(gomega.MatchError(gomega.MatchRegexp(requiredRegex, "Offering")))
		})

		ginkgo.It("should reject a CloudStackMachine with missing Template attribute", func() {
			dummies.CSMachine1.Spec.Template = infrav1.CloudStackResourceIdentifier{ID: "", Name: ""}
			gomega.Expect(k8sClient.Create(ctx, dummies.CSMachine1)).
				Should(gomega.MatchError(gomega.MatchRegexp(requiredRegex, "Template")))
		})
	})

	ginkgo.Context("When updating a CloudStackMachine", func() {
		ginkgo.BeforeEach(func() {
			gomega.Ω(k8sClient.Create(ctx, dummies.CSMachine1)).Should(gomega.Succeed())
		})

		ginkgo.It("should reject VM offering updates to the CloudStackMachine", func() {
			dummies.CSMachine1.Spec.Offering = infrav1.CloudStackResourceIdentifier{Name: "ArbitraryUpdateOffering"}
			gomega.Ω(k8sClient.Update(ctx, dummies.CSMachine1)).
				Should(gomega.MatchError(gomega.MatchRegexp(forbiddenRegex, "offering")))
		})

		ginkgo.It("should reject VM template updates to the CloudStackMachine", func() {
			dummies.CSMachine1.Spec.Template = infrav1.CloudStackResourceIdentifier{Name: "ArbitraryUpdateTemplate"}
			gomega.Ω(k8sClient.Update(ctx, dummies.CSMachine1)).
				Should(gomega.MatchError(gomega.MatchRegexp(forbiddenRegex, "template")))
		})

		ginkgo.It("should reject VM disk offering updates to the CloudStackMachine", func() {
			dummies.CSMachine1.Spec.DiskOffering.Name = "medium"
			gomega.Ω(k8sClient.Update(ctx, dummies.CSMachine1)).
				Should(gomega.MatchError(gomega.MatchRegexp(forbiddenRegex, "diskOffering")))
		})

		ginkgo.It("should reject updates to VM details of the CloudStackMachine", func() {
			dummies.CSMachine1.Spec.Details = map[string]string{"memoryOvercommitRatio": "1.5"}
			gomega.Ω(k8sClient.Update(ctx, dummies.CSMachine1)).
				Should(gomega.MatchError(gomega.MatchRegexp(forbiddenRegex, "details")))
		})

		ginkgo.It("should reject updates to the list of affinty groups of the CloudStackMachine", func() {
			dummies.CSMachine1.Spec.AffinityGroupIDs = []string{"28b907b8-75a7-4214-bd3d-6c61961fc2af"}
			gomega.Ω(k8sClient.Update(ctx, dummies.CSMachine1)).
				Should(gomega.MatchError(gomega.MatchRegexp(forbiddenRegex, "AffinityGroupIDs")))
		})
	})
})
