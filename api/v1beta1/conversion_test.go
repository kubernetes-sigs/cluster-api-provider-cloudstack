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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("Conversion", func() {
	BeforeEach(func() { // Reset test vars to initial state.
	})

	Context("GetFailureDomains function", func() {
		It("Return the correct value when the last state update time is known", func() {
			csCluster := &infrav1.CloudStackCluster{
				Spec: infrav1.CloudStackClusterSpec{
					Zones: []infrav1.Zone{
						{
							Name: "zone-name1",
							Network: infrav1.Network{
								Name: "network1",
							},
						},
						{
							ID: "76472a84-d23f-4e97-b154-ee1b975ed936",
							Network: infrav1.Network{
								Name: "network1",
							},
						},
					},
					ControlPlaneEndpoint: capiv1.APIEndpoint{
						Host: "endpoint1",
						Port: 443,
					},
					Account: "account1",
					Domain:  "domain1",
				},
				Status: infrav1.CloudStackClusterStatus{},
			}
			failureDomains, err := infrav1.GetFailureDomains(csCluster)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(failureDomains).Should(HaveLen(2))
		})
	})
})
