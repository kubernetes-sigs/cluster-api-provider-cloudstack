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
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"

	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"
)

var _ = Describe("Instance", func() {
	notFoundError := errors.New("no match found")
	const unknownErrorMessage = "unknown err"
	const offeringFakeID = "123"
	const templateFakeID = "456"
	unknownError := errors.New(unknownErrorMessage)

	var (
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient
		vms        *cloudstack.MockVirtualMachineServiceIface
		sos        *cloudstack.MockServiceOfferingServiceIface
		ts         *cloudstack.MockTemplateServiceIface
		client     cloud.Client
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		vms = mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
		sos = mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
		ts = mockClient.Template.(*cloudstack.MockTemplateServiceIface)
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

		const allFilter = "all"
		It("returns errors while fetching template", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).
				Return(dummies.CSMachine1.Spec.Offering.ID, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, allFilter, dummies.Zone1.ID).
				Return("", -1, unknownError)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				ShouldNot(Succeed())
		})

		It("returns errors when more than one template found", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).
				Return(dummies.CSMachine1.Spec.Offering.ID, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, allFilter, dummies.Zone1.ID).Return("", 2, nil)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				ShouldNot(Succeed())
		})

		It("handles deployment errors", func() {
			expectVMNotFound()
			sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).
				Return(offeringFakeID, 1, nil)
			ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, allFilter, dummies.Zone1.ID).
				Return(templateFakeID, 1, nil)
			vms.EXPECT().NewDeployVirtualMachineParams(offeringFakeID, templateFakeID, dummies.Zone1.ID).
				Return(&cloudstack.DeployVirtualMachineParams{})
			vms.EXPECT().DeployVirtualMachine(gomock.Any()).Return(nil, unknownError)
			vms.EXPECT().NewListVirtualMachinesParams().Return(&cloudstack.ListVirtualMachinesParams{})
			vms.EXPECT().ListVirtualMachines(gomock.Any()).Return(&cloudstack.ListVirtualMachinesResponse{}, nil)
			Ω(client.GetOrCreateVMInstance(
				dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
				Should(MatchError(unknownErrorMessage))
		})

		// The folloing test checks that DomainID and Account are set (or not) in the DeployVirtualMachineParams
		// interface passed to DeployVirtualMachine.
		describeDomainAccountTest := func(desc string) func(string, string) string {
			return func(account string, domainID string) string {
				return fmt.Sprintf(`"%s" and "%s", %s`, account, domainID, desc)
			}
		}
		DescribeTable("DomainID and Account test table.",
			func(account string, domainID string) {
				dummies.CSCluster.Spec.Account = account
				dummies.CSCluster.Status.DomainID = domainID
				vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(nil, -1, notFoundError)
				sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).Return(offeringFakeID, 1, nil)
				ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, allFilter, dummies.Zone1.ID).
					Return(templateFakeID, 1, nil)
				vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).
					Return(&cloudstack.VirtualMachinesMetric{}, 1, nil)
				vms.EXPECT().NewDeployVirtualMachineParams(offeringFakeID, templateFakeID, dummies.Zone1.ID).
					Return(&cloudstack.DeployVirtualMachineParams{})
				vms.EXPECT().DeployVirtualMachine(ParamMatch(And(AccountEquals(account), DomainIDEquals(domainID)))).
					Return(&cloudstack.DeployVirtualMachineResponse{Id: *dummies.CSMachine1.Spec.InstanceID}, nil)
				vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, -1, notFoundError)

				Ω(client.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
					Should(Succeed())
			},
			Entry(describeDomainAccountTest("all set case"), "FakeAccount", "FakeDomainID"),
			Entry(describeDomainAccountTest("empty case"), "", ""),
		)

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
				dummies.CSMachine1.Spec.Offering.ID = ""
				dummies.CSMachine1.Spec.Template.ID = ""
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).Return(offeringFakeID, 1, nil)
				ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, allFilter, dummies.Zone1.ID).
					Return(templateFakeID, 1, nil)

				ActionAndAssert()
			})

			It("works with service offering ID and template name", func() {
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = ""
				dummies.CSMachine1.Spec.Offering.Name = ""
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID).Return(&cloudstack.ServiceOffering{Name: ""}, 1, nil)
				ts.EXPECT().GetTemplateID(dummies.CSMachine1.Spec.Template.Name, allFilter, dummies.Zone1.ID).
					Return(templateFakeID, 1, nil)

				ActionAndAssert()
			})

			It("works with service offering name and template ID", func() {
				dummies.CSMachine1.Spec.Offering.ID = ""
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = ""

				sos.EXPECT().GetServiceOfferingID(dummies.CSMachine1.Spec.Offering.Name).Return(offeringFakeID, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, allFilter).Return(&cloudstack.Template{Name: ""}, 1, nil)

				ActionAndAssert()
			})

			It("works with service offering ID and template ID", func() {
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = ""
				dummies.CSMachine1.Spec.Template.Name = ""

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID).Return(&cloudstack.ServiceOffering{Name: "offering"}, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, allFilter).Return(&cloudstack.Template{Name: "template"}, 1, nil)

				ActionAndAssert()
			})

			It("works with Id and name both provided", func() {
				dummies.CSMachine1.Spec.Offering.ID = offeringFakeID
				dummies.CSMachine1.Spec.Template.ID = templateFakeID
				dummies.CSMachine1.Spec.Offering.Name = "offering"
				dummies.CSMachine1.Spec.Template.Name = "template"

				sos.EXPECT().GetServiceOfferingByID(dummies.CSMachine1.Spec.Offering.ID).Return(&cloudstack.ServiceOffering{Name: "offering"}, 1, nil)
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, allFilter).Return(&cloudstack.Template{Name: "template"}, 1, nil)

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
				ts.EXPECT().GetTemplateByID(dummies.CSMachine1.Spec.Template.ID, allFilter).Return(&cloudstack.Template{Name: "template-not-match"}, 1, nil)
				requiredRegexp := "template name %s does not match name %s returned using UUID %s"
				Ω(client.GetOrCreateVMInstance(
					dummies.CSMachine1, dummies.CAPIMachine, dummies.CSCluster, dummies.CSZone1, dummies.CSAffinityGroup, "")).
					Should(MatchError(MatchRegexp(requiredRegexp, dummies.CSMachine1.Spec.Template.Name, "template-not-match", templateFakeID)))

			})
		})
	})

	Context("when destroying a VM instance", func() {
		expungeDestroyParams := &cloudstack.DestroyVirtualMachineParams{}
		expungeDestroyParams.SetExpunge(true)

		It("calls destroy and finds VM doesn't exist, then returns nil", func() {
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, fmt.Errorf("unable to find uuid for id"))
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(Succeed())
		})

		It("calls destroy and returns unexpected error", func() {
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, fmt.Errorf("new error"))
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).Should(MatchError("new error"))
		})

		It("calls destroy without error but cannot resolve VM after", func() {
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(dummies.CSMachine1.Name).Return(nil, -1, notFoundError)
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(Succeed())
		})

		It("calls destroy without error and identifies it as expunging", func() {
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).
				Return(&cloudstack.VirtualMachinesMetric{
					State: "Expunging",
				}, 1, nil)
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(Succeed())
		})

		It("calls destroy without error and identifies it as expunged", func() {
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).
				Return(&cloudstack.VirtualMachinesMetric{
					State: "Expunged",
				}, 1, nil)
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).
				Should(Succeed())
		})

		It("calls destroy without error and identifies it as stopping", func() {
			vms.EXPECT().NewDestroyVirtualMachineParams(*dummies.CSMachine1.Spec.InstanceID).
				Return(expungeDestroyParams)
			vms.EXPECT().DestroyVirtualMachine(expungeDestroyParams).Return(nil, nil)
			vms.EXPECT().GetVirtualMachinesMetricByID(*dummies.CSMachine1.Spec.InstanceID).
				Return(&cloudstack.VirtualMachinesMetric{
					State: "Stopping",
				}, 1, nil)
			Ω(client.DestroyVMInstance(dummies.CSMachine1)).Should(MatchError("VM deletion in progress"))
		})
	})
})
