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

package v1beta1

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CloudStackCluster webhooks", func() {
	const (
		apiVersion         = "infrastructure.cluster.x-k8s.io/v1beta1"
		clusterKind        = "CloudStackCluster"
		clusterName        = "test-cluster"
		clusterNamespace   = "default"
		clusterID          = "0"
		identitySecretName = "IdentitySecret"
		zoneName           = "Zone"
		network            = "Network"
	)

	var ( // Shared base test vars.
		cloudStackCluster *CloudStackCluster
		testZone1         Zone
		testZone2         Zone
	)

	BeforeEach(func() { // Reset test vars to initial state.
		testZone1 = Zone{Name: zoneName}
		testZone2 = Zone{Name: zoneName}
		cloudStackCluster = &CloudStackCluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: apiVersion,
				Kind:       clusterKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: clusterNamespace,
			},
			Spec: CloudStackClusterSpec{
				IdentityRef: &CloudStackIdentityReference{
					Kind: defaultIdentityRefKind,
					Name: identitySecretName,
				},
				Zones: []Zone{testZone1, testZone2},
			},
		}

	})

	Context("When creating a CloudStackCluster with all validated attributes", func() {
		It("Should succeed", func() {
			ctx := context.Background()
			Ω(k8sClient.Create(ctx, cloudStackCluster)).Should(Succeed())
		})
	})

	Context("When creating a CloudStackCluster with missing Network attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			Ω(k8sClient.Create(ctx, cloudStackCluster).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Network"))
		})
	})

	Context("When creating a CloudStackCluster with missing Zone attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			Ω(k8sClient.Create(ctx, cloudStackCluster).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Zone"))
		})
	})

	Context("When creating a CloudStackCluster with the wrong kind of IdentityReference", func() {
		const (
			configMapKind = "ConfigMap"
			configMapName = "IdentityConfigMap"
		)

		It("Should be rejected by the validating webhooks", func() {
			ctx := context.Background()
			Ω(k8sClient.Create(ctx, cloudStackCluster).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Forbidden\\: must be a Secret"))
		})

		Context("When updating a CloudStackCluster", func() {
			var (
				ctx                     context.Context
				cloudStackCluster       *CloudStackCluster
				cloudStackClusterUpdate *CloudStackCluster
			)

			BeforeEach(func() {
				ctx = context.Background()
				cloudStackClusterUpdate = &CloudStackCluster{}
			})

			It("Should be rejected by the validating webhooks", func() {
				Ω(k8sClient.Create(ctx, cloudStackCluster)).Should(Succeed())

				forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"

				cloudStackCluster.DeepCopyInto(cloudStackClusterUpdate)
				cloudStackClusterUpdate.Spec.Zones = []Zone{testZone2}
				Ω(k8sClient.Update(ctx, cloudStackClusterUpdate).Error()).
					Should(MatchRegexp(forbiddenRegex, "zone"))

				cloudStackCluster.DeepCopyInto(cloudStackClusterUpdate)
				//cloudStackClusterUpdate.Spec.Network = "Network2"
				Ω(k8sClient.Update(ctx, cloudStackClusterUpdate).Error()).
					Should(MatchRegexp(forbiddenRegex, "network"))

				cloudStackCluster.DeepCopyInto(cloudStackClusterUpdate)
				cloudStackClusterUpdate.Spec.ControlPlaneEndpoint.Host = "1.1.1.1"
				Ω(k8sClient.Update(ctx, cloudStackClusterUpdate).Error()).
					Should(MatchRegexp(forbiddenRegex, "controlplaneendpointhost"))

				cloudStackCluster.DeepCopyInto(cloudStackClusterUpdate)
				cloudStackClusterUpdate.Spec.IdentityRef.Kind = "ConfigMap"
				Ω(k8sClient.Update(ctx, cloudStackClusterUpdate).Error()).
					Should(MatchRegexp(forbiddenRegex, "identityRef\\.Kind"))

				cloudStackCluster.DeepCopyInto(cloudStackClusterUpdate)
				cloudStackClusterUpdate.Spec.IdentityRef.Name = configMapName
				Ω(k8sClient.Update(ctx, cloudStackClusterUpdate).Error()).
					Should(MatchRegexp(forbiddenRegex, "identityRef\\.Name"))
			})

			It("Should reject changing the port", func() {
				cloudStackCluster.DeepCopyInto(cloudStackClusterUpdate)
				cloudStackClusterUpdate.Spec.ControlPlaneEndpoint.Port = int32(1234)
				Ω(k8sClient.Update(ctx, cloudStackClusterUpdate)).ShouldNot(Succeed())
			})
		})
	})
})
