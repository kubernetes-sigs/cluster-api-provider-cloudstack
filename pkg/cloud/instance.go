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
	"errors"
	"fmt"
	"strings"

	capiv1 "sigs.k8s.io/cluster-api/api/v1alpha4"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/util"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
	"k8s.io/utils/pointer"
)

// Set infrastructure spec and status from the CloudStack API's virtual machine metrics type.
func setMachineDataFromVMMetrics(vmResponse *cloudstack.VirtualMachinesMetric, csMachine *infrav1.CloudStackMachine) {
	csMachine.Spec.ProviderID = pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", vmResponse.Id))
	csMachine.Spec.InstanceID = pointer.StringPtr(vmResponse.Id)
	csMachine.Status.Addresses = []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: vmResponse.Ipaddress}}
	csMachine.Status.InstanceState = infrav1.InstanceState(vmResponse.State)
}

// Retch retrieves VM instance details by csMachine.Spec.InstanceID or csMachine.Name, and
// sets infrastructure machine spec and status if VM instance is found.
func (c *client) ResolveVMInstanceDetails(csMachine *infrav1.CloudStackMachine) error {
	// Attempt to fetch by ID.
	if csMachine.Spec.InstanceID != nil {
		vmResp, count, err := c.cs.VirtualMachine.GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match found") {
			return err
		} else if count > 1 {
			return fmt.Errorf("Found more than one VM Instance with ID %s.", *csMachine.Spec.InstanceID)
		} else if err == nil {
			setMachineDataFromVMMetrics(vmResp, csMachine)
			return nil
		}
	}

	// Attempt fetch by name.
	if csMachine.Name != "" {
		vmResp, count, err := c.cs.VirtualMachine.GetVirtualMachinesMetricByName(csMachine.Name) // add opts usage
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match") {
			return err
		} else if count > 1 {
			return fmt.Errorf("Found more than one VM Instance with name %s.", csMachine.Name)
		} else if err == nil {
			setMachineDataFromVMMetrics(vmResp, csMachine)
			return nil
		}
	}
	return errors.New("no match found")
}

// CreateVMInstance will fetch or create a VM instance, and
// sets the infrastructure machine spec and status accordingly.
func (c *client) GetOrCreateVMInstance(
	csMachine *infrav1.CloudStackMachine,
	machine *capiv1.Machine,
	csCluster *infrav1.CloudStackCluster,
	userData string) error {

	// Check if VM instance already exists.
	if err := c.ResolveVMInstanceDetails(csMachine); err == nil ||
		!strings.Contains(strings.ToLower(err.Error()), "no match") {
		return err
	}
	// Get machine offering ID from name.
	offeringID, count, err := c.cs.ServiceOffering.GetServiceOfferingID(csMachine.Spec.Offering)
	if err != nil {
		return err
	} else if count != 1 {
		return fmt.Errorf(
			"Did not find exactly one machine offering with the name %s", csMachine.Spec.Offering)
	}

	// Get template ID from name.
	templateID, count, err := c.cs.Template.GetTemplateID(csMachine.Spec.Template, "all", csCluster.Status.ZoneID)
	if err != nil {
		return err
	} else if count != 1 {
		return fmt.Errorf(
			"Did not find exactly one template with the name %s", csMachine.Spec.Template)
	}

	// Create VM instance.
	p := c.cs.VirtualMachine.NewDeployVirtualMachineParams(offeringID, templateID, csCluster.Status.ZoneID)
	p.SetNetworkids([]string{csCluster.Status.NetworkID})
	setIfNotEmpty(csMachine.Name, p.SetName)
	setIfNotEmpty(csMachine.Name, p.SetDisplayname)
	setIfNotEmpty(csMachine.Spec.SSHKey, p.SetKeypair)
	setIfNotEmpty(userData, p.SetUserdata)
	if util.IsControlPlaneMachine(machine) && csCluster.Status.NetworkType == NetworkTypeShared {
		setIfNotEmpty(csCluster.Spec.ControlPlaneEndpoint.Host, p.SetIpaddress)
	}
	if csMachine.Spec.Details != nil {
		p.SetDetails(csMachine.Spec.Details)
	}

	deployVMResp, err := c.cs.VirtualMachine.DeployVirtualMachine(p)
	if err != nil {
		return err
	}
	csMachine.Spec.InstanceID = pointer.StringPtr(deployVMResp.Id)

	// Resolve uses a VM metrics request response to fill cloudstack machine status.
	// The deployment response is insufficient.
	if err = c.ResolveVMInstanceDetails(csMachine); err == nil {
		csMachine.Status.Ready = true
	}
	return err
}

// Destroy a VM instane. Assumes machine has been fetched prior and has an instance ID.
func (c *client) DestroyVMInstance(csMachine *infrav1.CloudStackMachine) error {

	p := c.cs.VirtualMachine.NewDestroyVirtualMachineParams(*csMachine.Spec.InstanceID)
	p.SetExpunge(true)
	_, err := c.cs.VirtualMachine.DestroyVirtualMachine(p)

	return err
}
