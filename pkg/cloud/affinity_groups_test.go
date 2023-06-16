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
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
)

var _ = Describe("AffinityGroup Unit Tests", func() {
	const (
		errorMessage = "Fake Error"
	)

	fakeError := errors.New(errorMessage)
	var ( // Declare shared vars.
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient
		ags        *cloudstack.MockAffinityGroupServiceIface
		vms        *cloudstack.MockVirtualMachineServiceIface
		client     cloud.Client
	)

	BeforeEach(func() {
		// Setup new mock services.
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		ags = mockClient.AffinityGroup.(*cloudstack.MockAffinityGroupServiceIface)
		vms = mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
		dummies.SetDummyVars()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("Fetch or Create Affinity group", func() {
		It("fetches an affinity group by Name", func() {
			dummies.AffinityGroup.ID = "" // Force name fetching.
			ags.EXPECT().GetAffinityGroupByName(dummies.AffinityGroup.Name).Return(&cloudstack.AffinityGroup{}, 1, nil)

			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
		})

		It("fetches an affinity group by ID", func() {
			ags.EXPECT().GetAffinityGroupByID(dummies.AffinityGroup.ID).Return(&cloudstack.AffinityGroup{}, 1, nil)

			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
		})

		It("creates an affinity group", func() {
			// dummies.SetDummyDomainAndAccount()
			// dummies.SetDummyDomainID()
			ags.EXPECT().GetAffinityGroupByID(dummies.AffinityGroup.ID).Return(nil, -1, fakeError)
			ags.EXPECT().NewCreateAffinityGroupParams(dummies.AffinityGroup.Name, dummies.AffinityGroup.Type).
				Return(&cloudstack.CreateAffinityGroupParams{})
			ags.EXPECT().CreateAffinityGroup(ParamMatch(And(NameEquals(dummies.AffinityGroup.Name)))).
				Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
		})

		It("creates an affinity group if Name provided returns more than one affinity group", func() {
			dummies.AffinityGroup.ID = "" // Force name fetching.
			agp := &cloudstack.CreateAffinityGroupParams{}
			ags.EXPECT().GetAffinityGroupByName(dummies.AffinityGroup.Name).Return(&cloudstack.AffinityGroup{}, 2, nil)
			ags.EXPECT().NewCreateAffinityGroupParams(gomock.Any(), gomock.Any()).Return(agp)
			ags.EXPECT().CreateAffinityGroup(agp).Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
		})

		It("creates an affinity group if getting affinity group by name fails", func() {
			dummies.AffinityGroup.ID = "" // Force name fetching.
			agp := &cloudstack.CreateAffinityGroupParams{}
			ags.EXPECT().GetAffinityGroupByName(dummies.AffinityGroup.Name).Return(nil, -1, fakeError)
			ags.EXPECT().NewCreateAffinityGroupParams(gomock.Any(), gomock.Any()).Return(agp)
			ags.EXPECT().CreateAffinityGroup(agp).Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
		})

		It("creates an affinity group if ID provided returns more than one affinity group", func() {
			agp := &cloudstack.CreateAffinityGroupParams{}
			ags.EXPECT().GetAffinityGroupByID(dummies.AffinityGroup.ID).Return(&cloudstack.AffinityGroup{}, 2, nil)
			ags.EXPECT().NewCreateAffinityGroupParams(gomock.Any(), gomock.Any()).Return(agp)
			ags.EXPECT().CreateAffinityGroup(agp).Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
		})

		It("creates an affinity group if getting affinity group by ID fails", func() {
			agp := &cloudstack.CreateAffinityGroupParams{}
			ags.EXPECT().GetAffinityGroupByID(dummies.AffinityGroup.ID).Return(nil, -1, fakeError)
			ags.EXPECT().NewCreateAffinityGroupParams(gomock.Any(), gomock.Any()).Return(agp)
			ags.EXPECT().CreateAffinityGroup(agp).Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
		})
	})

	Context("Delete Affinity group in CloudStack", func() {
		It("delete affinity group", func() {
			agp := &cloudstack.DeleteAffinityGroupParams{}
			ags.EXPECT().NewDeleteAffinityGroupParams().Return(agp)
			ags.EXPECT().DeleteAffinityGroup(agp).Return(&cloudstack.DeleteAffinityGroupResponse{}, nil)

			Ω(client.DeleteAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
		})
	})

	Context("AffinityGroup Integ Tests", Label("integ"), func() {

		BeforeEach(func() {
			client = realCloudClient
			dummies.AffinityGroup.ID = "" // Force name fetching.
		})

		It("Associates an affinity group.", func() {
			Ω(client.ResolveZone(&dummies.CSFailureDomain1.Spec.Zone)).Should(Succeed())
			dummies.CSMachine1.Spec.DiskOffering.Name = ""

			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "",
			)).Should(Succeed())

			Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
			Ω(client.AssociateAffinityGroup(dummies.CSMachine1, *dummies.AffinityGroup)).Should(Succeed())

			// Make the created VM go away quickly by force stopping it.
			p := realCSClient.VirtualMachine.NewStopVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID)
			p.SetForced(true)
			_, err := realCSClient.VirtualMachine.StopVirtualMachine(p)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("Creates and deletes an affinity group.", func() {
			Ω(client.DeleteAffinityGroup(dummies.AffinityGroup)).Should(Succeed())
			Ω(client.FetchAffinityGroup(dummies.AffinityGroup)).ShouldNot(Succeed())
		})
	})

	It("Associate affinity group", func() {
		uagp := &cloudstack.UpdateVMAffinityGroupParams{}
		vmp := &cloudstack.StartVirtualMachineParams{}
		vms.EXPECT().GetVirtualMachineByID(*dummies.CSMachine1.Spec.InstanceID).Return(&cloudstack.VirtualMachine{}, 1, nil)
		ags.EXPECT().NewUpdateVMAffinityGroupParams(*dummies.CSMachine1.Spec.InstanceID).Return(uagp)
		vms.EXPECT().NewStopVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).Return(&cloudstack.StopVirtualMachineParams{})
		vms.EXPECT().StopVirtualMachine(&cloudstack.StopVirtualMachineParams{}).Return(&cloudstack.StopVirtualMachineResponse{State: "Stopping"}, nil)
		ags.EXPECT().UpdateVMAffinityGroup(uagp).Return(&cloudstack.UpdateVMAffinityGroupResponse{}, nil)
		vms.EXPECT().NewStartVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).Return(vmp)
		vms.EXPECT().StartVirtualMachine(vmp).Return(&cloudstack.StartVirtualMachineResponse{}, nil)
		Ω(client.AssociateAffinityGroup(dummies.CSMachine1, *dummies.AffinityGroup)).Should(Succeed())
	})

	It("Disassociate affinity group", func() {
		uagp := &cloudstack.UpdateVMAffinityGroupParams{}
		vmp := &cloudstack.StartVirtualMachineParams{}
		vms.EXPECT().GetVirtualMachineByID(*dummies.CSMachine1.Spec.InstanceID).Return(&cloudstack.VirtualMachine{}, 1, nil)
		ags.EXPECT().NewUpdateVMAffinityGroupParams(*dummies.CSMachine1.Spec.InstanceID).Return(uagp)
		vms.EXPECT().NewStopVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).Return(&cloudstack.StopVirtualMachineParams{})
		vms.EXPECT().StopVirtualMachine(&cloudstack.StopVirtualMachineParams{}).Return(&cloudstack.StopVirtualMachineResponse{State: "Stopping"}, nil)
		ags.EXPECT().UpdateVMAffinityGroup(uagp).Return(&cloudstack.UpdateVMAffinityGroupResponse{}, nil)
		vms.EXPECT().NewStartVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).Return(vmp)
		vms.EXPECT().StartVirtualMachine(vmp).Return(&cloudstack.StartVirtualMachineResponse{}, nil)
		Ω(client.DisassociateAffinityGroup(dummies.CSMachine1, *dummies.AffinityGroup)).Should(Succeed())
	})
})
