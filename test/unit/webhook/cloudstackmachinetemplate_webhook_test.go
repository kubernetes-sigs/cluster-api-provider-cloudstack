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

	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"

	"sigs.k8s.io/cluster-api-provider-cloudstack-staging/test/unit/dummies"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloudStackMachineTemplate webhook", func() {
	var ctx context.Context
	forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"
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
			dummies.CSMachineTemplate1.Spec.Spec.Spec.DiskOffering = v1beta1.CloudStackResourceDiskOffering{
				CloudStackResourceIdentifier: v1beta1.CloudStackResourceIdentifier{Name: "", ID: ""},
			}
			Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).Should(Succeed())
		})

		It("Should reject a CloudStackMachineTemplate when missing the VM Offering attribute", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.Offering = v1beta1.CloudStackResourceIdentifier{Name: "", ID: ""}
			Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).
				Should(MatchError(MatchRegexp(requiredRegex, "Offering")))
		})

		It("Should reject a CloudStackMachineTemplate when missing the VM Template attribute", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.Template = v1beta1.CloudStackResourceIdentifier{Name: "", ID: ""}
			Expect(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).
				Should(MatchError(MatchRegexp(requiredRegex, "Template")))
		})

		It("should reject a CloudStackMachineTemplate with IdentityRef not of kind 'Secret'", func() {
			dummies.CSMachine1.Spec.IdentityRef.Kind = "ConfigMap"
			Expect(k8sClient.Create(ctx, dummies.CSMachine1)).
				Should(MatchError(MatchRegexp(forbiddenRegex, "must be a Secret")))
		})
	})

	Context("When updating a CloudStackMachineTemplate", func() {
		BeforeEach(func() { // Reset test vars to initial state.
			Ω(k8sClient.Create(ctx, dummies.CSMachineTemplate1)).Should(Succeed())
		})

		It("should reject VM template updates to the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.Template = v1beta1.CloudStackResourceIdentifier{Name: "ArbitraryUpdateTemplate"}
			Ω(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).
				Should(MatchError(MatchRegexp(forbiddenRegex, "template")))
		})

		It("should reject VM disk offering updates to the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.DiskOffering = v1beta1.CloudStackResourceDiskOffering{
				CloudStackResourceIdentifier: v1beta1.CloudStackResourceIdentifier{Name: "DiskOffering2"}}
			Ω(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).
				Should(MatchError(MatchRegexp(forbiddenRegex, "diskOffering")))
		})

		It("should reject VM offering updates to the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.Offering = v1beta1.CloudStackResourceIdentifier{Name: "Offering2"}
			Ω(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).
				Should(MatchError(MatchRegexp(forbiddenRegex, "offering")))
		})

		It("should reject updates to VM details of the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.Details = map[string]string{"memoryOvercommitRatio": "1.5"}
			Ω(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).
				Should(MatchError(MatchRegexp(forbiddenRegex, "details")))
		})

		It("should reject identity reference kind updates to the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.IdentityRef.Kind = "configMap"
			Ω(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).
				Should(MatchError(MatchRegexp(forbiddenRegex, "identityRef\\.Kind")))
		})

		It("should reject identity reference name updates to the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.IdentityRef.Name = "IDentityConfigMap"
			Ω(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).
				Should(MatchError(MatchRegexp(forbiddenRegex, "identityRef\\.Name")))
		})

		It("should reject updates to the list of AffinityGroupIDs of the CloudStackMachineTemplate", func() {
			dummies.CSMachineTemplate1.Spec.Spec.Spec.AffinityGroupIDs = []string{"28b907b8-75a7-4214-bd3d-6c61961fc2ag"}
			Ω(k8sClient.Update(ctx, dummies.CSMachineTemplate1)).ShouldNot(Succeed())
		})
	})
})
