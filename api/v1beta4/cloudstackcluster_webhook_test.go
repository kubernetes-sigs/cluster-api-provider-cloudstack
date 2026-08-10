/*
Copyright 2024 The Kubernetes Authors.

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

package v1beta4_test

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta4"
)

var _ = ginkgo.Describe("CloudStackCluster webhooks", func() {
	var ctx context.Context

	const (
		forbiddenFmt = "admission webhook.*denied the request.*Forbidden\\: %s"
		requiredFmt  = "admission webhook.*denied the request.*Required value\\: %s"
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
		dummies.SetDummyVars()
		_ = k8sClient.Delete(ctx, dummies.CSCluster)
		dummies.SetDummyVars()
	})

	ginkgo.DescribeTable("create validation",
		func(mutate func(), wantErrRegex string) {
			mutate()
			err := k8sClient.Create(ctx, dummies.CSCluster)
			if wantErrRegex != "" {
				gomega.Expect(err).To(gomega.MatchError(gomega.MatchRegexp(wantErrRegex)))
			} else {
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}
		},
		ginkgo.Entry("accepts cluster with all attributes present", func() {}, ""),
		ginkgo.Entry("rejects cluster missing Zones.Network",
			func() {
				dummies.CSCluster.Spec.FailureDomains = []infrav1.CloudStackFailureDomainSpec{{}}
				dummies.CSCluster.Spec.FailureDomains[0].Zone.Name = "ZoneWNoNetwork"
			},
			fmt.Sprintf(requiredFmt, "each Zone requires a Network specification"),
		),
		ginkgo.Entry("rejects cluster with empty Zone spec",
			func() {
				dummies.CSCluster.Spec.FailureDomains[0].Zone = infrav1.CloudStackZoneSpec{}
			},
			fmt.Sprintf(requiredFmt, "each Zone requires a Network specification"),
		),
	)

	ginkgo.Describe("update validation", func() {
		ginkgo.BeforeEach(func() {
			gomega.Expect(k8sClient.Create(ctx, dummies.CSCluster)).Should(gomega.Succeed())
		})

		ginkgo.DescribeTable("immutable fields are rejected",
			func(mutate func(), wantErrRegex string) {
				mutate()
				gomega.Expect(k8sClient.Update(ctx, dummies.CSCluster)).
					To(gomega.MatchError(gomega.MatchRegexp(wantErrRegex)))
			},
			ginkgo.Entry("rejects FailureDomain name change",
				func() { dummies.CSCluster.Spec.FailureDomains[0].Zone.Name = "SomeRandomUpdate" },
				fmt.Sprintf(forbiddenFmt, "Cannot change FailureDomain"),
			),
			ginkgo.Entry("rejects Zone Network name change",
				func() { dummies.CSCluster.Spec.FailureDomains[0].Zone.Network.Name = "ArbitraryUpdateNetworkName" },
				fmt.Sprintf(forbiddenFmt, "Cannot change FailureDomain"),
			),
			ginkgo.Entry("rejects controlplaneendpoint.host change",
				func() { dummies.CSCluster.Spec.ControlPlaneEndpoint.Host = "1.1.1.1" },
				fmt.Sprintf(forbiddenFmt, "controlplaneendpoint\\.host"),
			),
			ginkgo.Entry("rejects controlplaneendpoint.port change",
				func() { dummies.CSCluster.Spec.ControlPlaneEndpoint.Port = int32(1234) },
				fmt.Sprintf(forbiddenFmt, "controlplaneendpoint\\.port"),
			),
		)
	})
})
