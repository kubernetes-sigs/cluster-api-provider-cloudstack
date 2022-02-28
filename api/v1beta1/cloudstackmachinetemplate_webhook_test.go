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

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloudStackMachineTemplate webhook", func() {
	var ctx context.Context

	BeforeEach(func() { // Reset test vars to initial state.
		dummies.SetDummyVars()
		ctx = context.Background()
	})

	Context("When creating a CloudStackMachineTemplate with all attributes", func() {
		It("Should succeed", func() {
			Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).Should(Succeed())
		})
	})

	Context("When creating a CloudStackMachineTemplate with missing Offering attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			dummies.CSMachine1.Spec.Offering = ""
			Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Offering"))
		})
	})

	Context("When creating a CloudStackMachineTemplate with missing Template attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			dummies.CSMachine1.Spec.Template = ""
			Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Template"))
		})
	})

	Context("When creating a CloudStackMachineTemplate with the wrong kind of IdentityReference", func() {
		It("Should be rejected by the validating webhooks", func() {
			dummies.CSMachine1.Spec.IdentityRef.Kind = "ConfigMap"
			Expect(k8sClient.Create(ctx, dummies.CSMachine1).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Forbidden\\: must be a Secret"))
		})
	})

	Context("When updating a CloudStackMachine", func() {
		It("Should be rejected by the validating webhooks", func() {
			Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).Should(Succeed())

			forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"
			cloudStackMachineTemplateUpdate := &infrav1.CloudStackMachineTemplate{}

			dummies.CSMachineTemplate1.DeepCopyInto(cloudStackMachineTemplateUpdate)
			cloudStackMachineTemplateUpdate.Spec.Spec.Spec.Template = "Template2"
			Expect(k8sClient.Update(ctx, cloudStackMachineTemplateUpdate).Error()).
				Should(MatchRegexp(forbiddenRegex, "template"))

			dummies.CSMachineTemplate1.DeepCopyInto(cloudStackMachineTemplateUpdate)
			cloudStackMachineTemplateUpdate.Spec.Spec.Spec.Offering = "Offering2"
			Expect(k8sClient.Update(ctx, cloudStackMachineTemplateUpdate).Error()).
				Should(MatchRegexp(forbiddenRegex, "offering"))

			dummies.CSMachineTemplate1.DeepCopyInto(cloudStackMachineTemplateUpdate)
			cloudStackMachineTemplateUpdate.Spec.Spec.Spec.Details = map[string]string{"memoryOvercommitRatio": "1.5"}
			Expect(k8sClient.Update(ctx, cloudStackMachineTemplateUpdate).Error()).
				Should(MatchRegexp(forbiddenRegex, "details"))

			dummies.CSMachineTemplate1.DeepCopyInto(cloudStackMachineTemplateUpdate)
			cloudStackMachineTemplateUpdate.Spec.Spec.Spec.IdentityRef.Kind = "configMap"
			Expect(k8sClient.Update(ctx, cloudStackMachineTemplateUpdate).Error()).
				Should(MatchRegexp(forbiddenRegex, "identityRef\\.Kind"))

			dummies.CSMachineTemplate1.DeepCopyInto(cloudStackMachineTemplateUpdate)
			cloudStackMachineTemplateUpdate.Spec.Spec.Spec.IdentityRef.Name = "IDentityConfigMap"
			Expect(k8sClient.Update(ctx, cloudStackMachineTemplateUpdate).Error()).
				Should(MatchRegexp(forbiddenRegex, "identityRef\\.Name"))

			dummies.CSMachineTemplate1.DeepCopyInto(cloudStackMachineTemplateUpdate)
			cloudStackMachineTemplateUpdate.Spec.Spec.Spec.AffinityGroupIDs = []string{"28b907b8-75a7-4214-bd3d-6c61961fc2ag"}
			Expect(k8sClient.Update(ctx, cloudStackMachineTemplateUpdate)).
				ShouldNot(Succeed())
		})
	})
})
