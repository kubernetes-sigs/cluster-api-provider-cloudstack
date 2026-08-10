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
	gomega "github.com/onsi/gomega"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta4"
)

var _ = ginkgo.Describe("CloudStackMachine webhook", func() {
	var ctx context.Context

	const (
		forbiddenFmt = "admission webhook.*denied the request.*Forbidden\\: %s"
		requiredFmt  = "admission webhook.*denied the request.*Required value\\: %s"
	)

	ginkgo.BeforeEach(func() {
		dummies.SetDummyVars()
		ctx = context.Background()
		_ = k8sClient.Delete(ctx, dummies.CSMachine1)
	})

	ginkgo.DescribeTable("create validation",
		func(mutate func(), wantErrRegex string) {
			mutate()
			err := k8sClient.Create(ctx, dummies.CSMachine1)
			if wantErrRegex != "" {
				gomega.Expect(err).To(gomega.MatchError(gomega.MatchRegexp(wantErrRegex)))
			} else {
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}
		},
		ginkgo.Entry("accepts machine with all attributes", func() {}, ""),
		ginkgo.Entry("accepts machine with disk offering",
			func() { dummies.CSMachine1.Spec.DiskOffering = dummies.DiskOffering },
			"",
		),
		ginkgo.Entry("accepts machine with positive disk offering size",
			func() {
				dummies.CSMachine1.Spec.DiskOffering = dummies.DiskOffering
				dummies.CSMachine1.Spec.DiskOffering.CustomSize = 1
			},
			"",
		),
		ginkgo.Entry("rejects machine with negative disk offering size",
			func() {
				dummies.CSMachine1.Spec.DiskOffering = dummies.DiskOffering
				dummies.CSMachine1.Spec.DiskOffering.CustomSize = -1
			},
			fmt.Sprintf(forbiddenFmt, "customSizeInGB"),
		),
		ginkgo.Entry("rejects machine missing Offering",
			func() { dummies.CSMachine1.Spec.Offering = infrav1.CloudStackResourceIdentifier{} },
			fmt.Sprintf(requiredFmt, "Offering"),
		),
		ginkgo.Entry("rejects machine missing Template",
			func() { dummies.CSMachine1.Spec.Template = infrav1.CloudStackResourceIdentifier{} },
			fmt.Sprintf(requiredFmt, "Template"),
		),
	)

	ginkgo.Describe("update validation", func() {
		ginkgo.BeforeEach(func() {
			gomega.Ω(k8sClient.Create(ctx, dummies.CSMachine1)).Should(gomega.Succeed())
		})

		ginkgo.DescribeTable("immutable fields are rejected",
			func(mutate func(), wantErrRegex string) {
				mutate()
				gomega.Ω(k8sClient.Update(ctx, dummies.CSMachine1)).
					Should(gomega.MatchError(gomega.MatchRegexp(wantErrRegex)))
			},
			ginkgo.Entry("rejects offering update",
				func() {
					dummies.CSMachine1.Spec.Offering = infrav1.CloudStackResourceIdentifier{Name: "ArbitraryUpdateOffering"}
				},
				fmt.Sprintf(forbiddenFmt, "offering"),
			),
			ginkgo.Entry("rejects template update",
				func() {
					dummies.CSMachine1.Spec.Template = infrav1.CloudStackResourceIdentifier{Name: "ArbitraryUpdateTemplate"}
				},
				fmt.Sprintf(forbiddenFmt, "template"),
			),
			ginkgo.Entry("rejects disk offering update",
				func() { dummies.CSMachine1.Spec.DiskOffering.Name = "medium" },
				fmt.Sprintf(forbiddenFmt, "diskOffering"),
			),
			ginkgo.Entry("rejects details update",
				func() { dummies.CSMachine1.Spec.Details = map[string]string{"memoryOvercommitRatio": "1.5"} },
				fmt.Sprintf(forbiddenFmt, "details"),
			),
			ginkgo.Entry("rejects AffinityGroupIDs update",
				func() { dummies.CSMachine1.Spec.AffinityGroupIDs = []string{"28b907b8-75a7-4214-bd3d-6c61961fc2af"} },
				fmt.Sprintf(forbiddenFmt, "AffinityGroupIDs"),
			),
		)
	})
})
