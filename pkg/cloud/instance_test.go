/*
Copyright 2021.

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
	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
	"gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/pkg/cloud"
)

var _ = Describe("Instance", func() {
	instanceID := "instance-id"
	instanceName := "instance-name"
	//notFoundError := errors.New("no match found")

	var (
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	csMachine := &infrav1.CloudStackMachine{
		Spec: infrav1.CloudStackMachineSpec{}}
	csMachine.Spec.InstanceID = &instanceID
	csMachine.Name = instanceName

	It("Fetches a VM instance", func() {
		Context("Unknown error when fetching by ID", func() {
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, errors.New("unknown error"))

			Ω(cloud.FetchVMInstance(mockClient, csMachine)).To(MatchError("unknown error"))
		})

		Context("More than one VM instance found by ID", func() {
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, 2, nil)

			Ω(cloud.FetchVMInstance(mockClient, csMachine)).
				Should(MatchError("Found more than one VM Instance with ID instance-id."))
		})

		// Context("Only one VM instance found by ID", func() {
		// 	vmsResp := &cloudstack.VirtualMachinesMetric{
		// 		Id: instanceID}
		// 	vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
		// 	vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(vmsResp, 1, nil)

		// 	Ω(cloud.FetchVMInstance(mockClient, csMachine)).Should(Succeed())
		// 	Ω(csMachine.Spec.ProviderID).Should(Equal(pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", vmsResp.Id))))
		// 	Ω(csMachine.Spec.InstanceID).Should(Equal(pointer.StringPtr(vmsResp.Id)))
		// })

		// Context("Not found by ID and unknown error by name", func() {
		// 	vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
		// 	vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, notFoundError)
		// 	vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, errors.New("unknown error"))

		// 	Ω(cloud.FetchVMInstance(mockClient, csMachine)).Should(MatchError("unkown error"))
		// })

		// Context("More than one VM instance found by Name", func() {
		// 	vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
		// 	vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, errors.New("no match"))
		// 	vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, 2, nil)

		// 	Ω(cloud.FetchVMInstance(mockClient, csMachine)).Should(MatchError("Found more than one VM Instance"))
		// })

		// Context("One VM instance found by Name", func() {
		// 	errorFromById := notFoundError
		// 	vmsResp := &cloudstack.VirtualMachinesMetric{
		// 		Id: instanceID}
		// 	vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
		// 	vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, errorFromById)
		// 	vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(vmsResp, -1, nil)

		// 	Ω(cloud.FetchVMInstance(mockClient, csMachine)).Should(Succeed())
		// 	Ω(csMachine.Spec.ProviderID).Should(Equal(pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", vmsResp.Id))))
		// 	Ω(csMachine.Spec.InstanceID).Should(Equal(pointer.StringPtr(vmsResp.Id)))
		// })
	})

	// Context("Creating VM instance", func() {
	// 	csCluster := &infrav1.CloudStackCluster{
	// 		Spec: infrav1.CloudStackClusterSpec{}}
	// 	serviceOfferingName := "service-offering-name"
	// 	serviceOfferingID := "service-offering-id"
	// 	zoneID := "zone-id"
	// 	templateName := "template-name"
	// 	csMachine.Spec.Template = templateName
	// 	csCluster.Status.ZoneID = zoneID

	// 	It("Succeeds", func() {
	// 		vmsResp := &cloudstack.VirtualMachinesMetric{Id: instanceID}
	// 		vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
	// 		vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(vmsResp, -1, nil)

	// 		Ω(cloud.CreateVMInstance(mockClient, csMachine, csCluster)).Should(Succeed())
	// 	})

	// 	It("Returns unknown errors encountered while fetching VM instance", func(t *testing.T) {
	// 		expectedErr := errors.New("unknown error")
	// 		vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
	// 		vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, expectedErr)

	// 		Ω(cloud.CreateVMInstance(mockClient, csMachine, csCluster)).Should(MatchError("unknown error"))
	// 	})

	// 	It("Handles errors occuring while fetching sevice offering information", func(t *testing.T) {
	// 		csMachine.Spec.Offering = serviceOfferingName

	// 		vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
	// 		vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, notFoundError)
	// 		vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, notFoundError)

	// 		sos := mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
	// 		sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return("", -1, "so error")

	// 		Ω(cloud.CreateVMInstance(mockClient, csMachine, csCluster)).Should(MatchError("so error"))

	// 		vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, notFoundError)
	// 		vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, notFoundError)
	// 		sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return("", 2, nil)

	// 		Ω(cloud.CreateVMInstance(mockClient, csMachine, csCluster)).Should(
	// 			MatchError("Did not find exactly one machine offering"))
	// 	})

	// 	Context("While fetching template information", func() {
	// 		It("returns errors in template fetching", func() {
	// 			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
	// 			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, notFoundError)
	// 			vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, notFoundError)

	// 			sos := mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
	// 			sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return(serviceOfferingID, 1, nil)

	// 			ts := mockClient.Template.(*cloudstack.MockTemplateServiceIface)
	// 			ts.EXPECT().GetTemplateID(templateName, "all", zoneID).Return("", -1, "template error")

	// 			Ω(cloud.CreateVMInstance(mockClient, csMachine, csCluster)).Should(MatchError("template error"))
	// 		})

	// 		It("errors when more than one template found", func() {
	// 			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
	// 			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, notFoundError)
	// 			vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, notFoundError)

	// 			sos := mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
	// 			sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return(serviceOfferingID, 1, nil)

	// 			ts := mockClient.Template.(*cloudstack.MockTemplateServiceIface)
	// 			ts.EXPECT().GetTemplateID(templateName, "all", zoneID).Return("", 2, nil)

	// 			Ω(cloud.CreateVMInstance(mockClient, csMachine, csCluster)).
	// 				Should(MatchError("Did not find exactly one template"))
	// 		})
	// 	})

	// 	Context("During VM instance deployment", func() {
	// 		templateID := "template-id"
	// 		vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
	// 		vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, notFoundError)
	// 		vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, notFoundError)

	// 		sos := mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
	// 		sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return(serviceOfferingID, 1, nil)

	// 		ts := mockClient.Template.(*cloudstack.MockTemplateServiceIface)
	// 		ts.EXPECT().GetTemplateID(templateName, "all", zoneID).Return(templateID, 1, nil)

	// 		params := &cloudstack.DeployVirtualMachineParams{}
	// 		vms.EXPECT().NewDeployVirtualMachineParams(serviceOfferingID, templateID, zoneID).Return(params)

	// 		vms.EXPECT().DeployVirtualMachine(params).Return(nil, "deployment error")

	// 		Ω(cloud.CreateVMInstance(mockClient, csMachine, csCluster)).Should(MatchError("deployment error"))
	// 	})

	// 	It("Can deploy a VM Instance", func(t *testing.T) {
	// 		templateID := "template-id"
	// 		vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
	// 		vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Times(2).Return(nil, -1, notFoundError)
	// 		vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Times(2).Return(nil, -1, notFoundError)

	// 		sos := mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
	// 		sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return(serviceOfferingID, 1, nil)

	// 		ts := mockClient.Template.(*cloudstack.MockTemplateServiceIface)
	// 		ts.EXPECT().GetTemplateID(templateName, "all", zoneID).Return(templateID, 1, nil)

	// 		params := &cloudstack.DeployVirtualMachineParams{}
	// 		vms.EXPECT().NewDeployVirtualMachineParams(serviceOfferingID, templateID, zoneID).Return(params)

	// 		deploymentResp := &cloudstack.DeployVirtualMachineResponse{Id: instanceID}
	// 		vms.EXPECT().DeployVirtualMachine(params).Return(deploymentResp, nil)

	// 		Ω(cloud.CreateVMInstance(mockClient, csMachine, csCluster)).Should(Succeed())
	// 		Ω(csMachine.Status.Ready).Should(BeFalse())
	// 	})
	// // // // // // // // })
})
