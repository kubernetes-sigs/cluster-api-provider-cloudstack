/*
Copyright 2022.

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
	"errors"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1alpha3"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AffinityGroup Unit Tests", func() {
	var ( // Declare shared vars.
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient
		ags        *cloudstack.MockAffinityGroupServiceIface
		fakeAG     cloud.AffinityGroup
		cluster    *infrav1.CloudStackCluster
		client     cloud.Client
	)

	BeforeEach(func() {
		// Setup new mock services.
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		ags = mockClient.AffinityGroup.(*cloudstack.MockAffinityGroupServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
		fakeAG = cloud.AffinityGroup{
			Name: "FakeAffinityGroup",
			Type: cloud.AffinityGroupType}
		cluster = &infrav1.CloudStackCluster{Spec: infrav1.CloudStackClusterSpec{}}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("fetches an affinity group", func() {
		ags.EXPECT().GetAffinityGroupByName(fakeAG.Name).Return(&cloudstack.AffinityGroup{}, 1, nil)

		Ω(client.GetOrCreateAffinityGroup(cluster, fakeAG)).Should(Succeed())
	})
	It("creates an affinity group", func() {
		fakeAG.Id = "FakeID"
		cluster.Spec.Account = "FakeAccount"
		cluster.Status.DomainID = "FakeDomainId"
		ags.EXPECT().GetAffinityGroupByID(fakeAG.Id).Return(nil, -1, errors.New("FakeError"))
		ags.EXPECT().NewCreateAffinityGroupParams(fakeAG.Name, fakeAG.Type).
			Return(&cloudstack.CreateAffinityGroupParams{})
		ags.EXPECT().CreateAffinityGroup(ParamMatch(And(AccountEquals("FakeAccount"), DomainIdEquals("FakeDomainId")))).
			Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

		Ω(client.GetOrCreateAffinityGroup(cluster, fakeAG)).Should(Succeed())
	})

	Context("AffinityGroup Integ Tests", func() {
		client, err := cloud.NewClient("../../cloud-config")
		var ( // Declare shared vars.
			arbitraryAG cloud.AffinityGroup
		)
		BeforeEach(func() {
			if err != nil { // Only do these tests if an actual ACS instance is available via cloud-config.
				Skip("Could not connect to ACS instance.")
			}
			arbitraryAG = cloud.AffinityGroup{Name: "ArbitraryAffinityGroup", Type: cloud.AffinityGroupType}
		})

		It("Creates an affinity group.", func() {
			Ω(client.GetOrCreateAffinityGroup(cluster, arbitraryAG)).Should(Succeed())
		})
		It("Deletes an affinity group.", func() {
			Ω(client.DeleteAffinityGroup(cluster, arbitraryAG)).Should(Succeed())
		})
	})
})
