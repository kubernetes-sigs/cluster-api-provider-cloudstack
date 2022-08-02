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
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
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
		dummies.SetDummyVars()
	})

	AfterEach(func() {
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

	Context("AffinityGroup Integ Tests", Label("integ"), func() {

		BeforeEach(func() {
			client = realCloudClient
			dummies.AffinityGroup.ID = "" // Force name fetching.
		})

		It("Associates an affinity group.", func() {
			Ω(client.ResolveZone(&dummies.CSZone1.Spec)).Should(Succeed())
			dummies.CSMachine1.Status.ZoneID = dummies.CSZone1.Spec.ID
			dummies.CSMachine1.Spec.DiskOffering.Name = ""

			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "",
			)).Should(Succeed())

			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
			Ω(client.AssociateAffinityGroup(dummies.CSMachine1, *dummies.AffinityGroup)).Should(Succeed())

			// Make the created VM go away quickly by force stopping it.
			p := realCSClient.VirtualMachine.NewStopVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID)
			p.SetForced(true)
			Ω(realCSClient.VirtualMachine.StopVirtualMachine(p)).Should(Succeed())
		})

		It("Creates and deletes an affinity group.", func() {
			Ω(client.DeleteAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
			Ω(client.FetchAffinityGroup(dummies.AffinityGroup)).ShouldNot(Succeed())
		})
	})
})
