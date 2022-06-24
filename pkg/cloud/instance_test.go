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
	"fmt"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"
)

var _ = Describe("Instance", func() {
	const (
		unknownErrorMessage  = "unknown err"
		offeringFakeID       = "123"
		templateFakeID       = "456"
		isoFakeID            = "901"
		executableFilter     = "executable"
		diskOfferingFakeID   = "789"
		virtualMachineFakeID = "abc"
	)

	notFoundError := errors.New("no match found")
	unknownError := errors.New(unknownErrorMessage)

	var (
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient
		vms        *cloudstack.MockVirtualMachineServiceIface
		sos        *cloudstack.MockServiceOfferingServiceIface
		dos        *cloudstack.MockDiskOfferingServiceIface
		iso        *cloudstack.MockISOServiceIface
		ts         *cloudstack.MockTemplateServiceIface
		vs         *cloudstack.MockVolumeServiceIface
		client     cloud.Client
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		vms = mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
		sos = mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
		dos = mockClient.DiskOffering.(*cloudstack.MockDiskOfferingServiceIface)
		iso = mockClient.ISO.(*cloudstack.MockISOServiceIface)
		ts = mockClient.Template.(*cloudstack.MockTemplateServiceIface)
		vs = mockClient.Volume.(*cloudstack.MockVolumeServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)

		dummies.SetDummyVars()
		dummies.SetDummyClusterStatus()
		dummies.SetDummyCSMachineStatuses()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("when fetching a VM instance", func() {
		It("Handles an unknown error when fetching by ID", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(nil, -1, unknownError)
			Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).To(MatchError(unknownErrorMessage))
		})

		It("Handles finding more than one VM instance by ID", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(nil, 2, nil)
			Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).
				Should(MatchError("found more than one VM Instance with ID " + *dummies.CSMachine1.Spec.InstanceID))
		})

		It("sets dummies.CSMachine1 spec and status values when VM instance found by ID", func() {
			vmsResp := &cloudstack.VirtualMachinesMetric{Id: *dummies.CSMachine1.Spec.InstanceID}
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(vmsResp, 1, nil)
			Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).Should(Succeed())
			Ω(dummies.CSMachine1.Spec.ProviderID).Should(Equal(pointer.StringPtr("cloudstack:///" + vmsResp.Id)))
			Ω(dummies.CSMachine1.Spec.InstanceID).Should(Equal(pointer.StringPtr(vmsResp.Id)))
		})

		It("handles an unknown error when fetching by name", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, -1, unknownError)

			Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).Should(MatchError(unknownErrorMessage))
		})

		It("handles finding more than one VM instance by Name", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, 2, nil)

			Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).Should(
				MatchError("found more than one VM Instance with name " + dummies.CSMachine1.Name))
		})

		It("sets dummies.CSMachine1 spec and status values when VM instance found by Name", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).
				Return(&cloudstack.VirtualMachinesMetric{Id: *dummies.CSMachine1.Spec.InstanceID}, -1, nil)

			Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).Should(Succeed())
			Ω(dummies.CSMachine1.Spec.ProviderID).Should(Equal(
				pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", *dummies.CSMachine1.Spec.InstanceID))))
			Ω(dummies.CSMachine1.Spec.InstanceID).Should(Equal(pointer.StringPtr(*dummies.CSMachine1.Spec.InstanceID)))
		})
	})

	Context("when creating a VM instance", func() {
		vmMetricResp := &cloudstack.VirtualMachinesMetric{}

		expectVMNotFound := func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, -1, notFoundError)
		}

		It("doesn't re-create if one already exists.", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(vmMetricResp, -1, nil)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				Should(Succeed())
		})

		It("returns unknown error while fetching VM instance", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(nil, -1, unknownError)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				Should(MatchError(unknownErrorMessage))
		})

		It("returns errors occurring while fetching service offering information", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).Return("", -1, unknownError)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				ShouldNot(Succeed())
		})

		It("returns errors if more than one service offering found", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).Return("", 2, nil)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				ShouldNot(Succeed())
		})

		It("returns errors while fetching template", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).
				Return(dummies.CSMachine1.Spec.Offering.ID, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID).
				Return("", -1, unknownError)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				ShouldNot(Succeed())
		})

		It("returns errors when more than one template found", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).
				Return(dummies.CSMachine1.Spec.Offering.ID, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID).Return("", 2, nil)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				ShouldNot(Succeed())
		})

		It("returns errors when more than one diskoffering found", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).
				Return(dummies.CSMachine1.Spec.Offering.ID, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID).Return(dummies.CSMachine1.Spec.Template.ID, 1, nil)
			dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name).Return(diskOfferingFakeID, 2, nil)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				ShouldNot(Succeed())
		})

		It("returns errors when fetching diskoffering", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).
				Return(dummies.CSMachine1.Spec.Offering.ID, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID).Return(dummies.CSMachine1.Spec.Template.ID, 1, nil)
			dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name).Return(diskOfferingFakeID, 1, nil)
			dos.EXPECT().GetDiskOfferingByID(diskOfferingFakeID).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, unknownError)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				ShouldNot(Succeed())
		})

		It("returns errors when disk size not zero for non-customized disk offering", func() {
			expectVMNotFound()
			dummies.CSMachine1.Spec.DiskOffering.CustomSize = 1
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).
				Return(dummies.CSMachine1.Spec.Offering.ID, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID).Return(dummies.CSMachine1.Spec.Template.ID, 1, nil)
			dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name).Return(diskOfferingFakeID, 1, nil)
			dos.EXPECT().GetDiskOfferingByID(diskOfferingFakeID).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				ShouldNot(Succeed())
		})

		It("returns errors when disk size zero for customized disk offering", func() {
			expectVMNotFound()
			dummies.CSMachine1.Spec.DiskOffering.CustomSize = 0
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).
				Return(dummies.CSMachine1.Spec.Offering.ID, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID).Return(dummies.CSMachine1.Spec.Template.ID, 1, nil)
			dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name).Return(diskOfferingFakeID, 1, nil)
			dos.EXPECT().GetDiskOfferingByID(diskOfferingFakeID).Return(&cloudstack.DiskOffering{Iscustomized: true}, 1, nil)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				ShouldNot(Succeed())
		})

		It("handles deployment errors", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).
				Return(offeringFakeID, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID).
				Return(templateFakeID, 1, nil)
			dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name).
				Return(diskOfferingFakeID, 1, nil)
			dos.EXPECT().GetDiskOfferingByID(diskOfferingFakeID).
				Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)
			vms.EXPECT().NewDeployVirtualMachineParams(offeringFakeID, templateFakeID, dummies.Zone1.ID).
				Return(&cloudstack.DeployVirtualMachineParams{})
			vms.EXPECT().DeployVirtualMachine(gomock.Any()).Return(nil, unknownError)
			vms.EXPECT().NewListVirtualMachinesParams().Return(&cloudstack.ListVirtualMachinesParams{})
			vms.EXPECT().ListVirtualMachines(gomock.Any()).Return(&cloudstack.ListVirtualMachinesResponse{}, nil)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				Should(MatchError(unknownErrorMessage))
		})

		Context("when using UUIDs and/or names to locate service offerings and templates", func() {
			BeforeEach(func() {
				gomock.InOrder(
					vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).
						Return(nil, -1, notFoundError),
					vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).
						Return(&cloudstack.VirtualMachinesMetric{}, 1, nil))

				vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, -1, notFoundError)
			})

			ActionAndAssert := func() {
				vms.EXPECT().NewDeployVirtualMachineParams(offeringFakeID, templateFakeID, dummies.Zone1.ID).
					Return(&cloudstack.DeployVirtualMachineParams{})

				deploymentResp := &cloudstack.DeployVirtualMachineResponse{Id: *dummies.CSMachine1.Spec.InstanceID}
				vms.EXPECT().DeployVirtualMachine(gomock.Any()).Return(deploymentResp, nil)

				Ω(client.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
					Should(Succeed())
			}

			It("works with service offering name and template name", func() {
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.ID = ""
				dummies.CSMachine1.Spec.Template.ID = ""
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).Return(offeringFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name).Return(diskOfferingFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingByID(dummies.CSMachine1.Spec.DiskOffering.ID).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)
				ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID).
					Return(templateFakeID, 1, nil)

				ActionAndAssert()
			})

			It("works with service offering name and template name without disk offering", func() {
				dummies.CSMachine1.Spec.Offering.ID = ""
				dummies.CSMachine1.Spec.Template.ID = ""
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"
				dummies.CSMachine1.Spec.DiskOffering = v1beta1.CloudStackResourceDiskOffering{}

				sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).Return(offeringFakeID, 1, nil)
				ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID).
					Return(templateFakeID, 1, nil)

				ActionAndAssert()
			})

			It("works with service offering ID and template name", func() {
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = ""
				dummies.CSMachine1.Spec.Offering.Name = ""
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID).Return(&cloudstack.ServiceOffering{Name: ""}, 1, nil)
				ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID).
					Return(templateFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name).Return(diskOfferingFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingByID(dummies.CSMachine1.Spec.DiskOffering.ID).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)

				ActionAndAssert()
			})

			It("works with service offering name and template ID", func() {
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.ID = ""
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = ""

				sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).Return(offeringFakeID, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, executableFilter).Return(&cloudstack.Template{Name: ""}, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name).Return(diskOfferingFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingByID(dummies.CSMachine1.Spec.DiskOffering.ID).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)

				ActionAndAssert()
			})

			It("works with service offering ID and template ID", func() {
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = ""
				dummies.CSMachine1.Spec.Template.Name = ""

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID).
					Return(&cloudstack.ServiceOffering{Name: "offering"}, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name).
					Return(diskOfferingFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingByID(dummies.CSMachine1.Spec.DiskOffering.ID).
					Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, executableFilter).
					Return(&cloudstack.Template{Name: "template"}, 1, nil)

				ActionAndAssert()
			})

			It("works with Id and name both provided", func() {
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID).Return(&cloudstack.ServiceOffering{Name: "offering"}, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, executableFilter).Return(&cloudstack.Template{Name: "template"}, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name).Return(diskOfferingFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingByID(dummies.CSMachine1.Spec.DiskOffering.ID).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)

				ActionAndAssert()
			})
		})

		Context("when using both UUIDs and names to locate service offerings and templates", func() {
			BeforeEach(func() {
				vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).
					Return(nil, -1, notFoundError)
				vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, -1, notFoundError)
			})

			It("works with Id and name both provided, offering name mismatch", func() {
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID).Return(&cloudstack.ServiceOffering{Name: "offering-not-match"}, 1, nil)
				requiredRegexp := "offering name %s does not match name %s returned using UUID %s"
				Ω(client.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
					Should(MatchError(MatchRegexp(requiredRegexp, dummies.CSMachine1.Spec.Offering.Name, "offering-not-match", offeringFakeID)))
			})

			It("works with Id and name both provided, template name mismatch", func() {
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID).Return(&cloudstack.ServiceOffering{Name: "offering"}, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, executableFilter).Return(&cloudstack.Template{Name: "template-not-match"}, 1, nil)
				requiredRegexp := "template name %s does not match name %s returned using UUID %s"
				Ω(client.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
					Should(MatchError(MatchRegexp(requiredRegexp, dummies.CSMachine1.Spec.Template.Name, "template-not-match", templateFakeID)))

			})

			It("works with Id and name both provided, disk offering id/name mismatch", func() {
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"
				dummies.CSMachine1.Spec.DiskOffering.Name = "diskoffering"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID).Return(&cloudstack.ServiceOffering{Name: "offering"}, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, executableFilter).Return(&cloudstack.Template{Name: "template"}, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name).Return(diskOfferingFakeID+"-not-match", 1, nil)
				requiredRegexp := "diskOffering ID %s does not match ID %s returned using name %s"
				Ω(client.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
					Should(MatchError(MatchRegexp(requiredRegexp, dummies.CSMachine1.Spec.DiskOffering.ID, diskOfferingFakeID+"-not-match", dummies.CSMachine1.Spec.DiskOffering.Name)))

			})
		})
	})
	Context("when attaching ISO to VM instance", func() {
		It("returns errors occurring while fetching iso offering information", func() {
			iso.EXPECT().GetIsoID(dummies.CSMachine1.Spec.ISOAttachment.Name, "all", dummies.Zone1.ID).Return("", -1, unknownError)
			Ω(client.AttachISOToVMInstance(
				dummies.CSMachine1, dummies.CSZone1)).
				ShouldNot(Succeed())
		})

		It("returns errors if more than one iso offering found", func() {
			iso.EXPECT().GetIsoID(dummies.CSMachine1.Spec.ISOAttachment.Name, "all", dummies.Zone1.ID).Return("", 2, nil)
			Ω(client.AttachISOToVMInstance(
				dummies.CSMachine1, dummies.CSZone1)).
				ShouldNot(Succeed())
		})

		It("returns errors if VM instance ID and name both empty", func() {
			dummies.CSMachine1.Spec.InstanceID = nil
			dummies.CSMachine1.Name = ""
			Ω(client.AttachISOToVMInstance(
				dummies.CSMachine1, dummies.CSZone1)).
				ShouldNot(Succeed())
		})

		It("returns OK", func() {
			iso.EXPECT().GetIsoID(dummies.CSMachine1.Spec.ISOAttachment.Name, "all", dummies.Zone1.ID).Return(isoFakeID, 1, nil)
			iso.EXPECT().NewAttachIsoParams(isoFakeID, *dummies.CSMachine1.Spec.InstanceID).Return(&cloudstack.AttachIsoParams{})
			iso.EXPECT().AttachIso(&cloudstack.AttachIsoParams{}).Return(nil, nil)
			Ω(client.AttachISOToVMInstance(
				dummies.CSMachine1, dummies.CSZone1)).
				Should(Succeed())
		})
	})
	Context("when attaching ISO to VM instance using iso ID", func() {
		It("returns errors occurring while fetching iso offering information", func() {
			dummies.CSMachine1.Spec.ISOAttachment.ID = isoFakeID
			iso.EXPECT().GetIsoByID(dummies.CSMachine1.Spec.ISOAttachment.ID).Return(nil, -1, unknownError)
			Ω(client.AttachISOToVMInstance(
				dummies.CSMachine1, dummies.CSZone1)).
				ShouldNot(Succeed())
		})

		It("returns errors if more than one iso offering found", func() {
			dummies.CSMachine1.Spec.ISOAttachment.ID = isoFakeID
			iso.EXPECT().GetIsoByID(dummies.CSMachine1.Spec.ISOAttachment.ID).Return(nil, 2, nil)
			Ω(client.AttachISOToVMInstance(
				dummies.CSMachine1, dummies.CSZone1)).
				ShouldNot(Succeed())
		})

		It("returns errors if name not match", func() {
			dummies.CSMachine1.Spec.ISOAttachment.ID = isoFakeID
			iso.EXPECT().GetIsoByID(dummies.CSMachine1.Spec.ISOAttachment.ID).Return(&cloudstack.Iso{
				Name: "cloudstack-script.iso-not-match",
			}, 1, nil)
			Ω(client.AttachISOToVMInstance(
				dummies.CSMachine1, dummies.CSZone1)).
				ShouldNot(Succeed())
		})

		It("returns errors if VM name not found", func() {
			dummies.CSMachine1.Spec.InstanceID = nil
			dummies.CSMachine1.Name = "fake"
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, -1, unknownError)
			Ω(client.AttachISOToVMInstance(
				dummies.CSMachine1, dummies.CSZone1)).
				ShouldNot(Succeed())
		})

		It("returns errors if more than one VM found", func() {
			dummies.CSMachine1.Spec.InstanceID = nil
			dummies.CSMachine1.Name = "fake"
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, 2, nil)
			Ω(client.AttachISOToVMInstance(
				dummies.CSMachine1, dummies.CSZone1)).
				ShouldNot(Succeed())
		})

		It("returns OK", func() {
			dummies.CSMachine1.Spec.InstanceID = nil
			dummies.CSMachine1.Name = "fake"
			dummies.CSMachine1.Spec.ISOAttachment.ID = isoFakeID
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(&cloudstack.VirtualMachinesMetric{
				Id: virtualMachineFakeID,
			}, 1, nil)
			iso.EXPECT().GetIsoByID(dummies.CSMachine1.Spec.ISOAttachment.ID).Return(&cloudstack.Iso{
				Name: "cloudstack-script.iso",
			}, 1, nil)
			iso.EXPECT().NewAttachIsoParams(isoFakeID, virtualMachineFakeID).Return(&cloudstack.AttachIsoParams{})
			iso.EXPECT().AttachIso(&cloudstack.AttachIsoParams{}).Return(nil, nil)
			Ω(client.AttachISOToVMInstance(
				dummies.CSMachine1, dummies.CSZone1)).
				Should(Succeed())
		})
	})
	Context("when starting VM instance", func() {
		It("returns OK if VM instance ID not empty", func() {
			p := &cloudstack.StartVirtualMachineParams{}
			vms.EXPECT().NewStartVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).Return(p)
			vms.EXPECT().StartVirtualMachine(p).Return(&cloudstack.StartVirtualMachineResponse{
				Id:    virtualMachineFakeID,
				State: "Running",
				Nic: []cloudstack.Nic{
					{Ipaddress: "172.0.0.1"},
				},
			}, nil)
			Ω(client.StartVMInstance(
				dummies.CSMachine1)).
				Should(Succeed())
		})

		It("returns OK if VM instance ID empty but Name not empty", func() {
			dummies.CSMachine1.Spec.InstanceID = nil
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(&cloudstack.VirtualMachinesMetric{
				Id: virtualMachineFakeID,
			}, 1, nil)
			p := &cloudstack.StartVirtualMachineParams{}
			vms.EXPECT().NewStartVirtualMachineParams(virtualMachineFakeID).Return(p)
			vms.EXPECT().StartVirtualMachine(p).Return(&cloudstack.StartVirtualMachineResponse{
				Id:    virtualMachineFakeID,
				State: "Running",
				Nic: []cloudstack.Nic{
					{Ipaddress: "172.0.0.1"},
				},
			}, nil)
			Ω(client.StartVMInstance(
				dummies.CSMachine1)).
				Should(Succeed())
		})

		It("returns error if VM instance ID and Name both empty", func() {
			dummies.CSMachine1.Spec.InstanceID = nil
			dummies.CSMachine1.Name = ""
			Ω(client.StartVMInstance(
				dummies.CSMachine1)).
				ShouldNot(Succeed())
		})

		It("returns error if VM instance Name not found", func() {
			dummies.CSMachine1.Spec.InstanceID = nil
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, -1, unknownError)
			Ω(client.StartVMInstance(
				dummies.CSMachine1)).
				ShouldNot(Succeed())
		})

		It("returns error if more than one VM instance found using Name", func() {
			dummies.CSMachine1.Spec.InstanceID = nil
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, 2, nil)
			Ω(client.StartVMInstance(
				dummies.CSMachine1)).
				ShouldNot(Succeed())
		})
		It("returns error if start VM instance error", func() {
			p := &cloudstack.StartVirtualMachineParams{}
			vms.EXPECT().NewStartVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).Return(p)
			vms.EXPECT().StartVirtualMachine(p).Return(nil, unknownError)
			Ω(client.StartVMInstance(
				dummies.CSMachine1)).
				ShouldNot(Succeed())
		})

	})
	Context("when destroying a VM instance", func() {
		expungeDestroyParams := &cloudstack.DestroyVirtualMachineParams{}
		expungeDestroyParams.SetExpunge(true)
		listVolumesParams := &cloudstack.ListVolumesParams{}
		listVolumesResponse := &cloudstack.ListVolumesResponse{
			Volumes: []*cloudstack.Volume{
				{
					Id: "123",
				},
				{
					Id: "456",
				},
			},
		}

		It("calls destroy and finds VM doesn't exist, then returns nil", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, fmt.Errorf("unable to find uuid for id"))
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(Succeed())
		})

		It("calls destroy and returns unexpected error", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, fmt.Errorf("new error"))
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).Should(MatchError("new error"))
		})

		It("calls destroy without error but cannot resolve VM after", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, -1, notFoundError)
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(Succeed())
		})

		It("calls destroy without error and identifies it as expunging", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).
				Return(&cloudstack.VirtualMachinesMetric{
					State: "Expunging",
				}, 1, nil)
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(Succeed())
		})

		It("calls destroy without error and identifies it as expunged", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).
				Return(&cloudstack.VirtualMachinesMetric{
					State: "Expunged",
				}, 1, nil)
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(Succeed())
		})

		It("calls destroy without error and identifies it as stopping", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).
				Return(&cloudstack.VirtualMachinesMetric{
					State: "Stopping",
				}, 1, nil)
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).Should(MatchError("VM deletion in progress"))
		})
	})
})
