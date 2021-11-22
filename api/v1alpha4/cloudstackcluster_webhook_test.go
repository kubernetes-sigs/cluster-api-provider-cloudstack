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

var _ = Describe("CloudStackCluster webhooks", func() {
	Context("When creating a CloudStackCluster with all validated attributes", func() {
		It("Should succeed", func() {
			ctx := context.Background()
			cloudStackCluster := &CloudStackCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackCluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: CloudStackClusterSpec{
					IdentityRef: &CloudStackIdentityReference{
						Kind: defaultIdentityRefKind,
						Name: "IdentitySecret",
					},
					Zone:    "Zone",
					Network: "Network",
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackCluster)).Should(Succeed())
		})
	})

	Context("When creating a CloudStackCluster with missing Network attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			cloudStackCluster := &CloudStackCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackCluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: CloudStackClusterSpec{
					IdentityRef: &CloudStackIdentityReference{
						Kind: defaultIdentityRefKind,
						Name: "IdentitySecret",
					},
					Zone: "Zone",
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackCluster).Error()).Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Network"))
		})
	})

	Context("When creating a CloudStackCluster with missing Zone attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			cloudStackCluster := &CloudStackCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackCluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: CloudStackClusterSpec{
					IdentityRef: &CloudStackIdentityReference{
						Kind: defaultIdentityRefKind,
						Name: "IdentitySecret",
					},
					Network: "Network",
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackCluster).Error()).Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Zone"))
		})
	})

	Context("When creating a CloudStackCluster with the wrong kind of IdentityReference", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			cloudStackCluster := &CloudStackCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
					Kind:       "CloudStackCluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: CloudStackClusterSpec{
					IdentityRef: &CloudStackIdentityReference{
						Kind: "ConfigMap",
						Name: "IdentityConfigMap",
					},
					Zone:    "Zone",
					Network: "Network",
				},
			}
			Expect(k8sClient.Create(ctx, cloudStackCluster).Error()).Should(MatchRegexp("admission webhook.*denied the request.*Forbidden\\: must be a Secret"))
		})

		Context("When updating a CloudStackCluster", func() {
			It("Should be rejected by the validating webhooks", func() {
				ctx := context.Background()
				cloudStackCluster := &CloudStackCluster{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
						Kind:       "CloudStackCluster",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster2",
						Namespace: "default",
					},
					Spec: CloudStackClusterSpec{
						IdentityRef: &CloudStackIdentityReference{
							Kind: defaultIdentityRefKind,
							Name: "IdentitySecret",
						},
						Zone:    "Zone",
						Network: "Network",
					},
				}
				Expect(k8sClient.Create(ctx, cloudStackCluster)).Should(Succeed())

				forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"
				cloudStackClusterUpdate := &CloudStackCluster{}

				cloudStackCluster.DeepCopyInto(cloudStackClusterUpdate)
				cloudStackClusterUpdate.Spec.Zone = "Zone2"
				Expect(k8sClient.Update(ctx, cloudStackClusterUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "zone"))

				cloudStackCluster.DeepCopyInto(cloudStackClusterUpdate)
				cloudStackClusterUpdate.Spec.Network = "Network2"
				Expect(k8sClient.Update(ctx, cloudStackClusterUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "network"))

				cloudStackCluster.DeepCopyInto(cloudStackClusterUpdate)
				cloudStackClusterUpdate.Spec.IdentityRef.Kind = "ConfigMap"
				Expect(k8sClient.Update(ctx, cloudStackClusterUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "identityRef\\.Kind"))

				cloudStackCluster.DeepCopyInto(cloudStackClusterUpdate)
				cloudStackClusterUpdate.Spec.IdentityRef.Name = "IdentityConfigMap"
				Expect(k8sClient.Update(ctx, cloudStackClusterUpdate).Error()).Should(MatchRegexp(forbiddenRegex, "identityRef\\.Name"))
			})
		})
	})
})
