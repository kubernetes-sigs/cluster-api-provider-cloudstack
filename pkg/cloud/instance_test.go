/*
Copyright 2023 The Kubernetes Authors.

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
	"encoding/base64"
	"fmt"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"

	"github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = ginkgo.Describe("Instance", func() {
	const (
		unknownErrorMessage = "unknown err"
		offeringFakeID      = "123"
		templateFakeID      = "456"
		executableFilter    = "executable"
		diskOfferingFakeID  = "789"
	)

	notFoundError := errors.New("no match found")
	unknownError := errors.New(unknownErrorMessage)

	var (
		mockCtrl      *gomock.Controller
		mockClient    *cloudstack.CloudStackClient
		configuration *cloudstack.MockConfigurationServiceIface
		vms           *cloudstack.MockVirtualMachineServiceIface
		sos           *cloudstack.MockServiceOfferingServiceIface
		dos           *cloudstack.MockDiskOfferingServiceIface
		ts            *cloudstack.MockTemplateServiceIface
		vs            *cloudstack.MockVolumeServiceIface
		client        cloud.Client
	)

	ginkgo.BeforeEach(func() {
		mockCtrl = gomock.NewController(ginkgo.GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		configuration = mockClient.Configuration.(*cloudstack.MockConfigurationServiceIface)
		vms = mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
		sos = mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
		dos = mockClient.DiskOffering.(*cloudstack.MockDiskOfferingServiceIface)
		ts = mockClient.Template.(*cloudstack.MockTemplateServiceIface)
		vs = mockClient.Volume.(*cloudstack.MockVolumeServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient, nil)

		dummies.SetDummyVars()
	})

	ginkgo.AfterEach(func() {
		mockCtrl.Finish()
	})

	ginkgo.Context("when fetching a VM instance", func() {
		ginkgo.It("Handles an unknown error when fetching by ID", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(nil, -1, unknownError)
			gomega.Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).To(gomega.MatchError(unknownErrorMessage))
		})

		ginkgo.It("Handles finding more than one VM instance by ID", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(nil, 2, nil)
			gomega.Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).
				Should(gomega.MatchError("found more than one VM Instance with ID " + *dummies.CSMachine1.Spec.InstanceID))
		})

		ginkgo.It("sets dummies.CSMachine1 spec and status values when VM instance found by ID", func() {
			vmsResp := &cloudstack.VirtualMachinesMetric{Id: *dummies.CSMachine1.Spec.InstanceID}
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(vmsResp, 1, nil)
			gomega.Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).Should(gomega.Succeed())
			gomega.Ω(dummies.CSMachine1.Spec.ProviderID).Should(gomega.Equal(ptr.To("cloudstack:///" + vmsResp.Id)))
			gomega.Ω(dummies.CSMachine1.Spec.InstanceID).Should(gomega.Equal(ptr.To(vmsResp.Id)))
		})

		ginkgo.It("handles an unknown error when fetching by name", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name, gomock.Any()).Return(nil, -1, unknownError)

			gomega.Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).Should(gomega.MatchError(unknownErrorMessage))
		})

		ginkgo.It("handles finding more than one VM instance by Name", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name, gomock.Any()).Return(nil, 2, nil)

			gomega.Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).Should(
				gomega.MatchError("found more than one VM Instance with name " + dummies.CSMachine1.Name))
		})

		ginkgo.It("sets dummies.CSMachine1 spec and status values when VM instance found by Name", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name, gomock.Any()).
				Return(&cloudstack.VirtualMachinesMetric{Id: *dummies.CSMachine1.Spec.InstanceID}, -1, nil)

			gomega.Ω(client.ResolveVMInstanceDetails(dummies.CSMachine1)).Should(gomega.Succeed())
			gomega.Ω(dummies.CSMachine1.Spec.ProviderID).Should(gomega.Equal(
				ptr.To(fmt.Sprintf("cloudstack:///%s", *dummies.CSMachine1.Spec.InstanceID))))
			gomega.Ω(dummies.CSMachine1.Spec.InstanceID).Should(gomega.Equal(ptr.To(*dummies.CSMachine1.Spec.InstanceID)))
		})
	})

	ginkgo.Context("when creating a VM instance", func() {
		vmMetricResp := &cloudstack.VirtualMachinesMetric{}

		expectVMNotFound := func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name, gomock.Any()).Return(nil, -1, notFoundError)
		}

		ginkgo.It("doesn't re-create if one already exists.", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(vmMetricResp, -1, nil)
			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
				Should(gomega.Succeed())
		})

		ginkgo.It("returns unknown error while fetching VM instance", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(nil, -1, unknownError)
			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
				Should(gomega.MatchError(unknownErrorMessage))
		})

		ginkgo.It("returns errors occurring while fetching service offering information", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).Return(&cloudstack.ServiceOffering{}, -1, unknownError)
			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
				ShouldNot(gomega.Succeed())
		})

		ginkgo.It("returns errors if more than one service offering found", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).Return(&cloudstack.ServiceOffering{
				Id:   dummies.CSMachine1.Spec.Offering.ID,
				Name: dummies.CSMachine1.Spec.Offering.Name,
			}, 2, nil)
			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
				ShouldNot(gomega.Succeed())
		})

		ginkgo.It("returns errors while fetching template", func() {
			expectVMNotFound()

			sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
				Return(&cloudstack.ServiceOffering{
					Id:   dummies.CSMachine1.Spec.Offering.ID,
					Name: dummies.CSMachine1.Spec.Offering.Name,
				}, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID, gomock.Any()).
				Return("", -1, unknownError)
			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
				ShouldNot(gomega.Succeed())
		})

		ginkgo.It("returns errors when more than one template found", func() {
			expectVMNotFound()

			sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
				Return(&cloudstack.ServiceOffering{
					Id:   dummies.CSMachine1.Spec.Offering.ID,
					Name: dummies.CSMachine1.Spec.Offering.Name,
				}, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID, gomock.Any()).Return("", 2, nil)
			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
				ShouldNot(gomega.Succeed())
		})

		ginkgo.It("returns errors when more than one diskoffering found", func() {
			expectVMNotFound()

			sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
				Return(&cloudstack.ServiceOffering{
					Id:   dummies.CSMachine1.Spec.Offering.ID,
					Name: dummies.CSMachine1.Spec.Offering.Name,
				}, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID, gomock.Any()).Return(dummies.CSMachine1.Spec.Template.ID, 1, nil)
			dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).Return(diskOfferingFakeID, 2, nil)
			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
				ShouldNot(gomega.Succeed())
		})

		ginkgo.It("returns errors when fetching diskoffering", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
				Return(&cloudstack.ServiceOffering{
					Id:   dummies.CSMachine1.Spec.Offering.ID,
					Name: dummies.CSMachine1.Spec.Offering.Name,
				}, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID, gomock.Any()).Return(dummies.CSMachine1.Spec.Template.ID, 1, nil)
			dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).Return(diskOfferingFakeID, 1, nil)
			dos.EXPECT().GetDiskOfferingByID(diskOfferingFakeID, gomock.Any()).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, unknownError)
			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
				ShouldNot(gomega.Succeed())
		})

		ginkgo.It("returns errors when disk size not zero for non-customized disk offering", func() {
			expectVMNotFound()
			dummies.CSMachine1.Spec.DiskOffering.CustomSize = 1
			sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
				Return(&cloudstack.ServiceOffering{
					Id:   dummies.CSMachine1.Spec.Offering.ID,
					Name: dummies.CSMachine1.Spec.Offering.Name,
				}, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID, gomock.Any()).Return(dummies.CSMachine1.Spec.Template.ID, 1, nil)
			dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).Return(diskOfferingFakeID, 1, nil)
			dos.EXPECT().GetDiskOfferingByID(diskOfferingFakeID, gomock.Any()).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)
			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
				ShouldNot(gomega.Succeed())
		})

		ginkgo.It("returns errors when disk size zero for customized disk offering", func() {
			expectVMNotFound()
			dummies.CSMachine1.Spec.DiskOffering.CustomSize = 0
			sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
				Return(&cloudstack.ServiceOffering{
					Id:        dummies.CSMachine1.Spec.Offering.ID,
					Name:      dummies.CSMachine1.Spec.Offering.Name,
					Cpunumber: 1,
					Memory:    1024,
				}, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID, gomock.Any()).Return(dummies.CSMachine1.Spec.Template.ID, 1, nil)
			dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).Return(diskOfferingFakeID, 1, nil)
			dos.EXPECT().GetDiskOfferingByID(diskOfferingFakeID, gomock.Any()).Return(&cloudstack.DiskOffering{Iscustomized: true}, 1, nil)
			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
				ShouldNot(gomega.Succeed())
		})

		ginkgo.Context("when account & domains have limits", func() {
			ginkgo.It("returns errors when there are not enough available CPU in account", func() {
				expectVMNotFound()
				dummies.CSMachine1.Spec.DiskOffering.CustomSize = 0
				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
					Return(&cloudstack.ServiceOffering{
						Id:        dummies.CSMachine1.Spec.Offering.ID,
						Name:      dummies.CSMachine1.Spec.Offering.Name,
						Cpunumber: 2,
						Memory:    1024,
					}, 1, nil)
				user := &cloud.User{
					Account: cloud.Account{
						Domain: cloud.Domain{
							CPUAvailable:    "20",
							MemoryAvailable: "2048",
							VMAvailable:     "20",
						},
						CPUAvailable:    "1",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
					Project: cloud.Project{
						ID:              "123",
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
				}
				c := cloud.NewClientFromCSAPIClient(mockClient, user)
				gomega.Ω(c.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError(gomega.MatchRegexp("CPU available .* in account can't fulfil the requirement:.*")))
			})

			ginkgo.It("returns errors when there are not enough available CPU in domain", func() {
				expectVMNotFound()
				dummies.CSMachine1.Spec.DiskOffering.CustomSize = 0
				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
					Return(&cloudstack.ServiceOffering{
						Id:        dummies.CSMachine1.Spec.Offering.ID,
						Name:      dummies.CSMachine1.Spec.Offering.Name,
						Cpunumber: 2,
						Memory:    1024,
					}, 1, nil)
				user := &cloud.User{
					Account: cloud.Account{
						Domain: cloud.Domain{
							CPUAvailable:    "1",
							MemoryAvailable: "2048",
							VMAvailable:     "20",
						},
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
					Project: cloud.Project{
						ID:              "123",
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
				}
				c := cloud.NewClientFromCSAPIClient(mockClient, user)
				gomega.Ω(c.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError(gomega.MatchRegexp("CPU available .* in domain can't fulfil the requirement:.*")))
			})

			ginkgo.It("returns errors when there are not enough available CPU in project", func() {
				expectVMNotFound()
				dummies.CSMachine1.Spec.DiskOffering.CustomSize = 0
				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
					Return(&cloudstack.ServiceOffering{
						Id:        dummies.CSMachine1.Spec.Offering.ID,
						Name:      dummies.CSMachine1.Spec.Offering.Name,
						Cpunumber: 2,
						Memory:    1024,
					}, 1, nil)
				user := &cloud.User{
					Account: cloud.Account{
						Domain: cloud.Domain{
							CPUAvailable:    "20",
							MemoryAvailable: "2048",
							VMAvailable:     "20",
						},
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
					Project: cloud.Project{
						ID:              "123",
						CPUAvailable:    "1",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
				}
				c := cloud.NewClientFromCSAPIClient(mockClient, user)
				gomega.Ω(c.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError(gomega.MatchRegexp("CPU available .* in project can't fulfil the requirement:.*")))
			})

			ginkgo.It("returns errors when there is not enough available memory in account", func() {
				expectVMNotFound()
				dummies.CSMachine1.Spec.DiskOffering.CustomSize = 0
				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
					Return(&cloudstack.ServiceOffering{
						Id:        dummies.CSMachine1.Spec.Offering.ID,
						Name:      dummies.CSMachine1.Spec.Offering.Name,
						Cpunumber: 2,
						Memory:    1024,
					}, 1, nil)
				user := &cloud.User{
					Account: cloud.Account{
						Domain: cloud.Domain{
							CPUAvailable:    "20",
							MemoryAvailable: "2048",
							VMAvailable:     "20",
						},
						CPUAvailable:    "20",
						MemoryAvailable: "512",
						VMAvailable:     "20",
					},
					Project: cloud.Project{
						ID:              "123",
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
				}
				c := cloud.NewClientFromCSAPIClient(mockClient, user)
				gomega.Ω(c.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError(gomega.MatchRegexp("memory available .* in account can't fulfil the requirement:.*")))
			})

			ginkgo.It("returns errors when there is not enough available memory in domain", func() {
				expectVMNotFound()
				dummies.CSMachine1.Spec.DiskOffering.CustomSize = 0
				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
					Return(&cloudstack.ServiceOffering{
						Id:        dummies.CSMachine1.Spec.Offering.ID,
						Name:      dummies.CSMachine1.Spec.Offering.Name,
						Cpunumber: 2,
						Memory:    1024,
					}, 1, nil)
				user := &cloud.User{
					Account: cloud.Account{
						Domain: cloud.Domain{
							CPUAvailable:    "20",
							MemoryAvailable: "512",
							VMAvailable:     "20",
						},
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
					Project: cloud.Project{
						ID:              "123",
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
				}
				c := cloud.NewClientFromCSAPIClient(mockClient, user)
				gomega.Ω(c.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError(gomega.MatchRegexp("memory available .* in domain can't fulfil the requirement:.*")))
			})

			ginkgo.It("returns errors when there is not enough available memory in project", func() {
				expectVMNotFound()
				dummies.CSMachine1.Spec.DiskOffering.CustomSize = 0
				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
					Return(&cloudstack.ServiceOffering{
						Id:        dummies.CSMachine1.Spec.Offering.ID,
						Name:      dummies.CSMachine1.Spec.Offering.Name,
						Cpunumber: 2,
						Memory:    1024,
					}, 1, nil)
				user := &cloud.User{
					Account: cloud.Account{
						Domain: cloud.Domain{
							CPUAvailable:    "20",
							MemoryAvailable: "2048",
							VMAvailable:     "20",
						},
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
					Project: cloud.Project{
						ID:              "123",
						CPUAvailable:    "20",
						MemoryAvailable: "512",
						VMAvailable:     "20",
					},
				}
				c := cloud.NewClientFromCSAPIClient(mockClient, user)
				gomega.Ω(c.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError(gomega.MatchRegexp("memory available .* in project can't fulfil the requirement:.*")))
			})

			ginkgo.It("returns errors when there is not enough available VM limit in account", func() {
				expectVMNotFound()
				dummies.CSMachine1.Spec.DiskOffering.CustomSize = 0
				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
					Return(&cloudstack.ServiceOffering{
						Id:        dummies.CSMachine1.Spec.Offering.ID,
						Name:      dummies.CSMachine1.Spec.Offering.Name,
						Cpunumber: 2,
						Memory:    1024,
					}, 1, nil)
				user := &cloud.User{
					Account: cloud.Account{
						Domain: cloud.Domain{
							CPUAvailable:    "20",
							MemoryAvailable: "2048",
							VMAvailable:     "20",
						},
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "0",
					},
					Project: cloud.Project{
						ID:              "123",
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
				}
				c := cloud.NewClientFromCSAPIClient(mockClient, user)
				gomega.Ω(c.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError("VM Limit in account has reached it's maximum value"))
			})

			ginkgo.It("returns errors when there is not enough available VM limit in domain", func() {
				expectVMNotFound()
				dummies.CSMachine1.Spec.DiskOffering.CustomSize = 0
				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
					Return(&cloudstack.ServiceOffering{
						Id:        dummies.CSMachine1.Spec.Offering.ID,
						Name:      dummies.CSMachine1.Spec.Offering.Name,
						Cpunumber: 2,
						Memory:    1024,
					}, 1, nil)
				user := &cloud.User{
					Account: cloud.Account{
						Domain: cloud.Domain{
							CPUAvailable:    "20",
							MemoryAvailable: "2048",
							VMAvailable:     "0",
						},
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "10",
					},
					Project: cloud.Project{
						ID:              "123",
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "20",
					},
				}
				c := cloud.NewClientFromCSAPIClient(mockClient, user)
				gomega.Ω(c.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError("VM Limit in domain has reached it's maximum value"))
			})

			ginkgo.It("returns errors when there is not enough available VM limit in project", func() {
				expectVMNotFound()
				dummies.CSMachine1.Spec.DiskOffering.CustomSize = 0
				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
					Return(&cloudstack.ServiceOffering{
						Id:        dummies.CSMachine1.Spec.Offering.ID,
						Name:      dummies.CSMachine1.Spec.Offering.Name,
						Cpunumber: 2,
						Memory:    1024,
					}, 1, nil)
				user := &cloud.User{
					Account: cloud.Account{
						Domain: cloud.Domain{
							CPUAvailable:    "20",
							MemoryAvailable: "2048",
							VMAvailable:     "10",
						},
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "10",
					},
					Project: cloud.Project{
						ID:              "123",
						CPUAvailable:    "20",
						MemoryAvailable: "2048",
						VMAvailable:     "0",
					},
				}
				c := cloud.NewClientFromCSAPIClient(mockClient, user)
				gomega.Ω(c.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError("VM Limit in project has reached it's maximum value"))
			})
		})

		ginkgo.It("handles deployment errors", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
				Return(&cloudstack.ServiceOffering{
					Id:        offeringFakeID,
					Name:      dummies.CSMachine1.Spec.Offering.Name,
					Cpunumber: 1,
					Memory:    1024,
				}, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID, gomock.Any()).
				Return(templateFakeID, 1, nil)
			dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).
				Return(diskOfferingFakeID, 1, nil)
			dos.EXPECT().GetDiskOfferingByID(diskOfferingFakeID, gomock.Any()).
				Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)
			vms.EXPECT().NewDeployVirtualMachineParams(offeringFakeID, templateFakeID, dummies.Zone1.ID).
				Return(&cloudstack.DeployVirtualMachineParams{})
			vms.EXPECT().DeployVirtualMachine(gomock.Any()).Return(nil, unknownError)
			vms.EXPECT().NewListVirtualMachinesParams().Return(&cloudstack.ListVirtualMachinesParams{})
			vms.EXPECT().ListVirtualMachines(gomock.Any()).Return(&cloudstack.ListVirtualMachinesResponse{}, nil)
			gomega.Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
				Should(gomega.MatchError(unknownErrorMessage))
		})

		ginkgo.Context("when using UUIDs and/or names to locate service offerings and templates", func() {
			ginkgo.BeforeEach(func() {
				gomock.InOrder(
					vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).
						Return(nil, -1, notFoundError),
					vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).
						Return(&cloudstack.VirtualMachinesMetric{}, 1, nil))

				vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name, gomock.Any()).Return(nil, -1, notFoundError)
			})

			ActionAndAssert := func() {
				vms.EXPECT().NewDeployVirtualMachineParams(offeringFakeID, templateFakeID, dummies.Zone1.ID).
					Return(&cloudstack.DeployVirtualMachineParams{})

				deploymentResp := &cloudstack.DeployVirtualMachineResponse{Id: *dummies.CSMachine1.Spec.InstanceID}

				expectUserData := "my special userdata"

				vms.EXPECT().DeployVirtualMachine(gomock.Any()).Do(
					func(p interface{}) {
						params := p.(*cloudstack.DeployVirtualMachineParams)
						displayName, _ := params.GetDisplayname()
						gomega.Ω(displayName == dummies.CAPIMachine.Name).Should(gomega.BeTrue())

						b64UserData, _ := params.GetUserdata()

						userData, err := base64.StdEncoding.DecodeString(b64UserData)
						gomega.Ω(err).ToNot(gomega.HaveOccurred())

						decompressedUserData, err := decompress(userData)
						gomega.Ω(err).ToNot(gomega.HaveOccurred())

						gomega.Ω(string(decompressedUserData)).To(gomega.Equal(expectUserData))
					}).Return(deploymentResp, nil)

				gomega.Ω(client.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, expectUserData)).
					Should(gomega.Succeed())
			}

			ginkgo.It("works with service offering name and template name", func() {
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.ID = ""
				dummies.CSMachine1.Spec.Template.ID = ""
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).Return(&cloudstack.ServiceOffering{
					Id:        offeringFakeID,
					Cpunumber: 1,
					Memory:    1024,
				}, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).Return(diskOfferingFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingByID(dummies.CSMachine1.Spec.DiskOffering.ID, gomock.Any()).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)
				ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID, gomock.Any()).
					Return(templateFakeID, 1, nil)

				ActionAndAssert()
			})

			ginkgo.It("works with service offering name and template name without disk offering", func() {
				dummies.CSMachine1.Spec.Offering.ID = ""
				dummies.CSMachine1.Spec.Template.ID = ""
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"
				dummies.CSMachine1.Spec.DiskOffering = infrav1.CloudStackResourceDiskOffering{}

				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).Return(&cloudstack.ServiceOffering{
					Id:        offeringFakeID,
					Cpunumber: 1,
					Memory:    1024,
				}, 1, nil)
				ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID, gomock.Any()).
					Return(templateFakeID, 1, nil)

				ActionAndAssert()
			})

			ginkgo.It("works with service offering ID and template name", func() {
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = ""
				dummies.CSMachine1.Spec.Offering.Name = ""
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID, gomock.Any()).Return(&cloudstack.ServiceOffering{
					Id:        offeringFakeID,
					Cpunumber: 1,
					Memory:    1024,
				}, 1, nil)
				ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID, gomock.Any()).
					Return(templateFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).Return(diskOfferingFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingByID(dummies.CSMachine1.Spec.DiskOffering.ID, gomock.Any()).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)

				ActionAndAssert()
			})

			ginkgo.It("works with service offering name and template ID", func() {
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.ID = ""
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = ""

				sos.EXPECT().GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).Return(&cloudstack.ServiceOffering{
					Id:        offeringFakeID,
					Cpunumber: 1,
					Memory:    1024,
				}, 1, nil)

				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, executableFilter, gomock.Any()).Return(&cloudstack.Template{Name: ""}, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).Return(diskOfferingFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingByID(dummies.CSMachine1.Spec.DiskOffering.ID, gomock.Any()).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)

				ActionAndAssert()
			})

			ginkgo.It("works with service offering ID and template ID", func() {
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = ""
				dummies.CSMachine1.Spec.Template.Name = ""

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID, gomock.Any()).
					Return(&cloudstack.ServiceOffering{
						Id:        offeringFakeID,
						Cpunumber: 1,
						Memory:    1024,
					}, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).
					Return(diskOfferingFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingByID(dummies.CSMachine1.Spec.DiskOffering.ID, gomock.Any()).
					Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, executableFilter, gomock.Any()).
					Return(&cloudstack.Template{Name: "template"}, 1, nil)

				ActionAndAssert()
			})

			ginkgo.It("works with Id and name both provided", func() {
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID, gomock.Any()).Return(&cloudstack.ServiceOffering{
					Id:        dummies.CSMachine1.Spec.Offering.ID,
					Name:      dummies.CSMachine1.Spec.Offering.Name,
					Cpunumber: 1,
					Memory:    1024,
				}, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, executableFilter, gomock.Any()).Return(&cloudstack.Template{Name: "template"}, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).Return(diskOfferingFakeID, 1, nil)
				dos.EXPECT().GetDiskOfferingByID(dummies.CSMachine1.Spec.DiskOffering.ID, gomock.Any()).Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)

				ActionAndAssert()
			})
		})

		ginkgo.Context("when using both UUIDs and names to locate service offerings and templates", func() {
			ginkgo.BeforeEach(func() {
				vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).
					Return(nil, -1, notFoundError)
				vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name, gomock.Any()).Return(nil, -1, notFoundError)
			})

			ginkgo.It("works with Id and name both provided, offering name mismatch", func() {
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID, gomock.Any()).Return(&cloudstack.ServiceOffering{Name: "offering-not-match"}, 1, nil)
				requiredRegexp := "offering name %s does not match name %s returned using UUID %s"
				gomega.Ω(client.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError(gomega.MatchRegexp(requiredRegexp, dummies.CSMachine1.Spec.Offering.Name, "offering-not-match", offeringFakeID)))
			})

			ginkgo.It("works with Id and name both provided, template name mismatch", func() {
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID, gomock.Any()).Return(&cloudstack.ServiceOffering{Name: "offering"}, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, executableFilter, gomock.Any()).Return(&cloudstack.Template{Name: "template-not-match"}, 1, nil)
				requiredRegexp := "template name %s does not match name %s returned using UUID %s"
				gomega.Ω(client.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError(gomega.MatchRegexp(requiredRegexp, dummies.CSMachine1.Spec.Template.Name, "template-not-match", templateFakeID)))
			})

			ginkgo.It("works with Id and name both provided, disk offering id/name mismatch", func() {
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"
				dummies.CSMachine1.Spec.DiskOffering.Name = "diskoffering"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID, gomock.Any()).Return(&cloudstack.ServiceOffering{Name: "offering"}, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, executableFilter, gomock.Any()).Return(&cloudstack.Template{Name: "template"}, 1, nil)
				dos.EXPECT().GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).Return(diskOfferingFakeID+"-not-match", 1, nil)
				requiredRegexp := "diskOffering ID %s does not match ID %s returned using name %s"
				gomega.Ω(client.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSFailureDomain1, dummies.CSAffinityGroup, "")).
					Should(gomega.MatchError(gomega.MatchRegexp(requiredRegexp, dummies.CSMachine1.Spec.DiskOffering.ID, diskOfferingFakeID+"-not-match", dummies.CSMachine1.Spec.DiskOffering.Name)))
			})
		})

		ginkgo.It("doesn't compress user data", func() {
			dummies.CSMachine1.Spec.DiskOffering.ID = diskOfferingFakeID
			dummies.CSMachine1.Spec.Offering.ID = ""
			dummies.CSMachine1.Spec.Template.ID = ""
			dummies.CSMachine1.Spec.Offering.Name = "offering"
			dummies.CSMachine1.Spec.Template.Name = "template"
			dummies.CSMachine1.Spec.UncompressedUserData = ptr.To(true)

			vms.EXPECT().
				GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).
				Return(nil, -1, notFoundError)
			vms.EXPECT().
				GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).
				Return(&cloudstack.VirtualMachinesMetric{}, 1, nil)
			vms.EXPECT().
				GetVirtualMachinesMetricByName(dummies.CSMachine1.Name, gomock.Any()).
				Return(nil, -1, notFoundError)

			sos.EXPECT().
				GetServiceOfferingByName(dummies.CSMachine1.Spec.Offering.Name, gomock.Any()).
				Return(&cloudstack.ServiceOffering{
					Id:        offeringFakeID,
					Cpunumber: 1,
					Memory:    1024,
				}, 1, nil)
			dos.EXPECT().
				GetDiskOfferingID(dummies.CSMachine1.Spec.DiskOffering.Name, gomock.Any()).
				Return(diskOfferingFakeID, 1, nil)
			dos.EXPECT().
				GetDiskOfferingByID(dummies.CSMachine1.Spec.DiskOffering.ID, gomock.Any()).
				Return(&cloudstack.DiskOffering{Iscustomized: false}, 1, nil)
			ts.EXPECT().
				GetTemplateID(dummies.CSMachine1.Spec.Template.Name, executableFilter, dummies.Zone1.ID, gomock.Any()).
				Return(templateFakeID, 1, nil)
			vms.EXPECT().
				NewDeployVirtualMachineParams(offeringFakeID, templateFakeID, dummies.Zone1.ID).
				Return(&cloudstack.DeployVirtualMachineParams{})

			deploymentResp := &cloudstack.DeployVirtualMachineResponse{
				Id: *dummies.CSMachine1.Spec.InstanceID,
			}

			expectUserData := "my special userdata"

			vms.EXPECT().DeployVirtualMachine(gomock.Any()).Do(
				func(p interface{}) {
					params := p.(*cloudstack.DeployVirtualMachineParams)
					displayName, _ := params.GetDisplayname()
					gomega.Ω(displayName == dummies.CAPIMachine.Name).Should(gomega.BeTrue())

					// Ensure the user data is only base64 encoded.
					b64UserData, _ := params.GetUserdata()
					userData, err := base64.StdEncoding.DecodeString(b64UserData)
					gomega.Ω(err).ToNot(gomega.HaveOccurred())
					gomega.Ω(string(userData)).To(gomega.Equal(expectUserData))
				}).Return(deploymentResp, nil)

			err := client.GetOrCreateVMInstance(
				dummies.CSMachine1,
				dummies.CAPIMachine,
				dummies.CSCluster,
				dummies.CSFailureDomain1,
				dummies.CSAffinityGroup,
				expectUserData,
			)
			gomega.Ω(err).Should(gomega.Succeed())
		})
	})

	ginkgo.Context("when destroying a VM instance", func() {
		listCapabilitiesParams := &cloudstack.ListCapabilitiesParams{}
		expungeDestroyParams := &cloudstack.DestroyVirtualMachineParams{}
		expungeDestroyParams.SetExpunge(true)
		listCapabilitiesResponse := &cloudstack.ListCapabilitiesResponse{
			Capabilities: &cloudstack.Capability{Allowuserexpungerecovervm: true},
		}
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

		ginkgo.BeforeEach(func() {
			configuration.EXPECT().NewListCapabilitiesParams().Return(listCapabilitiesParams)
			configuration.EXPECT().ListCapabilities(listCapabilitiesParams).Return(listCapabilitiesResponse, nil)
		})

		ginkgo.It("calls destroy and finds VM doesn't exist, then returns nil", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, fmt.Errorf("unable to find uuid for id"))
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			gomega.Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(gomega.Succeed())
		})

		ginkgo.It("calls destroy and returns unexpected error", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, fmt.Errorf("new error"))
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			gomega.Ω(client.DestroyVMInstance(dummies.CSMachine1)).Should(gomega.MatchError("new error"))
		})

		ginkgo.It("calls destroy without error but cannot resolve VM after", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name, gomock.Any()).Return(nil, -1, notFoundError)
			gomega.Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(gomega.Succeed())
		})

		ginkgo.It("calls destroy without error and identifies it as expunging", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).
				Return(&cloudstack.VirtualMachinesMetric{
					State: "Expunging",
				}, 1, nil)
			gomega.Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(gomega.Succeed())
		})

		ginkgo.It("calls destroy without error and identifies it as expunged", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).
				Return(&cloudstack.VirtualMachinesMetric{
					State: "Expunged",
				}, 1, nil)
			gomega.Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(gomega.Succeed())
		})

		ginkgo.It("calls destroy without error and identifies it as stopping", func() {
			listVolumesParams.SetVirtualmachineid(*dummies.CSMachine1.Spec.InstanceID)
			listVolumesParams.SetType("DATADISK")
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vs.EXPECT().NewListVolumesParams().Return(listVolumesParams)
			vs.EXPECT().ListVolumes(listVolumesParams).Return(listVolumesResponse, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID, gomock.Any()).
				Return(&cloudstack.VirtualMachinesMetric{
					State: "Stopping",
				}, 1, nil)
			gomega.Ω(client.DestroyVMInstance(dummies.CSMachine1)).Should(gomega.MatchError("VM deletion in progress"))
		})
	})
})
