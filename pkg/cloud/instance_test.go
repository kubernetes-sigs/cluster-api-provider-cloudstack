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

	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
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
	unknownError := errors.New(unknownErrorMessage)

	var (
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient
		vms        *cloudstack.MockVirtualMachineServiceIface
		csMachine  *infrav1.CloudStackMachine
		csCluster  *infrav1.CloudStackCluster
		sos        *cloudstack.MockServiceOfferingServiceIface
		ts         *cloudstack.MockTemplateServiceIface
		machine    *capiv1.Machine
		client     cloud.Client
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		vms = mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
		sos = mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
		ts = mockClient.Template.(*cloudstack.MockTemplateServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)

		csMachine = &infrav1.CloudStackMachine{
			Spec: infrav1.CloudStackMachineSpec{
				InstanceID: pointer.StringPtr("instance-id"),
				Offering:   "service-offering-name",
				Template:   "template-name"}}
		csMachine.Name = "instance-name"
		csCluster = &infrav1.CloudStackCluster{
			Spec:   infrav1.CloudStackClusterSpec{},
			Status: infrav1.CloudStackClusterStatus{ZoneID: "zone-id"}}
		machine = &capiv1.Machine{}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("when fetching a VM instance", func() {
		It("Handles an unknown error when fetching by ID", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, -1, unknownError)
			Ω(client.ResolveVMInstanceDetails(csMachine)).To(MatchError(unknownErrorMessage))
		})

		It("Handles finding more than one VM instance by ID", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, 2, nil)
			Ω(client.ResolveVMInstanceDetails(csMachine)).
				Should(MatchError("Found more than one VM Instance with ID instance-id."))
		})

		It("sets csMachine spec and status values when VM instance found by ID", func() {
			vmsResp := &cloudstack.VirtualMachinesMetric{Id: *csMachine.Spec.InstanceID}
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(vmsResp, 1, nil)
			Ω(client.ResolveVMInstanceDetails(csMachine)).Should(Succeed())
			Ω(csMachine.Spec.ProviderID).Should(Equal(pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", vmsResp.Id))))
			Ω(csMachine.Spec.InstanceID).Should(Equal(pointer.StringPtr(vmsResp.Id)))
		})

		It("handles an unknown error when fetching by name", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(csMachine.Name).Return(nil, -1, unknownError)

			Ω(client.ResolveVMInstanceDetails(csMachine)).Should(MatchError(unknownErrorMessage))
		})

		It("handles finding more than one VM instance by Name", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(csMachine.Name).Return(nil, 2, nil)

			Ω(client.ResolveVMInstanceDetails(csMachine)).Should(
				MatchError("Found more than one VM Instance with name instance-name."))
		})

		It("sets csMachine spec and status values when VM instance found by Name", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(csMachine.Name).
				Return(&cloudstack.VirtualMachinesMetric{Id: *csMachine.Spec.InstanceID}, -1, nil)

			Ω(client.ResolveVMInstanceDetails(csMachine)).Should(Succeed())
			Ω(csMachine.Spec.ProviderID).Should(Equal(
				pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", *csMachine.Spec.InstanceID))))
			Ω(csMachine.Spec.InstanceID).Should(Equal(pointer.StringPtr(*csMachine.Spec.InstanceID)))
		})
	})

	Context("when creating a VM instance", func() {
		vmMetricResp := &cloudstack.VirtualMachinesMetric{}
		serviceOfferingID := "service-offering-id"
		templateID := "template-id"

		It("doesn't re-create if one already exists.", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(vmMetricResp, -1, nil)
			Ω(client.GetOrCreateVMInstance(csMachine, machine, csCluster, "")).Should(Succeed())
		})

		It("returns unknown errors encountered while fetching VM instance", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, -1, unknownError)
			Ω(client.GetOrCreateVMInstance(csMachine, machine, csCluster, "")).Should(MatchError(unknownErrorMessage))
		})

		It("returns errors occuring while fetching sevice offering information", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(csMachine.Name).Return(nil, -1, notFoundError)
			sos.EXPECT().GetServiceOfferingID(csMachine.Spec.Offering).Return("", -1, unknownError)
			sos.EXPECT().GetServiceOfferingByID(csMachine.Spec.Offering).Return(nil, -1, unknownError)
			Ω(client.GetOrCreateVMInstance(csMachine, machine, csCluster, "")).ShouldNot(Succeed())
		})

		It("returns errors if more than one sevice offering found", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(csMachine.Name).Return(nil, -1, notFoundError)
			sos.EXPECT().GetServiceOfferingID(csMachine.Spec.Offering).Return("", 2, nil)
			sos.EXPECT().GetServiceOfferingByID(csMachine.Spec.Offering).Return(nil, -1, unknownError)
			Ω(client.GetOrCreateVMInstance(csMachine, machine, csCluster, "")).ShouldNot(Succeed())
		})

		const allFilter = "all"
		It("returns errors encountered while fetching template", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(csMachine.Name).Return(nil, -1, notFoundError)
			sos.EXPECT().GetServiceOfferingID(csMachine.Spec.Offering).Return(serviceOfferingID, 1, nil)
			ts.EXPECT().GetTemplateID(csMachine.Spec.Template, allFilter, csCluster.Status.ZoneID).
				Return("", -1, unknownError)
			ts.EXPECT().GetTemplateByID(csMachine.Spec.Template, allFilter).Return(nil, -1, unknownError)
			Ω(client.GetOrCreateVMInstance(csMachine, machine, csCluster, "")).ShouldNot(Succeed())
		})

		It("returns errors when more than one template found", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(csMachine.Name).Return(nil, -1, notFoundError)
			sos.EXPECT().GetServiceOfferingID(csMachine.Spec.Offering).Return(serviceOfferingID, 1, nil)
			ts.EXPECT().GetTemplateID(csMachine.Spec.Template, allFilter, csCluster.Status.ZoneID).Return("", 2, nil)
			ts.EXPECT().GetTemplateByID(csMachine.Spec.Template, allFilter).Return(nil, -1, unknownError)
			Ω(client.GetOrCreateVMInstance(csMachine, machine, csCluster, "")).ShouldNot(Succeed())
		})

		It("handles deployment errors", func() {
			vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(csMachine.Name).Return(nil, -1, notFoundError)
			sos.EXPECT().GetServiceOfferingID(csMachine.Spec.Offering).Return(serviceOfferingID, 1, nil)
			ts.EXPECT().GetTemplateID(csMachine.Spec.Template, allFilter, csCluster.Status.ZoneID).
				Return(templateID, 1, nil)
			vms.EXPECT().NewDeployVirtualMachineParams(serviceOfferingID, templateID, csCluster.Status.ZoneID).
				Return(&cloudstack.DeployVirtualMachineParams{})
			vms.EXPECT().DeployVirtualMachine(gomock.Any()).Return(nil, unknownError)

			Ω(client.GetOrCreateVMInstance(csMachine, machine, csCluster, "")).Should(MatchError(unknownErrorMessage))
		})

		// The folloing test checks that DomainId and Account are set (or not) in the DeployVirtualMachineParams
		// interface passed to DeployVirtualMachine.
		describeDomainAccountTest := func(desc string) func(string, string) string {
			return func(account string, domainID string) string {
				return fmt.Sprintf(`"%s" and "%s", %s`, account, domainID, desc)
			}
		}
		DescribeTable("DomainID and Account test table.",
			func(account string, domainID string) {
				csCluster.Spec.Account = account
				csCluster.Status.DomainID = domainID
				vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).Return(nil, -1, notFoundError)
				sos.EXPECT().GetServiceOfferingID(csMachine.Spec.Offering).Return(serviceOfferingID, 1, nil)
				ts.EXPECT().GetTemplateID(csMachine.Spec.Template, allFilter, csCluster.Status.ZoneID).
					Return(templateID, 1, nil)
				vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).
					Return(&cloudstack.VirtualMachinesMetric{}, 1, nil)
				vms.EXPECT().NewDeployVirtualMachineParams(serviceOfferingID, templateID, csCluster.Status.ZoneID).
					Return(&cloudstack.DeployVirtualMachineParams{})
				vms.EXPECT().DeployVirtualMachine(ParamMatch(And(AccountEquals(account), DomainIdEquals(domainID)))).
					Return(&cloudstack.DeployVirtualMachineResponse{Id: *csMachine.Spec.InstanceID}, nil)
				vms.EXPECT().GetVirtualMachinesMetricByName(csMachine.Name).Return(nil, -1, notFoundError)

				Ω(client.GetOrCreateVMInstance(csMachine, machine, csCluster, "")).Should(Succeed())
			},
			Entry(describeDomainAccountTest("all set case"), "FakeAccount", "FakeDomainID"),
			Entry(describeDomainAccountTest("empty case"), "", ""),
		)

		Context("when using UUIDs and/or names to locate service offerings and templates", func() {
			BeforeEach(func() {
				gomock.InOrder(
					vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).
						Return(nil, -1, notFoundError),
					vms.EXPECT().GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID).
						Return(&cloudstack.VirtualMachinesMetric{}, 1, nil))

				vms.EXPECT().GetVirtualMachinesMetricByName(csMachine.Name).Return(nil, -1, notFoundError)
			})

			ActionAndAssert := func() {
				vms.EXPECT().NewDeployVirtualMachineParams(serviceOfferingID, templateID, csCluster.Status.ZoneID).
					Return(&cloudstack.DeployVirtualMachineParams{})

				deploymentResp := &cloudstack.DeployVirtualMachineResponse{Id: *csMachine.Spec.InstanceID}
				vms.EXPECT().DeployVirtualMachine(gomock.Any()).Return(deploymentResp, nil)

				Ω(client.GetOrCreateVMInstance(csMachine, machine, csCluster, "")).Should(Succeed())
			}

			It("works with service offering name and template name", func() {
				sos.EXPECT().GetServiceOfferingID(csMachine.Spec.Offering).Return(serviceOfferingID, 1, nil)
				ts.EXPECT().GetTemplateID(csMachine.Spec.Template, allFilter, csCluster.Status.ZoneID).
					Return(templateID, 1, nil)

				ActionAndAssert()
			})

			It("works with service offering ID and template name", func() {
				csMachine.Spec.Offering = serviceOfferingID
				sos.EXPECT().GetServiceOfferingID(csMachine.Spec.Offering).Return("", -1, notFoundError)
				sos.EXPECT().GetServiceOfferingByID(csMachine.Spec.Offering).Return(&cloudstack.ServiceOffering{}, 1, nil)
				ts.EXPECT().GetTemplateID(csMachine.Spec.Template, allFilter, csCluster.Status.ZoneID).
					Return(templateID, 1, nil)

				ActionAndAssert()
			})

			It("works with service offering name and template ID", func() {
				csMachine.Spec.Template = templateID
				sos.EXPECT().GetServiceOfferingID(csMachine.Spec.Offering).Return(serviceOfferingID, 1, nil)
				ts.EXPECT().GetTemplateID(csMachine.Spec.Template, allFilter, csCluster.Status.ZoneID).
					Return("", -1, notFoundError)
				ts.EXPECT().GetTemplateByID(csMachine.Spec.Template, allFilter).
					Return(&cloudstack.Template{}, 1, nil)

				ActionAndAssert()
			})

			It("works with service offering ID and template ID", func() {
				csMachine.Spec.Offering = serviceOfferingID
				csMachine.Spec.Template = templateID
				sos.EXPECT().GetServiceOfferingID(csMachine.Spec.Offering).Return("", -1, notFoundError)
				sos.EXPECT().GetServiceOfferingByID(csMachine.Spec.Offering).Return(&cloudstack.ServiceOffering{}, 1, nil)
				ts.EXPECT().GetTemplateID(csMachine.Spec.Template, allFilter, csCluster.Status.ZoneID).
					Return("", -1, notFoundError)
				ts.EXPECT().GetTemplateByID(csMachine.Spec.Template, allFilter).
					Return(&cloudstack.Template{}, 1, nil)

				ActionAndAssert()
			})
		})
	})
})
