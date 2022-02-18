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
	"errors"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "github.com/aws/cluster-api-provider-cloudstack/pkg/cloud/test_dummies"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AffinityGroup Unit Tests", func() {
	var ( // Declare shared vars.
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient
		ags        *cloudstack.MockAffinityGroupServiceIface
		client     cloud.Client
	)

	BeforeEach(func() {
		// Setup new mock services.
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		ags = mockClient.AffinityGroup.(*cloudstack.MockAffinityGroupServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
		dummies.SetDummyClusterSpecVars()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("fetches an affinity group", func() {
		ags.EXPECT().GetAffinityGroupByName(dummies.AffinityGroup.Name).Return(&cloudstack.AffinityGroup{}, 1, nil)

		Ω(client.GetOrCreateAffinityGroup(dummies.Cluster, dummies.AffinityGroup)).Should(Succeed())
	})
	It("creates an affinity group", func() {
<<<<<<< HEAD
		fakeAG.ID = "FakeID"
		cluster.Spec.Account = "FakeAccount"
		cluster.Status.DomainID = "FakeDomainId"
		ags.EXPECT().GetAffinityGroupByID(fakeAG.ID).Return(nil, -1, errors.New("fakeError"))
		ags.EXPECT().NewCreateAffinityGroupParams(fakeAG.Name, fakeAG.Type).
=======
		dummies.AffinityGroup.Id = "FakeID"
		dummies.Cluster.Spec.Account = "FakeAccount"
		dummies.Cluster.Status.DomainID = "FakeDomainId"
		ags.EXPECT().GetAffinityGroupByID(dummies.AffinityGroup.Id).Return(nil, -1, errors.New("FakeError"))
		ags.EXPECT().NewCreateAffinityGroupParams(dummies.AffinityGroup.Name, dummies.AffinityGroup.Type).
>>>>>>> 36c0963 (Major moves toward multizone, but still broken.)
			Return(&cloudstack.CreateAffinityGroupParams{})
		ags.EXPECT().CreateAffinityGroup(ParamMatch(And(AccountEquals("FakeAccount"), DomainIDEquals("FakeDomainId")))).
			Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

		Ω(client.GetOrCreateAffinityGroup(dummies.Cluster, dummies.AffinityGroup)).Should(Succeed())
	})

	Context("AffinityGroup Integ Tests", func() {
		client, connectionErr := cloud.NewClient("../../cloud-config")

		var ( // Declare shared vars.
			arbitraryAG *cloud.AffinityGroup
		)
		BeforeEach(func() {
			if connectionErr != nil { // Only do these tests if an actual ACS instance is available via cloud-config.
				Skip("Could not connect to ACS instance.")
			}
			arbitraryAG = &cloud.AffinityGroup{Name: "ArbitraryAffinityGroup", Type: cloud.AffinityGroupType}
		})
		AfterEach(func() {
			mockCtrl.Finish()
		})

		It("Creates an affinity group.", func() {
			Ω(client.GetOrCreateAffinityGroup(dummies.Cluster, arbitraryAG)).Should(Succeed())
			arbitraryAG2 := &cloud.AffinityGroup{Name: arbitraryAG.Name}
			Ω(client.GetOrCreateAffinityGroup(dummies.Cluster, arbitraryAG2)).Should(Succeed())
			Ω(arbitraryAG2).Should(Equal(arbitraryAG))
		})
		It("Associates an affinity group.", func() {
			if err := client.GetOrCreateCluster(dummies.Cluster); err != nil {
				Skip("Could not flesh out Cluster." + err.Error())
			}
			if err := client.GetOrCreateVMInstance(dummies.Machine, dummies.CapiMachine, dummies.Cluster, ""); err != nil {
				Skip("Could not create VM." + err.Error())
			}
			Ω(client.GetOrCreateAffinityGroup(dummies.Cluster, arbitraryAG)).Should(Succeed())
			Ω(client.AssociateAffinityGroup(dummies.Machine, *arbitraryAG)).Should(Succeed())
		})
		It("Deletes an affinity group.", func() {
			Ω(client.DeleteAffinityGroup(arbitraryAG)).Should(Succeed())
			Ω(client.FetchAffinityGroup(arbitraryAG)).ShouldNot(Succeed())
		})
	})
})
