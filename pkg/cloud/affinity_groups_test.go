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
	"github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
)

var _ = ginkgo.Describe("AffinityGroup Unit Tests", func() {
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

	ginkgo.BeforeEach(func() {
		// Setup new mock services.
		mockCtrl = gomock.NewController(ginkgo.GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		ags = mockClient.AffinityGroup.(*cloudstack.MockAffinityGroupServiceIface)
		vms = mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient, nil)
		dummies.SetDummyVars()
	})

	ginkgo.AfterEach(func() {
		mockCtrl.Finish()
	})

	ginkgo.Context("Fetch or Create Affinity group", func() {
		ginkgo.It("fetches an affinity group by Name", func() {
			dummies.AffinityGroup.ID = "" // Force name fetching.
			ags.EXPECT().GetAffinityGroupByName(dummies.AffinityGroup.Name, gomock.Any()).Return(&cloudstack.AffinityGroup{}, 1, nil)

			gomega.Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(gomega.Succeed())
		})

		ginkgo.It("fetches an affinity group by ID", func() {
			ags.EXPECT().GetAffinityGroupByID(dummies.AffinityGroup.ID, gomock.Any()).Return(&cloudstack.AffinityGroup{}, 1, nil)

			gomega.Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(gomega.Succeed())
		})

		ginkgo.It("creates an affinity group", func() {
			// dummies.SetDummyDomainAndAccount()
			// dummies.SetDummyDomainID()
			ags.EXPECT().GetAffinityGroupByID(dummies.AffinityGroup.ID, gomock.Any()).Return(nil, -1, fakeError)
			ags.EXPECT().NewCreateAffinityGroupParams(dummies.AffinityGroup.Name, dummies.AffinityGroup.Type).
				Return(&cloudstack.CreateAffinityGroupParams{})
			ags.EXPECT().CreateAffinityGroup(ParamMatch(gomega.And(NameEquals(dummies.AffinityGroup.Name)))).
				Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

			gomega.Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(gomega.Succeed())
		})

		ginkgo.It("creates an affinity group if Name provided returns more than one affinity group", func() {
			dummies.AffinityGroup.ID = "" // Force name fetching.
			agp := &cloudstack.CreateAffinityGroupParams{}
			ags.EXPECT().GetAffinityGroupByName(dummies.AffinityGroup.Name, gomock.Any()).Return(&cloudstack.AffinityGroup{}, 2, nil)
			ags.EXPECT().NewCreateAffinityGroupParams(gomock.Any(), gomock.Any()).Return(agp)
			ags.EXPECT().CreateAffinityGroup(agp).Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

			gomega.Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(gomega.Succeed())
		})

		ginkgo.It("creates an affinity group if getting affinity group by name fails", func() {
			dummies.AffinityGroup.ID = "" // Force name fetching.
			agp := &cloudstack.CreateAffinityGroupParams{}
			ags.EXPECT().GetAffinityGroupByName(dummies.AffinityGroup.Name, gomock.Any()).Return(nil, -1, fakeError)
			ags.EXPECT().NewCreateAffinityGroupParams(gomock.Any(), gomock.Any()).Return(agp)
			ags.EXPECT().CreateAffinityGroup(agp).Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

			gomega.Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(gomega.Succeed())
		})

		ginkgo.It("creates an affinity group if ID provided returns more than one affinity group", func() {
			agp := &cloudstack.CreateAffinityGroupParams{}
			ags.EXPECT().GetAffinityGroupByID(dummies.AffinityGroup.ID, gomock.Any()).Return(&cloudstack.AffinityGroup{}, 2, nil)
			ags.EXPECT().NewCreateAffinityGroupParams(gomock.Any(), gomock.Any()).Return(agp)
			ags.EXPECT().CreateAffinityGroup(agp).Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

			gomega.Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(gomega.Succeed())
		})

		ginkgo.It("creates an affinity group if getting affinity group by ID fails", func() {
			agp := &cloudstack.CreateAffinityGroupParams{}
			ags.EXPECT().GetAffinityGroupByID(dummies.AffinityGroup.ID, gomock.Any()).Return(nil, -1, fakeError)
			ags.EXPECT().NewCreateAffinityGroupParams(gomock.Any(), gomock.Any()).Return(agp)
			ags.EXPECT().CreateAffinityGroup(agp).Return(&cloudstack.CreateAffinityGroupResponse{}, nil)

			gomega.Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(gomega.Succeed())
		})
	})

	ginkgo.Context("Delete Affinity group in CloudStack", func() {
		ginkgo.It("delete affinity group", func() {
			agp := &cloudstack.DeleteAffinityGroupParams{}
			ags.EXPECT().NewDeleteAffinityGroupParams().Return(agp)
			ags.EXPECT().DeleteAffinityGroup(agp).Return(&cloudstack.DeleteAffinityGroupResponse{}, nil)

			gomega.Ω(client.DeleteAffinityGroup(dummies.AffinityGroup)).Should(gomega.Succeed())
		})
	})

	ginkgo.Context("AffinityGroup Integ Tests", ginkgo.Label("integ"), func() {

		ginkgo.BeforeEach(func() {
			client = realCloudClient
			dummies.AffinityGroup.ID = "" // Force name fetching.
		})

		ginkgo.It("Associates an affinity group.", func() {
			gomega.Ω(client.ResolveZone(&dummies.CSFailureDomain1.Spec.Zone)).Should(gomega.Succeed())
			dummies.CSMachine1.Spec.DiskOffering.Name = ""

			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "",
			)).Should(gomega.Succeed())

			gomega.Ω(client.GetOrCreateAffinityGroup(dummies.AffinityGroup)).Should(gomega.Succeed())
			gomega.Ω(client.AssociateAffinityGroup(dummies.CSMachine1, *dummies.AffinityGroup)).Should(gomega.Succeed())

			// Make the created VM go away quickly by force stopping it.
			p := realCSClient.VirtualMachine.NewStopVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID)
			p.SetForced(true)
			_, err := realCSClient.VirtualMachine.StopVirtualMachine(p)
			gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("Creates and deletes an affinity group.", func() {
			gomega.Ω(client.DeleteAffinityGroup(dummies.AffinityGroup)).Should(gomega.Succeed())
			gomega.Ω(client.FetchAffinityGroup(dummies.AffinityGroup)).ShouldNot(gomega.Succeed())
		})
	})

	ginkgo.It("Associate affinity group", func() {
		uagp := &cloudstack.UpdateVMAffinityGroupParams{}
		vmp := &cloudstack.StartVirtualMachineParams{}
		vms.EXPECT().GetVirtualMachineByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(&cloudstack.VirtualMachine{}, 1, nil)
		ags.EXPECT().NewUpdateVMAffinityGroupParams(*dummies.CSMachine1.Spec.InstanceID).Return(uagp)
		vms.EXPECT().NewStopVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).Return(&cloudstack.StopVirtualMachineParams{})
		vms.EXPECT().StopVirtualMachine(&cloudstack.StopVirtualMachineParams{}).Return(&cloudstack.StopVirtualMachineResponse{State: "Stopping"}, nil)
		ags.EXPECT().UpdateVMAffinityGroup(uagp).Return(&cloudstack.UpdateVMAffinityGroupResponse{}, nil)
		vms.EXPECT().NewStartVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).Return(vmp)
		vms.EXPECT().StartVirtualMachine(vmp).Return(&cloudstack.StartVirtualMachineResponse{}, nil)
		gomega.Ω(client.AssociateAffinityGroup(dummies.CSMachine1, *dummies.AffinityGroup)).Should(gomega.Succeed())
	})

	ginkgo.It("Disassociate affinity group", func() {
		uagp := &cloudstack.UpdateVMAffinityGroupParams{}
		vmp := &cloudstack.StartVirtualMachineParams{}
		vms.EXPECT().GetVirtualMachineByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(&cloudstack.VirtualMachine{}, 1, nil)
		ags.EXPECT().NewUpdateVMAffinityGroupParams(*dummies.CSMachine1.Spec.InstanceID).Return(uagp)
		vms.EXPECT().NewStopVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).Return(&cloudstack.StopVirtualMachineParams{})
		vms.EXPECT().StopVirtualMachine(&cloudstack.StopVirtualMachineParams{}).Return(&cloudstack.StopVirtualMachineResponse{State: "Stopping"}, nil)
		ags.EXPECT().UpdateVMAffinityGroup(uagp).Return(&cloudstack.UpdateVMAffinityGroupResponse{}, nil)
		vms.EXPECT().NewStartVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).Return(vmp)
		vms.EXPECT().StartVirtualMachine(vmp).Return(&cloudstack.StartVirtualMachineResponse{}, nil)
		gomega.Ω(client.DisassociateAffinityGroup(dummies.CSMachine1, *dummies.AffinityGroup)).Should(gomega.Succeed())
	})
})
