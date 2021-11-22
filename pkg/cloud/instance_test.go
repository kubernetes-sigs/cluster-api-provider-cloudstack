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

package cloud

import (
	"fmt"
	"testing"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	_ "github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
	"k8s.io/utils/pointer"
)

func TestInstance(t *testing.T) {
	instanceID := "instance-id"
	instanceName := "instance-name"

	mockCtrl := gomock.NewController(t)
	mockClient := cloudstack.NewMockClient(mockCtrl)
	csMachine := &infrav1.CloudStackMachine{
		Spec: infrav1.CloudStackMachineSpec{}}
	csMachine.Spec.InstanceID = &instanceID
	csMachine.Name = instanceName

	t.Run("Fetching VM instance", func(t *testing.T) {
		t.Run("Unknown error by ID", func(t *testing.T) {
			g := NewWithT(t)
			expectedErr := errors.New("unknown error")
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, expectedErr)

			if err := FetchVMInstance(mockClient, csMachine); err != nil {
				g.Expect(errors.Cause(err)).To(MatchError(expectedErr))
			} else {
				t.Error()
			}
		})

		t.Run("More than one found by ID", func(t *testing.T) {
			g := NewWithT(t)
			expectedErr := "Found more than one VM Instance"
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, 2, nil)

			if err := FetchVMInstance(mockClient, csMachine); err != nil {
				g.Expect(errors.Cause(err).Error()).To(ContainSubstring(expectedErr))
			} else {
				t.Error()
			}
		})

		t.Run("One found by ID", func(t *testing.T) {
			g := NewWithT(t)
			vmsResp := &cloudstack.VirtualMachinesMetric{
				Id: instanceID}
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(vmsResp, 1, nil)

			if err := FetchVMInstance(mockClient, csMachine); err != nil {
				t.Error()
			} else {
				g.Expect(csMachine.Spec.ProviderID).To(Equal(pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", vmsResp.Id))))
				g.Expect(csMachine.Spec.InstanceID).To(Equal(pointer.StringPtr(vmsResp.Id)))
			}
		})

		t.Run("Not found by ID and unknown error by name", func(t *testing.T) {
			g := NewWithT(t)
			errorFromById := errors.New("no match found")
			expectedErr := errors.New("unknown error")
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, errorFromById)
			vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, expectedErr)

			if err := FetchVMInstance(mockClient, csMachine); err != nil {
				g.Expect(errors.Cause(err)).To(MatchError(expectedErr))
			} else {
				t.Error()
			}
		})

		t.Run("More than one found by Name", func(t *testing.T) {
			g := NewWithT(t)
			errorFromById := errors.New("no match found")
			expectedErr := "Found more than one VM Instance"
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, errorFromById)
			vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, 2, nil)

			if err := FetchVMInstance(mockClient, csMachine); err != nil {
				g.Expect(errors.Cause(err).Error()).To(ContainSubstring(expectedErr))
			} else {
				t.Error()
			}
		})

		t.Run("One found by Name", func(t *testing.T) {
			g := NewWithT(t)
			errorFromById := errors.New("no match found")
			vmsResp := &cloudstack.VirtualMachinesMetric{
				Id: instanceID}
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, errorFromById)
			vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(vmsResp, -1, nil)

			if err := FetchVMInstance(mockClient, csMachine); err != nil {
				t.Error()
			} else {
				g.Expect(csMachine.Spec.ProviderID).To(Equal(pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", vmsResp.Id))))
				g.Expect(csMachine.Spec.InstanceID).To(Equal(pointer.StringPtr(vmsResp.Id)))
			}
		})
	})

	t.Run("Creating VM instance", func(t *testing.T) {
		csCluster := &infrav1.CloudStackCluster{
			Spec: infrav1.CloudStackClusterSpec{}}
		serviceOfferingName := "service-offering-name"
		serviceOfferingID := "service-offering-id"
		zoneID := "zone-id"
		templateName := "template-name"
		csMachine.Spec.Template = templateName
		csCluster.Status.ZoneID = zoneID

		t.Run("No error from fetching VM instance", func(t *testing.T) {
			g := NewWithT(t)
			vmsResp := &cloudstack.VirtualMachinesMetric{
				Id: instanceID}
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(vmsResp, -1, nil)

			err := CreateVMInstance(mockClient, csMachine, csCluster)
			g.Expect(err).To(BeNil())
		})

		t.Run("Unknown error from fetching VM instance", func(t *testing.T) {
			g := NewWithT(t)
			expectedErr := errors.New("unknown error")
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, expectedErr)

			if err := CreateVMInstance(mockClient, csMachine, csCluster); err != nil {
				g.Expect(errors.Cause(err)).To(MatchError(expectedErr))
			} else {
				t.Error()
			}
		})

		t.Run("Error from sevice offering", func(t *testing.T) {
			csMachine.Spec.Offering = serviceOfferingName

			t.Run("Service offering error when instance VM not found", func(t *testing.T) {
				g := NewWithT(t)
				notFoundError := errors.New("no match found")
				vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
				vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, notFoundError)
				vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, notFoundError)

				expectedErr := errors.New("service offering error")
				sos := mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
				sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return("", -1, expectedErr)

				if err := CreateVMInstance(mockClient, csMachine, csCluster); err != nil {
					g.Expect(errors.Cause(err)).To(MatchError(expectedErr))
				} else {
					t.Error()
				}
			})

			t.Run("More than one service offering", func(t *testing.T) {
				g := NewWithT(t)
				notFoundError := errors.New("no match found")
				vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
				vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, notFoundError)
				vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, notFoundError)

				expectedErr := "Did not find exactly one machine offering"
				sos := mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
				sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return("", 2, nil)

				if err := CreateVMInstance(mockClient, csMachine, csCluster); err != nil {
					g.Expect(errors.Cause(err).Error()).To(ContainSubstring(expectedErr))
				} else {
					t.Error()
				}
			})
		})

		t.Run("Error from template", func(t *testing.T) {

			t.Run("Template error when instance VM not found", func(t *testing.T) {
				g := NewWithT(t)
				notFoundError := errors.New("no match found")
				vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
				vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, notFoundError)
				vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, notFoundError)

				sos := mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
				sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return(serviceOfferingID, 1, nil)

				expectedErr := errors.New("template error")
				ts := mockClient.Template.(*cloudstack.MockTemplateServiceIface)
				ts.EXPECT().GetTemplateID(templateName, "all", zoneID).Return("", -1, expectedErr)

				if err := CreateVMInstance(mockClient, csMachine, csCluster); err != nil {
					g.Expect(errors.Cause(err)).To(MatchError(expectedErr))
				} else {
					t.Error()
				}
			})

			t.Run("More than one template", func(t *testing.T) {
				g := NewWithT(t)
				notFoundError := errors.New("no match found")
				vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
				vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, notFoundError)
				vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, notFoundError)

				sos := mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
				sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return(serviceOfferingID, 1, nil)

				expectedErr := "Did not find exactly one template"
				ts := mockClient.Template.(*cloudstack.MockTemplateServiceIface)
				ts.EXPECT().GetTemplateID(templateName, "all", zoneID).Return("", 2, nil)

				if err := CreateVMInstance(mockClient, csMachine, csCluster); err != nil {
					g.Expect(errors.Cause(err).Error()).To(ContainSubstring(expectedErr))
				} else {
					t.Error()
				}
			})
		})

		t.Run("Error from deployment", func(t *testing.T) {
			g := NewWithT(t)
			templateID := "template-id"
			notFoundError := errors.New("no match found")
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Return(nil, -1, notFoundError)

			sos := mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
			sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return(serviceOfferingID, 1, nil)

			ts := mockClient.Template.(*cloudstack.MockTemplateServiceIface)
			ts.EXPECT().GetTemplateID(templateName, "all", zoneID).Return(templateID, 1, nil)

			params := &cloudstack.DeployVirtualMachineParams{}
			vms.EXPECT().NewDeployVirtualMachineParams(serviceOfferingID, templateID, zoneID).Return(params)

			expectedErr := errors.New("deployment error")
			vms.EXPECT().DeployVirtualMachine(params).Return(nil, expectedErr)

			if err := CreateVMInstance(mockClient, csMachine, csCluster); err != nil {
				g.Expect(errors.Cause(err)).To(MatchError(expectedErr))
			} else {
				t.Error()
			}
		})

		t.Run("Successful deployment", func(t *testing.T) {
			g := NewWithT(t)
			templateID := "template-id"
			notFoundError := errors.New("no match found")
			expectedErr := "machine not found"
			vms := mockClient.VirtualMachine.(*cloudstack.MockVirtualMachineServiceIface)
			vms.EXPECT().GetVirtualMachinesMetricByID(instanceID).Times(2).Return(nil, -1, notFoundError)
			vms.EXPECT().GetVirtualMachinesMetricByName(instanceName).Times(2).Return(nil, -1, notFoundError)

			sos := mockClient.ServiceOffering.(*cloudstack.MockServiceOfferingServiceIface)
			sos.EXPECT().GetServiceOfferingID(serviceOfferingName).Return(serviceOfferingID, 1, nil)

			ts := mockClient.Template.(*cloudstack.MockTemplateServiceIface)
			ts.EXPECT().GetTemplateID(templateName, "all", zoneID).Return(templateID, 1, nil)

			params := &cloudstack.DeployVirtualMachineParams{}
			vms.EXPECT().NewDeployVirtualMachineParams(serviceOfferingID, templateID, zoneID).Return(params)

			deploymentResp := &cloudstack.DeployVirtualMachineResponse{
				Id: instanceID}
			vms.EXPECT().DeployVirtualMachine(params).Return(deploymentResp, nil)

			if err := CreateVMInstance(mockClient, csMachine, csCluster); err != nil {
				g.Expect(errors.Cause(err).Error()).To(Equal(expectedErr))
				g.Expect(csMachine.Status.Ready).To(BeFalse())
			} else {
				t.Error()
			}
		})
	})
}
