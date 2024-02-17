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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/annotations"
)

var _ = Describe("CloudStackCluster webhooks", func() {
	var ctx context.Context
	forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"
	requiredRegex := "admission webhook.*denied the request.*Required value\\: %s"

	BeforeEach(func() { // Reset test vars to initial state.
		ctx = context.Background()
		dummies.SetDummyVars()                       // Reset cluster var.
		_ = k8sClient.Delete(ctx, dummies.CSCluster) // Delete any remnants.
		dummies.SetDummyVars()                       // Reset again since the k8s client can set this on delete.
	})

	Context("When creating a CloudStackCluster", func() {
		It("Should accept a CloudStackCluster with all attributes present", func() {
			Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(Succeed())
		})

		It("Should reject a CloudStackCluster with missing Zones.Network attribute", func() {
			dummies.CSCluster.Spec.FailureDomains = []infrav1.CloudStackFailureDomainSpec{{}}
			dummies.CSCluster.Spec.FailureDomains[0].Zone.Name = "ZoneWNoNetwork"
			Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(
				MatchError(MatchRegexp(requiredRegex, "each Zone requires a Network specification")))
		})

		It("Should reject a CloudStackCluster with missing Zone attribute", func() {
			dummies.CSCluster.Spec.FailureDomains[0].Zone = infrav1.CloudStackZoneSpec{}
			Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(MatchError(MatchRegexp(requiredRegex,
				"each Zone requires a Network specification")))
		})
	})

	Context("When updating a CloudStackCluster", func() {
		BeforeEach(func() {
			Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(Succeed())
		})

		It("Should reject updates to CloudStackCluster FailureDomains", func() {
			dummies.CSCluster.Spec.FailureDomains[0].Zone.Name = "SomeRandomUpdate"
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).Should(MatchError(MatchRegexp(forbiddenRegex, "Cannot change FailureDomain")))
		})
		It("Should reject updates to Networks specified in CloudStackCluster Zones", func() {
			dummies.CSCluster.Spec.FailureDomains[0].Zone.Network.Name = "ArbitraryUpdateNetworkName"
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).Should(MatchError(MatchRegexp(forbiddenRegex, "Cannot change FailureDomain")))
		})
		It("Should reject updates to CloudStackCluster controlplaneendpoint.host", func() {
			dummies.CSCluster.Spec.ControlPlaneEndpoint.Host = "1.1.1.1"
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).
				Should(MatchError(MatchRegexp(forbiddenRegex, "controlplaneendpoint\\.host")))
		})

		It("Should reject updates to CloudStackCluster controlplaneendpoint.port", func() {
			dummies.CSCluster.Spec.ControlPlaneEndpoint.Port = int32(1234)
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).
				Should(MatchError(MatchRegexp(forbiddenRegex, "controlplaneendpoint\\.port")))
		})
	})

	Context("When updating a CloudStackCluster's annotations", func() {
		It("Should reject removal of externally managed ('managed-by') annotation from CloudStackCluster", func() {
			// Create a CloudStackCluster with managed-by annotation
			annotations.AddAnnotations(dummies.CSCluster, map[string]string{clusterv1.ManagedByAnnotation: ""})
			Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(Succeed())

			// Remove the annotation and update CloudStackCluster
			dummies.CSCluster.Annotations = make(map[string]string)
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).
				Should(MatchError(MatchRegexp(forbiddenRegex, "removal of externally managed")))
		})

		It("Should allow adding of externally managed ('managed-by') annotation to CloudStackCluster", func() {
			Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(Succeed())

			annotations.AddAnnotations(dummies.CSCluster, map[string]string{clusterv1.ManagedByAnnotation: ""})
			Ω(k8sClient.Update(ctx, dummies.CSCluster)).Should(Succeed())
		})
	})
})
