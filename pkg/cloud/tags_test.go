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
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tag Unit Tests", func() {
	BeforeEach(func() {
		dummies.SetDummyVars()
	})

	Context("Tag Integ Tests", func() {
		client, connectionErr := cloud.NewClient("../../cloud-config")

		BeforeEach(func() {
			if connectionErr != nil { // Only do these tests if an actual ACS instance is available via cloud-config.
				Skip("Could not connect to ACS instance.")
			}

			if err := client.ResolveNetwork(dummies.CSCluster, &dummies.Net1); err != nil {
				Skip("Could not find network.")
			}

			// Delete any existing tags
			existingTags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.Net1.ID)
			if err != nil {
				Fail("Failed to get existing tags. Error: " + err.Error())
			}
			err = client.DeleteTags(cloud.ResourceTypeNetwork, dummies.Net1.ID, existingTags)
			if err != nil {
				Fail("Failed to delete existing tags. Error: " + err.Error())
			}
		})

		It("adds and gets a resource tag", func() {
			Ω(client.AddTags(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.TestTags)).Should(Succeed())
			Ω(client.GetTags(cloud.ResourceTypeNetwork, dummies.Net1.ID)).Should(Equal(dummies.TestTags))
		})

		It("deletes a resource tag", func() {
			Ω(client.AddTags(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.TestTags)).Should(Succeed())
			Ω(client.DeleteTags(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.TestTags)).Should(Succeed())
			Ω(client.GetTags(cloud.ResourceTypeNetwork, dummies.Net1.ID)).Should(Equal(map[string]string{}))
		})

		It("returns an error when you delete a tag that doesn't exist", func() {
			Ω(client.DeleteTags(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.TestTags)).ShouldNot(Succeed())
		})

		It("adds the tags for a cluster (resource created by CAPC)", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster, true)).
				Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.Net1.ID)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(tags["create_by_CAPC"]).Should(Equal("1"))
			Ω(tags[dummies.CSClusterTagKey]).Should(Equal(dummies.CSClusterTagVal))
		})

		It("does not fail when the cluster tags are added twice", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster, true)).Should(Succeed())
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster, true)).Should(Succeed())
		})

		It("adds the tags for a cluster (resource NOT created by CAPC)", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster, false)).Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.Net1.ID)
			Ω(err).Should(BeNil())
			Ω(tags[dummies.CreatedByCapcKey]).Should(Equal(""))
			Ω(tags[dummies.CSClusterTagKey]).Should(Equal("1"))
		})

		It("deletes a cluster tag", func() {
			_ = client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster, true)
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster)).Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.Net1.ID)
			Ω(err).Should(BeNil())
			Ω(tags[dummies.CreatedByCapcKey]).Should(Equal("1"))
			Ω(tags[dummies.CSClusterTagKey]).Should(Equal(""))
		})

		It("deletes a CAPC created tag", func() {
			_ = client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster, true)
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.Net1.ID)).Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.Net1.ID)
			Ω(err).Should(BeNil())
			Ω(tags[dummies.CreatedByCapcKey]).Should(Equal(""))
			Ω(tags[dummies.CSClusterTagKey]).Should(Equal("1"))
		})

		It("does not fail when cluster and CAPC created tags are deleted twice", func() {
			_ = client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster, true)
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.Net1.ID)).Should(Succeed())
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.Net1.ID)).Should(Succeed())
		})

		It("does not allow a resource to be deleted when there are no tags", func() {
			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, dummies.Net1.ID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeFalse())
		})

		It("does not allow a resource to be deleted when there is a cluster tag", func() {
			_ = client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster, true)
			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, dummies.Net1.ID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeFalse())
		})

		It("does allow a resource to be deleted when there are no cluster tags and there is a CAPC created tag", func() {
			_ = client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster, true)
			_ = client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.Net1.ID, dummies.CSCluster)
			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, dummies.Net1.ID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeTrue())
		})
	})
})
