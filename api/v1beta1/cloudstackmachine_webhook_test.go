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

package v1beta1_test

import (
	"context"

	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloudStackMachine webhook", func() {
	var ctx context.Context

	BeforeEach(func() { // Reset test vars to initial state.
		dummies.SetDummyVars()
		ctx = context.Background()
		// Clear out any remaining machines. Ignore errors.
		_ = k8sClient.Delete(ctx, dummies.CSMachine1)
	})

	Context("When creating a CloudStackMachine with all attributes", func() {
		It("Should succeed", func() {
			Expect(k8sClient.Create(ctx, dummies.CSMachine1)).Should(Succeed())
		})
	})

	Context("When creating a CloudStackMachine with missing Offering attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			dummies.CSMachine1.Spec.Offering = ""
			Expect(k8sClient.Create(ctx, dummies.CSMachine1).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Offering"))
		})
	})

	Context("When creating a CloudStackMachine with missing Template attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			dummies.CSMachine1.Spec.Template = ""
			Expect(k8sClient.Create(ctx, dummies.CSMachine1).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Template"))
		})
	})

	Context("When creating a CloudStackMachine with the wrong kind of IdentityReference", func() {
		It("Should be rejected by the validating webhooks", func() {
			dummies.CSMachine1.Spec.IdentityRef.Kind = "ConfigMap"
			Expect(k8sClient.Create(ctx, dummies.CSMachine1).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Forbidden\\: must be a Secret"))
		})
	})

	// Need the `-- not template` here to make the context unique. Apparently ginkgo uses startswith.
	Context("When updating a CloudStackMachine", func() {
		forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"

		BeforeEach(func() { // Reset test vars to initial state.
			Ω(k8sClient.Create(ctx, dummies.CSMachine1)).Should(Succeed())
		})

		It("should reject VM offering updates to the CloudStackMachine", func() {
			dummies.CSMachine1.Spec.Offering = "ArbitraryUpdateOffering"
			Ω(k8sClient.Update(ctx, dummies.CSMachine1).Error()).Should(MatchRegexp(forbiddenRegex, "offering"))
		})

		It("should reject VM template updates to the CloudStackMachine", func() {
			dummies.CSMachine1.Spec.Template = "ArbitraryUpdateTemplate"
			Ω(k8sClient.Update(ctx, dummies.CSMachine1).Error()).Should(MatchRegexp(forbiddenRegex, "template"))
		})

		It("should reject updates to VM details of the CloudStackMachine", func() {
			dummies.CSMachine1.Spec.Details = map[string]string{"memoryOvercommitRatio": "1.5"}
			Ω(k8sClient.Update(ctx, dummies.CSMachine1).Error()).Should(MatchRegexp(forbiddenRegex, "details"))
		})

		It("should reject identity reference kind udpates to the CloudStackMachine", func() {
			dummies.CSMachine1.Spec.IdentityRef.Kind = "ConfigMap"
			Ω(k8sClient.Update(ctx, dummies.CSMachine1).Error()).
				Should(MatchRegexp(forbiddenRegex, "identityRef\\.Kind"))
		})

		It("should reject identity reference name udpates to the CloudStackMachine", func() {
			dummies.CSMachine1.Spec.IdentityRef.Name = "IdentityConfigMap"
			Ω(k8sClient.Update(ctx, dummies.CSMachine1).Error()).
				Should(MatchRegexp(forbiddenRegex, "identityRef\\.Name"))
		})

		It("should reject udpates to the list of affinty groups of the CloudStackMachine", func() {
			dummies.CSMachine1.Spec.AffinityGroupIDs = []string{"28b907b8-75a7-4214-bd3d-6c61961fc2af"}
			Ω(k8sClient.Update(ctx, dummies.CSMachine1).Error()).
				Should(MatchRegexp(forbiddenRegex, "AffinityGroupIDs"))
		})
	})
})
