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
	"fmt"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloudStackCluster webhooks", func() {

	var ctx context.Context

	BeforeEach(func() { // Reset test vars to initial state.
		dummies.SetDummyVars()
		ctx = context.Background()
	})

	Context("When creating a CloudStackCluster with all validated attributes", func() {
		It("Should succeed", func() {
			Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(Succeed())
		})
	})

	Context("When creating a CloudStackCluster with missing Network attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			Ω(k8sClient.Create(ctx, dummies.CSCluster).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Network"))
		})
	})

	Context("When creating a CloudStackCluster with missing Zone attribute", func() {
		It("Should be rejected by the validating webhooks", func() {
			Ω(k8sClient.Create(ctx, dummies.CSCluster).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Required value\\: Zone"))
		})
	})

	Context("When creating a CloudStackCluster with the wrong kind of IdentityReference", func() {
		It("Should be rejected by the validating webhooks", func() {
			dummies.CSCluster.Spec.IdentityRef.Kind = "Wrong"
			Ω(k8sClient.Create(ctx, dummies.CSCluster).Error()).
				Should(MatchRegexp("admission webhook.*denied the request.*Forbidden\\: must be a Secret"))
		})
	})

	Context("When updating a CloudStackCluster", func() {
		type CSClusterModFunc func(*infrav1.CloudStackCluster)
		description := func() func(string, CSClusterModFunc) string {
			return func(field string, mod CSClusterModFunc) string {
				return fmt.Sprintf(
					"CloudStackCluster.Spec %s modification should be rejected by the validating webhooks", field)
			}
		}
		DescribeTable("Forbidden field modification",
			func(field string, mod CSClusterModFunc) {
				Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(Succeed())
				forbiddenRegex := "admission webhook.*denied the request.*Forbidden\\: %s"
				mod(dummies.CSCluster)
				Ω(k8sClient.Update(ctx, dummies.CSCluster).Error()).Should(MatchRegexp(forbiddenRegex, field))
			},
			Entry(description(), "zone", func(CSC *infrav1.CloudStackCluster) {
				CSC.Spec.Zones = []infrav1.Zone{dummies.Zone1}
			}),
			Entry(description(), "zonenetwork", func(CSC *infrav1.CloudStackCluster) {
				CSC.Spec.Zones[0].Network.Name = "ArbitraryNetworkName"
			}),
			Entry(description(), "controlplaneendpoint\\.host", func(CSC *infrav1.CloudStackCluster) {
				CSC.Spec.ControlPlaneEndpoint.Host = "1.1.1.1"
			}),
			Entry(description(), "identityRef\\.Kind", func(CSC *infrav1.CloudStackCluster) {
				CSC.Spec.IdentityRef.Kind = "ArbitraryKind"
			}),
			Entry(description(), "identityRef\\.Name", func(CSC *infrav1.CloudStackCluster) {
				CSC.Spec.IdentityRef.Name = "ArbitraryName"
			}),
			Entry(description(), "controlplaneendpoint\\.port", func(CSC *infrav1.CloudStackCluster) {
				CSC.Spec.ControlPlaneEndpoint.Port = int32(1234)
			}),
		)
	})
})
