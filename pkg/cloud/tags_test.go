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

package cloud

import (
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Tag Unit Tests", func() {
	var (
		cluster *infrav1.CloudStackCluster
	)

	BeforeEach(func() {
		cluster = &infrav1.CloudStackCluster{
			Spec: infrav1.CloudStackClusterSpec{
				Zone: "Zone1", Network: "SharedGuestNet1",
			},
			ObjectMeta: metav1.ObjectMeta{
				UID: "0",
			},
		}
	})

	Context("Tag Integ Tests", func() {
		client, connectionErr := NewClient("../../cloud-config")

		const (
			tagKey   = "test_tag"
			tagValue = "arbitrary_value"
		)

		var (
			networkID string
			testTags  map[string]string
		)

		BeforeEach(func() {
			if connectionErr != nil { // Only do these tests if an actual ACS instance is available via cloud-config.
				Skip("Could not connect to ACS instance.")
			}

			if err := client.GetOrCreateNetwork(cluster); err != nil {
				Skip("Could not find network.")
			}

			networkID = cluster.Status.NetworkID
			testTags = map[string]string{tagKey: tagValue}
		})

		It("Tags a network with an arbitrary tag.", func() {
			// Delete the tag if it already exists from a prior test run, otherwise the test will fail.
			_ = client.DeleteNetworkTags(networkID, testTags)
			Ω(client.AddNetworkTags(networkID, testTags)).Should(Succeed())
		})

		It("Fetches said tag.", func() {
			tags, err := client.GetNetworkTags(networkID)
			Ω(err).Should(BeNil())
			Ω(tags[tagKey]).Should(Equal(tagValue))
		})

		It("Deletes said tag.", func() {
			Ω(client.DeleteNetworkTags(networkID, testTags)).Should(Succeed())
			remainingTags, err := client.GetNetworkTags(networkID)
			Ω(err).Should(BeNil())
			Ω(remainingTags[tagKey]).Should(Equal(""))
		})
	})
})
