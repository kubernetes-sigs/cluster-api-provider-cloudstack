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

package cloud_test

import (
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
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
		client, connectionErr := cloud.NewClient("../../cloud-config")

		const (
			tagKey           = "test_tag"
			tagValue         = "arbitrary_value"
			clusterID        = "123456"
			createdByCAPCTag = "created_by_CAPC"
			clusterTag       = "CAPC_cluster_" + clusterID
		)

		var (
			networkID string
			testTags  map[string]string
			csCluster *infrav1.CloudStackCluster
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
			csCluster = &infrav1.CloudStackCluster{}
			csCluster.SetUID(clusterID)

			// Delete any existing tags
			existingTags, err := client.GetTags(cloud.ResourceTypeNetwork, networkID)
			if err != nil {
				Fail("Failed to get existing tags. Error: " + err.Error())
			}
			err = client.DeleteTags(cloud.ResourceTypeNetwork, networkID, existingTags)
			if err != nil {
				Fail("Failed to delete existing tags. Error: " + err.Error())
			}
		})

		It("Adds and gets a resource tag", func() {
			Ω(client.AddTags(cloud.ResourceTypeNetwork, networkID, testTags)).Should(Succeed())
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, networkID)
			Ω(err).Should(BeNil())
			Ω(tags[tagKey]).Should(Equal(tagValue))
		})

		It("deletes a resource tag", func() {
			_ = client.AddTags(cloud.ResourceTypeNetwork, networkID, testTags)
			Ω(client.DeleteTags(cloud.ResourceTypeNetwork, networkID, testTags)).Should(Succeed())
			remainingTags, err := client.GetTags(cloud.ResourceTypeNetwork, networkID)
			Ω(err).Should(BeNil())
			Ω(remainingTags[tagKey]).Should(Equal(""))
		})

		It("adds the tags for a cluster (resource created by CAPC)", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster, true)).Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, networkID)
			Ω(err).Should(BeNil())
			Ω(tags[createdByCAPCTag]).Should(Equal("1"))
			Ω(tags[clusterTag]).Should(Equal("1"))
		})

		It("does not fail when the cluster tags are added twice", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster, true)).Should(Succeed())
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster, true)).Should(Succeed())
		})

		It("adds the tags for a cluster (resource NOT created by CAPC)", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster, false)).Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, networkID)
			Ω(err).Should(BeNil())
			Ω(tags[createdByCAPCTag]).Should(Equal(""))
			Ω(tags[clusterTag]).Should(Equal("1"))
		})

		It("deletes a cluster tag", func() {
			_ = client.AddClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster, true)
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster)).Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, networkID)
			Ω(err).Should(BeNil())
			Ω(tags[createdByCAPCTag]).Should(Equal("1"))
			Ω(tags[clusterTag]).Should(Equal(""))
		})

		It("deletes a CAPC created tag", func() {
			_ = client.AddClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster, true)
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, networkID)).Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, networkID)
			Ω(err).Should(BeNil())
			Ω(tags[createdByCAPCTag]).Should(Equal(""))
			Ω(tags[clusterTag]).Should(Equal("1"))
		})

		It("does not fail when cluster and CAPC created tags are deleted twice", func() {
			_ = client.AddClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster, true)
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster)).Should(Succeed())
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster)).Should(Succeed())
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, networkID)).Should(Succeed())
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, networkID)).Should(Succeed())
		})

		It("does not allow a resource to be deleted when there are no tags", func() {
			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, networkID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeFalse())
		})

		It("does not allow a resource to be deleted when there is a cluster tag", func() {
			_ = client.AddClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster, true)
			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, networkID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeFalse())
		})

		It("does allow a resource to be deleted when there are no cluster tags and there is a CAPC created tag", func() {
			_ = client.AddClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster, true)
			_ = client.DeleteClusterTag(cloud.ResourceTypeNetwork, networkID, csCluster)
			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, networkID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeTrue())
		})
	})
})
