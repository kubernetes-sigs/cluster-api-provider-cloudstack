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

package cloud

import (
	"fmt"
	"net"

	"strings"

	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

const antiAffinityValue = "anti"

type VMIface interface {
	GetOrCreateVMInstance(*infrav1.CloudStackMachine, *capiv1.Machine, *infrav1.CloudStackCluster, string) error
	ResolveVMInstanceDetails(*infrav1.CloudStackMachine) error
	DestroyVMInstance(*infrav1.CloudStackMachine) error
	AssignVMToLoadBalancerRule(*infrav1.CloudStackCluster, string) error
}

// Set infrastructure spec and status from the CloudStack API's virtual machine metrics type.
func setMachineDataFromVMMetrics(vmResponse *cloudstack.VirtualMachinesMetric, csMachine *infrav1.CloudStackMachine) {
	csMachine.Spec.ProviderID = pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", vmResponse.Id))
	csMachine.Spec.InstanceID = pointer.StringPtr(vmResponse.Id)
	csMachine.Status.Addresses = []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: vmResponse.Ipaddress}}
	csMachine.Status.InstanceState = infrav1.InstanceState(vmResponse.State)
}

// ResolveVMInstanceDetails Retrieves VM instance details by csMachine.Spec.InstanceID or csMachine.Name, and
// sets infrastructure machine spec and status if VM instance is found.
func (c *client) ResolveVMInstanceDetails(csMachine *infrav1.CloudStackMachine) error {
	// Attempt to fetch by ID.
	if csMachine.Spec.InstanceID != nil {
		vmResp, count, err := c.cs.VirtualMachine.GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match found") {
			return err
		} else if count > 1 {
			return fmt.Errorf("found more than one VM Instance with ID %s", *csMachine.Spec.InstanceID)
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
			return fmt.Errorf("found more than one VM Instance with name %s", csMachine.Name)
		} else if err == nil {
			setMachineDataFromVMMetrics(vmResp, csMachine)
			return nil
		}
	}
	return errors.New("no match found")
}

func (c *client) ResolveServiceOffering(csMachine *infrav1.CloudStackMachine) (offeringID string, retErr error) {
	if len(csMachine.Spec.Offering.ID) > 0 {
		csOffering, count, err := c.cs.ServiceOffering.GetServiceOfferingByID(csMachine.Spec.Offering.ID)
		if err != nil {
			return "", multierror.Append(retErr, errors.Wrapf(
				err, "could not get Service Offering by ID %s", csMachine.Spec.Offering.ID))
		} else if count != 1 {
			return "", multierror.Append(retErr, errors.Errorf(
				"expected 1 Service Offering with UUID %s, but got %d", csMachine.Spec.Offering.ID, count))
		}

		if len(csMachine.Spec.Offering.Name) > 0 && csMachine.Spec.Offering.Name != csOffering.Name {
			return "", multierror.Append(retErr, errors.Errorf(
				"offering name %s does not match name %s returned using UUID %s", csMachine.Spec.Offering.Name, csOffering.Name, csMachine.Spec.Offering.ID))
		}
		return csMachine.Spec.Offering.ID, nil
	}
	offeringID, count, err := c.cs.ServiceOffering.GetServiceOfferingID(csMachine.Spec.Offering.Name)
	if err != nil {
		return "", multierror.Append(retErr, errors.Wrapf(
			err, "could not get Service Offering ID from %s", csMachine.Spec.Offering.Name))
	} else if count != 1 {
		return "", multierror.Append(retErr, errors.Errorf(
			"expected 1 Service Offering with name %s, but got %d", csMachine.Spec.Offering.Name, count))
	}
	return offeringID, nil
}

func (c *client) ResolveTemplate(
	csCluster *infrav1.CloudStackCluster,
	csMachine *infrav1.CloudStackMachine,
	zoneID string,
) (templateID string, retErr error) {
	if len(csMachine.Spec.Template.ID) > 0 {
		csTemplate, count, err := c.cs.Template.GetTemplateByID(csMachine.Spec.Template.ID, "all")
		if err != nil {
			return "", multierror.Append(retErr, errors.Wrapf(
				err, "could not get Template by ID %s", csMachine.Spec.Template.ID))
		} else if count != 1 {
			return "", multierror.Append(retErr, errors.Errorf(
				"expected 1 Template with UUID %s, but got %d", csMachine.Spec.Template.ID, count))
		}

		if len(csMachine.Spec.Template.Name) > 0 && csMachine.Spec.Template.Name != csTemplate.Name {
			return "", multierror.Append(retErr, errors.Errorf(
				"template name %s does not match name %s returned using UUID %s", csMachine.Spec.Template.Name, csTemplate.Name, csMachine.Spec.Template.ID))
		}
		return csMachine.Spec.Template.ID, nil
	}
	templateID, count, err := c.cs.Template.GetTemplateID(csMachine.Spec.Template.Name, "all", zoneID)
	if err != nil {
		return "", multierror.Append(retErr, errors.Wrapf(
			err, "could not get Template ID from %s", csMachine.Spec.Template.Name))
	} else if count != 1 {
		return "", multierror.Append(retErr, errors.Errorf(
			"expected 1 Template with name %s, but got %d", csMachine.Spec.Template.Name, count))
	}
	return templateID, nil
}

// GetOrCreateVMInstance CreateVMInstance will fetch or create a VM instance, and
// sets the infrastructure machine spec and status accordingly.
func (c *client) GetOrCreateVMInstance(
	csMachine *infrav1.CloudStackMachine,
	capiMachine *capiv1.Machine,
	csCluster *infrav1.CloudStackCluster,
	userData string) error {

	// Check if VM instance already exists.
	if err := c.ResolveVMInstanceDetails(csMachine); err == nil ||
		!strings.Contains(strings.ToLower(err.Error()), "no match") {
		return err
	}

	offeringID, err := c.ResolveServiceOffering(csMachine)
	if err != nil {
		return err
	}
	templateID, err := c.ResolveTemplate(csCluster, csMachine, csMachine.Status.ZoneID)
	if err != nil {
		return err
	}

	// Create VM instance.
	p := c.cs.VirtualMachine.NewDeployVirtualMachineParams(offeringID, templateID, csMachine.Status.ZoneID)
	zone := csCluster.Status.Zones[csMachine.Status.ZoneID]
	p.SetNetworkids([]string{zone.Network.ID})
	setIfNotEmpty(csMachine.Name, p.SetName)
	setIfNotEmpty(csMachine.Name, p.SetDisplayname)

	setIfNotEmpty(csMachine.Spec.SSHKey, p.SetKeypair)

	compressedAndEncodedUserData, err := CompressAndEncodeString(userData)
	if err != nil {
		return err
	}
	setIfNotEmpty(compressedAndEncodedUserData, p.SetUserdata)

	if len(csMachine.Spec.AffinityGroupIDs) > 0 {
		p.SetAffinitygroupids(csMachine.Spec.AffinityGroupIDs)
	} else if strings.ToLower(csMachine.Spec.Affinity) != "no" && csMachine.Spec.Affinity != "" {
		affinityType := AffinityGroupType
		if strings.ToLower(csMachine.Spec.Affinity) == antiAffinityValue {
			affinityType = AntiAffinityGroupType
		}
		name, err := csMachine.AffinityGroupName(capiMachine)
		if err != nil {
			return err
		}
		group := &AffinityGroup{Name: name, Type: affinityType}
		if err := c.GetOrCreateAffinityGroup(csCluster, group); err != nil {
			return err
		}
		p.SetAffinitygroupids([]string{group.ID})
	}
	setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
	setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)

	// If this VM instance is a control plane, consider setting its IP.
	_, isControlPlanceMachine := capiMachine.ObjectMeta.Labels["cluster.x-k8s.io/control-plane"]
	if isControlPlanceMachine && zone.Network.Type == NetworkTypeShared {
		// If the specified control plane endpoint is an IP address, specify the IP address of this VM instance.
		if net.ParseIP(csCluster.Spec.ControlPlaneEndpoint.Host) != nil {
			p.SetIpaddress(csCluster.Spec.ControlPlaneEndpoint.Host)
		}
	}
	if csMachine.Spec.Details != nil {
		p.SetDetails(csMachine.Spec.Details)
	}

	deployVMResp, err := c.cs.VirtualMachine.DeployVirtualMachine(p)
	if err != nil {
		// Just because an error was returned doesn't mean a (failed) VM wasn't created and will need to be dealt with.
		// Regretfully the deployVMResp may be nil, so we need to get the VM ID with a separate query, so we
		// can return it to the caller, so they can clean it up.
		listVirtualMachineParams := c.cs.VirtualMachine.NewListVirtualMachinesParams()
		listVirtualMachineParams.SetTemplateid(templateID)
		listVirtualMachineParams.SetZoneid(csMachine.Status.ZoneID)
		listVirtualMachineParams.SetNetworkid(zone.Network.ID)
		listVirtualMachineParams.SetName(csMachine.Name)
		setIfNotEmpty(csCluster.Status.DomainID, listVirtualMachineParams.SetDomainid)
		setIfNotEmpty(csCluster.Spec.Account, listVirtualMachineParams.SetAccount)
		if listVirtualMachinesResponse, err2 := c.cs.VirtualMachine.ListVirtualMachines(listVirtualMachineParams); err2 == nil && listVirtualMachinesResponse.Count > 0 {
			csMachine.Spec.InstanceID = pointer.StringPtr(listVirtualMachinesResponse.VirtualMachines[0].Id)
		}
		return err
	}
	csMachine.Spec.InstanceID = pointer.StringPtr(deployVMResp.Id)

	// Resolve uses a VM metrics request response to fill cloudstack machine status.
	// The deployment response is insufficient.
	return c.ResolveVMInstanceDetails(csMachine)

}

// DestroyVMInstance Destroys a VM instance. Assumes machine has been fetched prior and has an instance ID.
func (c *client) DestroyVMInstance(csMachine *infrav1.CloudStackMachine) error {

	// Attempt deletion regardless of machine state.
	p := c.cs.VirtualMachine.NewDestroyVirtualMachineParams(*csMachine.Spec.InstanceID)
	p.SetExpunge(true)
	if _, err := c.csAsync.VirtualMachine.DestroyVirtualMachine(p); err != nil &&
		strings.Contains(strings.ToLower(err.Error()), "unable to find uuid for id") {
		// VM doesn't exist. Success...
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("VM deletion in progress")
}
