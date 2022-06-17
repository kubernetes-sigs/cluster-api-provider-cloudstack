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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies"
)

var _ = Describe("Tag Unit Tests", func() {
	BeforeEach(func() {
		dummies.SetDummyVars()
	})

	Context("Tag Integ Tests", Label("integ"), func() {
		BeforeEach(func() {
			client = realCloudClient
			FetchIntegTestResources()

			existingTags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			if err != nil {
				Fail("Failed to get existing tags. Error: " + err.Error())
			}
			if len(existingTags) > 0 {
				err = client.DeleteTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, existingTags)
				if err != nil {
					Fail("Failed to delete existing tags. Error: " + err.Error())
				}
			}
		})

		It("adds and gets a resource tag", func() {
			Ω(client.AddTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.Tags)).Should(Succeed())
			Ω(client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Equal(dummies.Tags))
		})

		It("deletes a resource tag", func() {
			Ω(client.AddTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.Tags)).Should(Succeed())
			Ω(client.DeleteTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.Tags)).Should(Succeed())
			Ω(client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Equal(map[string]string{}))
		})

		It("returns an error when you delete a tag that doesn't exist", func() {
			Ω(client.DeleteTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.Tags)).Should(Succeed())
		})

		It("adds the tags for a cluster (resource created by CAPC)", func() {
			Ω(client.AddCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).
				Should(Succeed())
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).
				Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(tags[dummies.CSClusterTagKey]).Should(Equal(dummies.CSClusterTagVal))
		})

		It("does not fail when the cluster tags are added twice", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
		})

		It("doesn't adds the tags for a cluster (resource NOT created by CAPC)", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			Ω(err).Should(BeNil())
			Ω(tags[dummies.CreatedByCapcKey]).Should(Equal(""))
			Ω(tags[dummies.CSClusterTagKey]).Should(Equal(""))
		})

		It("deletes a cluster tag", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())

			Ω(client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).ShouldNot(HaveKey(dummies.CSClusterTagKey))
		})

		It("adds and deletes a created by capc tag", func() {
			Ω(client.AddCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Succeed())
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Succeed())
		})

		It("does not fail when cluster and CAPC created tags are deleted twice", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Succeed())
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Succeed())
		})

		It("does not allow a resource to be deleted when there are no tags", func() {
			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeFalse())
		})

		It("does not allow a resource to be deleted when there is a cluster tag", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeFalse())
		})

		It("does allow a resource to be deleted when there are no cluster tags and there is a CAPC created tag", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.AddCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Succeed())
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())

			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeTrue())
		})
	})
})
