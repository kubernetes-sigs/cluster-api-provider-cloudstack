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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1beta1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	v1beta2 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("Conversion", func() {
	BeforeEach(func() { // Reset test vars to initial state.
	})

	Context("GetFailureDomains function", func() {
		It("Converts v1beta1 cluster spec to v1beta2 failure domains", func() {
			csCluster := &v1beta1.CloudStackCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster1",
					Namespace: "namespace1",
				},
				Spec: v1beta1.CloudStackClusterSpec{
					Zones: []v1beta1.Zone{
						{
							ID: "76472a84-d23f-4e97-b154-ee1b975ed936",
							Network: v1beta1.Network{
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
				Status: v1beta1.CloudStackClusterStatus{},
			}
			failureDomains, err := v1beta1.GetFailureDomains(csCluster)
			expectedResult := []v1beta2.CloudStackFailureDomainSpec{
				{
					Name: "76472a84-d23f-4e97-b154-ee1b975ed936",
					Zone: v1beta2.CloudStackZoneSpec{
						ID:      "76472a84-d23f-4e97-b154-ee1b975ed936",
						Network: v1beta2.Network{Name: "network1"},
					},
					Account: "account1",
					Domain:  "domain1",
					ACSEndpoint: corev1.SecretReference{
						Name:      "global",
						Namespace: "namespace1",
					},
				},
			}
			Ω(err).ShouldNot(HaveOccurred())
			Ω(failureDomains).Should(Equal(expectedResult))
		})
	})
})
