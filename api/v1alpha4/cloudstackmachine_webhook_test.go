/*
Copyright 2021.

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

package v1alpha4

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CloudStackMachine webhook", func() {
	Context("When creating a CloudStackMachine with all validated attributes", func() {
		It("Should succeed", func() {
			ctx := context.Background()
			cloudStackMachine := &CloudStackMachine{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackMachine",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machine",
					Namespace: "default",
				},
				Spec: CloudStackMachineSpec{
					IdentityRef: &CloudStackIdentityReference{
						Kind: defaultIdentityRefKind,
						Name: "IdentitySecret",
					},
					Template: "Template",
					Offering: "Offering",
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackMachine)).Should(Succeed())
		})
	})

	Context("When creating a CloudStackMachine with missing Offering attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			cloudStackMachine := &CloudStackMachine{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackMachine",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machine",
					Namespace: "default",
				},
				Spec: CloudStackMachineSpec{
					IdentityRef: &CloudStackIdentityReference{
						Kind: defaultIdentityRefKind,
						Name: "IdentitySecret",
					},
					Template: "Template",
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackMachine).Error()).Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Offering"))
		})
	})

	Context("When creating a CloudStackMachine with missing Template attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			cloudStackMachine := &CloudStackMachine{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackMachine",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machine",
					Namespace: "default",
				},
				Spec: CloudStackMachineSpec{
					IdentityRef: &CloudStackIdentityReference{
						Kind: defaultIdentityRefKind,
						Name: "IdentitySecret",
					},
					Offering: "Offering",
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackMachine).Error()).Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Template"))
		})
	})

	Context("When creating a CloudStackMachine with the wrong kind of IdentityReference", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			cloudStackMachine := &CloudStackMachine{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackMachine",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machine",
					Namespace: "default",
				},
				Spec: CloudStackMachineSpec{
					IdentityRef: &CloudStackIdentityReference{
						Kind: "ConfigMap",
						Name: "IdentityConfigMap",
					},
					Template: "Template",
					Offering: "Offering",
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackMachine).Error()).Should(MatchRegexp("admission webhook.*denied the request.*Forbidden\\: must be a Secret"))
		})
	})

	Context("When updating a CloudStackMachine", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			cloudStackMachine := &CloudStackMachine{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackMachine",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machine2",
					Namespace: "default",
				},
				Spec: CloudStackMachineSpec{
					IdentityRef: &CloudStackIdentityReference{
						Kind: defaultIdentityRefKind,
						Name: "IdentitySecret",
					},
					Template: "Template",
					Offering: "Offering",
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackMachine)).Should(Succeed())

			forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"
			cloudStackMachineUpdate := &CloudStackMachine{}

			cloudStackMachine.DeepCopyInto(cloudStackMachineUpdate)
			cloudStackMachineUpdate.Spec.Template = "Template2"
			Expect(k8sClient.Update(ctx, cloudStackMachineUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "template"))

			cloudStackMachine.DeepCopyInto(cloudStackMachineUpdate)
			cloudStackMachineUpdate.Spec.Offering = "Offering2"
			Expect(k8sClient.Update(ctx, cloudStackMachineUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "offering"))

			cloudStackMachine.DeepCopyInto(cloudStackMachineUpdate)
			cloudStackMachineUpdate.Spec.IdentityRef.Kind = "ConfigMap"
			Expect(k8sClient.Update(ctx, cloudStackMachineUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "identityRef\\.Kind"))

			cloudStackMachine.DeepCopyInto(cloudStackMachineUpdate)
			cloudStackMachineUpdate.Spec.IdentityRef.Name = "IdentityConfigMap"
			Expect(k8sClient.Update(ctx, cloudStackMachineUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "identityRef\\.Name"))
		})
	})
})
