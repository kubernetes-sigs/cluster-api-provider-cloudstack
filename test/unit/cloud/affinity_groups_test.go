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
	"fmt"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/aws/cluster-api-provider-cloudstack-staging/test/unit/dummies"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("The AffinityGroup interface", func() {
	var client cloud.Client
	BeforeEach(func() {
		dummies.SetDummyVars()
	})
	When("using a mock CloudStack Client", func() {
		var (
			mockCtrl   *gomock.Controller
			mockClient *cloudstack.CloudStackClient
			ags        *cloudstack.MockAffinityGroupServiceIface
		)
		BeforeEach(func() {
			// Setup mock CloudstackClient.
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = cloudstack.NewMockClient(mockCtrl)
			ags = mockClient.AffinityGroup.(*cloudstack.MockAffinityGroupServiceIface)
			client = cloud.NewClientFromCSAPIClient(mockClient)
		})
		AfterEach(func() {
			// Check mocked calls match.
			mockCtrl.Finish()
		})

		It("fetches an affinity group", func() {
			dummies.AffinityGroup.ID = "" // Force name fetching.
			ags.EXPECT().GetAffinityGroupByName(dummies.AffinityGroup.Name).Return(&cloudstack.AffinityGroup{}, 1, nil)

			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
		})
		It("creates an affinity group", func() {
			dummies.SetDummyDomainAndAccount()
			dummies.SetDummyDomainID()
			ags.EXPECT().GetAffinityGroupByID(dummies.AffinityGroup.ID).Return(nil, -1, errors.New("FakeError"))
			ags.EXPECT().NewCreateAffinityGroupParams(dummies.AffinityGroup.Name, dummies.AffinityGroup.Type).
				Return(&cloudstack.CreateAffinityGroupParams{})
			ags.EXPECT().CreateAffinityGroup(ParamMatch(And(NameEquals(dummies.AffinityGroup.Name)))).
				Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
		})
	})

	When("using a real CloudStack client", func() {
		BeforeEach(func() {
			fmt.Println("blah2oomuch")
			client = realCloudClient
			dummies.AffinityGroup.ID = "" // Force name fetching.
		})
		It("creates an affinity group.", func() {
			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
		})
		Context("with an existing VM", func() {
			BeforeEach(func() {
				Ω(client.ResolveZone(dummies.CSZone1)).Should(Succeed())
				Ω(dummies.CSZone1.Spec.ID).ShouldNot(BeEmpty())
				dummies.CSMachine1.Status.ZoneID = dummies.CSZone1.Spec.ID
				dummies.CSMachine1.Spec.DiskOffering.Name = ""
				dummies.CSCluster.Spec.ControlPlaneEndpoint.Host = ""
				Ω(client.GetOrCreateIsolatedNetwork(dummies.CSZone1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
				dummies.CSZone1.Spec.Network.ID = dummies.CSISONet1.Spec.ID
				Ω(client.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "",
				)).Should(Succeed())
			})
			AfterEach(func() {
				Ω(client.DestroyVMInstance(dummies.CSMachine1)).Should(MatchError("VM deletion in progress"))
			})
			It("associates an affinity group and disassociates it", func() {
				Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
				Ω(client.AssociateAffinityGroup(dummies.CSMachine1, *dummies.AffinityGroup)).Should(Succeed())
				Ω(client.DisassociateAffinityGroup(dummies.CSMachine1, *dummies.AffinityGroup)).Should(Succeed())
			})
		})
		It("deletes an affinity group.", func() {
			Ω(client.DeleteAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
			Ω(client.FetchAffinityGroup(dummies.AffinityGroup)).ShouldNot(Succeed())
		})
		Context("del", func() {})
	})
})
