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

var _ = Describe("CloudStackMachineTemplate webhook", func() {
	Context("When creating a CloudStackMachineTemplate with all validated attributes", func() {
		It("Should succeed", func() {
			ctx := context.Background()
			cloudStackMachineTemplate := &CloudStackMachineTemplate{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackMachineTemplate",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machinetemplate",
					Namespace: "default",
				},
				Spec: CloudStackMachineTemplateSpec{
					Spec: CloudStackMachineTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-machinetemplateresource",
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
					},
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackMachineTemplate)).Should(Succeed())
		})
	})

	Context("When creating a CloudStackMachineTemplate with all attributes", func() {
		It("Should succeed", func() {
			ctx := context.Background()
			cloudStackMachineTemplate := &CloudStackMachineTemplate{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackMachineTemplate",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machinetemplate-2",
					Namespace: "default",
				},
				Spec: CloudStackMachineTemplateSpec{
					Spec: CloudStackMachineTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-machinetemplateresource",
							Namespace: "default",
						},
						Spec: CloudStackMachineSpec{
							IdentityRef: &CloudStackIdentityReference{
								Kind: defaultIdentityRefKind,
								Name: "IdentitySecret",
							},
							Template: "Template",
							Offering: "Offering",
							Details: map[string]string{
								"memoryOvercommitRatio": "1.2",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackMachineTemplate)).Should(Succeed())
		})
	})

	Context("When creating a CloudStackMachineTemplate with missing Offering attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			cloudStackMachineTemplate := &CloudStackMachineTemplate{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackMachineTemplate",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machinetemplate",
					Namespace: "default",
				},
				Spec: CloudStackMachineTemplateSpec{
					Spec: CloudStackMachineTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-machinetemplateresource",
							Namespace: "default",
						},
						Spec: CloudStackMachineSpec{
							IdentityRef: &CloudStackIdentityReference{
								Kind: defaultIdentityRefKind,
								Name: "IdentitySecret",
							},
							Template: "Template",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackMachineTemplate).Error()).Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Offering"))
		})
	})

	Context("When creating a CloudStackMachineTemplate with missing Template attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			cloudStackMachineTemplate := &CloudStackMachineTemplate{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackMachineTemplate",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machinetemplate",
					Namespace: "default",
				},
				Spec: CloudStackMachineTemplateSpec{
					Spec: CloudStackMachineTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-machinetemplateresource",
							Namespace: "default",
						},
						Spec: CloudStackMachineSpec{
							IdentityRef: &CloudStackIdentityReference{
								Kind: defaultIdentityRefKind,
								Name: "IdentitySecret",
							},
							Offering: "Offering",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackMachineTemplate).Error()).Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Template"))
		})
	})

	Context("When creating a CloudStackMachineTemplate with the wrong kind of IdentityReference", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			cloudStackMachineTemplate := &CloudStackMachineTemplate{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackMachineTemplate",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machinetemplate",
					Namespace: "default",
				},
				Spec: CloudStackMachineTemplateSpec{
					Spec: CloudStackMachineTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-machinetemplateresource",
							Namespace: "default",
						},
						Spec: CloudStackMachineSpec{
							IdentityRef: &CloudStackIdentityReference{
								Kind: "ConfigMap",
								Name: "IdentitySecret",
							},
							Offering: "Offering",
							Template: "Template",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackMachineTemplate).Error()).Should(MatchRegexp("admission webhook.*denied the request.*Forbidden\\: must be a Secret"))
		})
	})

	Context("When updating a CloudStackMachine", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			cloudStackMachineTemplate := &CloudStackMachineTemplate{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackMachineTemplate",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machinetemplate-3",
					Namespace: "default",
				},
				Spec: CloudStackMachineTemplateSpec{
					Spec: CloudStackMachineTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-machinetemplateresource",
							Namespace: "default",
						},
						Spec: CloudStackMachineSpec{
							IdentityRef: &CloudStackIdentityReference{
								Kind: defaultIdentityRefKind,
								Name: "IdentitySecret",
							},
							Template: "Template",
							Offering: "Offering",
							Details: map[string]string{
								"memoryOvercommitRatio": "1.2",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackMachineTemplate)).Should(Succeed())

			forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"
			cloudStackMachineTemplateUpdate := &CloudStackMachineTemplate{}

			cloudStackMachineTemplate.DeepCopyInto(cloudStackMachineTemplateUpdate)
			cloudStackMachineTemplateUpdate.Spec.Spec.Spec.Template = "Template2"
			Expect(k8sClient.Update(ctx, cloudStackMachineTemplateUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "template"))

			cloudStackMachineTemplate.DeepCopyInto(cloudStackMachineTemplateUpdate)
			cloudStackMachineTemplateUpdate.Spec.Spec.Spec.Offering = "Offering2"
			Expect(k8sClient.Update(ctx, cloudStackMachineTemplateUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "offering"))

			cloudStackMachineTemplate.DeepCopyInto(cloudStackMachineTemplateUpdate)
			cloudStackMachineTemplateUpdate.Spec.Spec.Spec.Details = map[string]string{
				"memoryOvercommitRatio": "1.5",
			}
			Expect(k8sClient.Update(ctx, cloudStackMachineTemplateUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "details"))

			cloudStackMachineTemplate.DeepCopyInto(cloudStackMachineTemplateUpdate)
			cloudStackMachineTemplateUpdate.Spec.Spec.Spec.IdentityRef.Kind = "ConfigMap"
			Expect(k8sClient.Update(ctx, cloudStackMachineTemplateUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "identityRef\\.Kind"))

			cloudStackMachineTemplate.DeepCopyInto(cloudStackMachineTemplateUpdate)
			cloudStackMachineTemplateUpdate.Spec.Spec.Spec.IdentityRef.Name = "IdentityConfigMap"
			Expect(k8sClient.Update(ctx, cloudStackMachineTemplateUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "identityRef\\.Name"))
		})
	})
})
