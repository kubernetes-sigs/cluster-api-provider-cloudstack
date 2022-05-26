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
	"strings"

	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

type VMIface interface {
	GetOrCreateVMInstance(*infrav1.CloudStackMachine, *capiv1.Machine, *infrav1.CloudStackCluster, *infrav1.CloudStackZone, *infrav1.CloudStackAffinityGroup, string) error
	ResolveVMInstanceDetails(*infrav1.CloudStackMachine) error
	DestroyVMInstance(*infrav1.CloudStackMachine) error
	StartVMInstance(*infrav1.CloudStackMachine) error
	AttachISOToVMInstance(*infrav1.CloudStackMachine, *infrav1.CloudStackZone) error
}

// Set infrastructure spec and status from the CloudStack API's virtual machine metrics type.
func setMachineDataFromVMMetrics(vmResponse *cloudstack.VirtualMachinesMetric, csMachine *infrav1.CloudStackMachine) {
	csMachine.Spec.ProviderID = pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", vmResponse.Id))
	csMachine.Spec.InstanceID = pointer.StringPtr(vmResponse.Id)
	csMachine.Status.Addresses = []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: vmResponse.Ipaddress}}
	csMachine.Status.InstanceState = infrav1.InstanceState(vmResponse.State)
}

// Set infrastructure spec and status from the CloudStack API's virtual machine metrics type.
func setMachineDataFromStartVMResponse(startVMResponse *cloudstack.StartVirtualMachineResponse, csMachine *infrav1.CloudStackMachine) {
	csMachine.Spec.ProviderID = pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", startVMResponse.Id))
	csMachine.Spec.InstanceID = pointer.StringPtr(startVMResponse.Id)
	var addresses []corev1.NodeAddress
	for _, nic := range startVMResponse.Nic {
		addresses = append(addresses, corev1.NodeAddress{
			Type:    corev1.NodeInternalIP,
			Address: nic.Ipaddress})
	}
	csMachine.Status.Addresses = addresses
	csMachine.Status.InstanceState = infrav1.InstanceState(startVMResponse.State)
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

// StartVMInstance Start VM instance using csMachine.Spec.InstanceID.
func (c *client) StartVMInstance(csMachine *infrav1.CloudStackMachine) error {
	// Get VM instance ID
	instanceID := csMachine.Spec.InstanceID

	if instanceID == nil && len(csMachine.Name) > 0 {
		vmResp, count, err := c.cs.VirtualMachine.GetVirtualMachinesMetricByName(csMachine.Name) // add opts usage
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match") {
			return err
		} else if count > 1 {
			return fmt.Errorf("found more than one VM Instance with name %s", csMachine.Name)
		} else if err == nil {
			instanceID = pointer.StringPtr(vmResp.Id)
		}
	}
	// Attempt to fetch by ID.
	if instanceID != nil {
		p := c.cs.VirtualMachine.NewStartVirtualMachineParams(*instanceID)
		startVMResp, err := c.cs.VirtualMachine.StartVirtualMachine(p)
		if err != nil {
			return err
		}
		setMachineDataFromStartVMResponse(startVMResp, csMachine)
		return nil
	}

	return errors.New("vm instance ID empty")
}

// AttachISOToVMInstance Attach ISO to VM instance using csMachine.Spec.InstanceID.
func (c *client) AttachISOToVMInstance(csMachine *infrav1.CloudStackMachine, zone *infrav1.CloudStackZone) error {
	// Get VM instance ID
	instanceID := csMachine.Spec.InstanceID

	if instanceID == nil && len(csMachine.Name) > 0 {
		vmResp, count, err := c.cs.VirtualMachine.GetVirtualMachinesMetricByName(csMachine.Name) // add opts usage
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match") {
			return err
		} else if count > 1 {
			return fmt.Errorf("found more than one VM Instance with name %s", csMachine.Name)
		} else if err == nil {
			instanceID = pointer.StringPtr(vmResp.Id)
		}
	}
	// Attempt to fetch by ID.
	if instanceID != nil {
		isoID, err := c.resolveISO(csMachine, zone)
		if err != nil {
			return err
		}
		p := c.cs.ISO.NewAttachIsoParams(isoID, *instanceID)
		_, err = c.cs.ISO.AttachIso(p)
		return err
	}

	return errors.New("vm instance ID empty")
}

// resolveISO get ISO id if only name configured, verify name and id match if both configured
func (c *client) resolveISO(csMachine *infrav1.CloudStackMachine, zone *infrav1.CloudStackZone) (isoID string, err error) {
	if len(csMachine.Spec.ISOAttachment.ID) > 0 {
		iso, count, err := c.cs.ISO.GetIsoByID(csMachine.Spec.ISOAttachment.ID)
		if err != nil {
			return "", multierror.Append(err, errors.Wrapf(
				err, "could not get ISO by ID %s", csMachine.Spec.ISOAttachment.ID))
		} else if count != 1 {
			return "", multierror.Append(err, errors.Errorf(
				"expected 1 ISO with UUID %s, but got %d", csMachine.Spec.ISOAttachment.ID, count))
		}

		if len(csMachine.Spec.ISOAttachment.Name) > 0 && csMachine.Spec.ISOAttachment.Name != iso.Name {
			return "", multierror.Append(err, errors.Errorf(
				"iso name %s does not match name %s returned using UUID %s", csMachine.Spec.ISOAttachment.Name, iso.Name, csMachine.Spec.ISOAttachment.ID))
		}
		return csMachine.Spec.ISOAttachment.ID, nil
	}
	isoID, count, err := c.cs.ISO.GetIsoID(csMachine.Spec.ISOAttachment.Name, "", zone.Spec.ID)
	if err != nil {
		return "", multierror.Append(err, errors.Wrapf(
			err, "could not get ISO by Name %s", csMachine.Spec.ISOAttachment.Name))
	} else if count != 1 {
		return "", multierror.Append(err, errors.Errorf(
			"expected 1 ISO with Name %s, but got %d", csMachine.Spec.ISOAttachment.Name, count))
	}

	return isoID, nil
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
		csTemplate, count, err := c.cs.Template.GetTemplateByID(csMachine.Spec.Template.ID, "executable")
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
	templateID, count, err := c.cs.Template.GetTemplateID(csMachine.Spec.Template.Name, "executable", zoneID)
	if err != nil {
		return "", multierror.Append(retErr, errors.Wrapf(
			err, "could not get Template ID from %s", csMachine.Spec.Template.Name))
	} else if count != 1 {
		return "", multierror.Append(retErr, errors.Errorf(
			"expected 1 Template with name %s, but got %d", csMachine.Spec.Template.Name, count))
	}
	return templateID, nil
}

// ResolveDiskOffering Retrieves diskOffering by using disk offering ID if ID is provided and confirm returned
// disk offering name matches name provided in spec.
// If disk offering ID is not provided, the disk offering name is used to retrieve disk offering ID.
func (c *client) ResolveDiskOffering(csMachine *infrav1.CloudStackMachine) (diskOfferingID string, retErr error) {
	diskOfferingID = csMachine.Spec.DiskOffering.ID
	if len(csMachine.Spec.DiskOffering.Name) > 0 {
		diskID, count, err := c.cs.DiskOffering.GetDiskOfferingID(csMachine.Spec.DiskOffering.Name)
		if err != nil {
			return "", multierror.Append(retErr, errors.Wrapf(
				err, "could not get DiskOffering ID from %s", csMachine.Spec.DiskOffering.Name))
		} else if count != 1 {
			return "", multierror.Append(retErr, errors.Errorf(
				"expected 1 DiskOffering with name %s, but got %d", csMachine.Spec.DiskOffering.Name, count))
		} else if len(csMachine.Spec.DiskOffering.ID) > 0 && diskID != csMachine.Spec.DiskOffering.ID {
			return "", multierror.Append(retErr, errors.Errorf(
				"diskOffering ID %s does not match ID %s returned using name %s",
				csMachine.Spec.DiskOffering.ID, diskID, csMachine.Spec.DiskOffering.Name))
		} else if len(diskID) == 0 {
			return "", multierror.Append(retErr, errors.Errorf(
				"empty diskOffering ID %s returned using name %s",
				diskID, csMachine.Spec.DiskOffering.Name))
		}
		diskOfferingID = diskID
	}
	if len(diskOfferingID) == 0 {
		return "", nil
	}

	return verifyDiskoffering(csMachine, c, diskOfferingID, retErr)
}

func verifyDiskoffering(csMachine *infrav1.CloudStackMachine, c *client, diskOfferingID string, retErr error) (string, error) {
	csDiskOffering, count, err := c.cs.DiskOffering.GetDiskOfferingByID(diskOfferingID)
	if err != nil {
		return "", multierror.Append(retErr, errors.Wrapf(
			err, "could not get DiskOffering by ID %s", diskOfferingID))
	} else if count != 1 {
		return "", multierror.Append(retErr, errors.Errorf(
			"expected 1 DiskOffering with UUID %s, but got %d", diskOfferingID, count))
	}

	if csDiskOffering.Iscustomized && csMachine.Spec.DiskOffering.CustomSize == 0 {
		return "", multierror.Append(retErr, errors.Errorf(
			"diskOffering with UUID %s is customized, disk size can not be 0 GB",
			diskOfferingID))
	}

	if !csDiskOffering.Iscustomized && csMachine.Spec.DiskOffering.CustomSize > 0 {
		return "", multierror.Append(retErr, errors.Errorf(
			"diskOffering with UUID %s is not customized, disk size can not be specified",
			diskOfferingID))
	}
	return diskOfferingID, nil
}

// GetOrCreateVMInstance CreateVMInstance will fetch or create a VM instance, and
// sets the infrastructure machine spec and status accordingly.
func (c *client) GetOrCreateVMInstance(
	csMachine *infrav1.CloudStackMachine,
	capiMachine *capiv1.Machine,
	csCluster *infrav1.CloudStackCluster,
	zone *infrav1.CloudStackZone,
	affinity *infrav1.CloudStackAffinityGroup,
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
	diskOfferingID, err := c.ResolveDiskOffering(csMachine)
	if err != nil {
		return err
	}

	// Create VM instance.
	p := c.cs.VirtualMachine.NewDeployVirtualMachineParams(offeringID, templateID, csMachine.Status.ZoneID)
	p.SetNetworkids([]string{zone.Spec.Network.ID})
	setIfNotEmpty(csMachine.Name, p.SetName)
	setIfNotEmpty(csMachine.Name, p.SetDisplayname)
	setIfNotEmpty(diskOfferingID, p.SetDiskofferingid)
	setIntIfPositive(csMachine.Spec.DiskOffering.CustomSize, p.SetSize)
	p.SetStartvm(false)

	setIfNotEmpty(csMachine.Spec.SSHKey, p.SetKeypair)

	compressedAndEncodedUserData, err := CompressAndEncodeString(userData)
	if err != nil {
		return err
	}
	setIfNotEmpty(compressedAndEncodedUserData, p.SetUserdata)

	if len(csMachine.Spec.AffinityGroupIDs) > 0 {
		p.SetAffinitygroupids(csMachine.Spec.AffinityGroupIDs)
	} else if strings.ToLower(csMachine.Spec.Affinity) != "no" && csMachine.Spec.Affinity != "" {
		p.SetAffinitygroupids([]string{affinity.Spec.ID})
		if err != nil {
			return err
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
		listVirtualMachineParams.SetNetworkid(zone.Spec.Network.ID)
		listVirtualMachineParams.SetName(csMachine.Name)
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
	p := c.csAsync.VirtualMachine.NewDestroyVirtualMachineParams(*csMachine.Spec.InstanceID)
	volIDs, err := c.listVMInstanceVolumeIDs(*csMachine.Spec.InstanceID)
	if err != nil {
		return err
	}
	p.SetExpunge(true)
	setArrayIfNotEmpty(volIDs, p.SetVolumeids)
	if _, err := c.csAsync.VirtualMachine.DestroyVirtualMachine(p); err != nil &&
		strings.Contains(strings.ToLower(err.Error()), "unable to find uuid for id") {
		// VM doesn't exist. Success...
		return nil
	} else if err != nil {
		return err
	}

	if err := c.ResolveVMInstanceDetails(csMachine); err == nil && (csMachine.Status.InstanceState == "Expunging" ||
		csMachine.Status.InstanceState == "Expunged") {
		// VM is stopped and getting expunged.  So the desired state is getting satisfied.  Let's move on.
		return nil
	} else if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no match found") {
			// VM doesn't exist.  So the desired state is in effect.  Our work is done here.
			return nil
		}
		return err
	}

	return errors.New("VM deletion in progress")
}

func (c *client) listVMInstanceVolumeIDs(instanceID string) ([]string, error) {
	p := c.cs.Volume.NewListVolumesParams()
	p.SetVirtualmachineid(instanceID)

	listVOLResp, err := c.csAsync.Volume.ListVolumes(p)
	if err != nil {
		return nil, err
	}

	var ret []string
	for _, vol := range listVOLResp.Volumes {
		ret = append(ret, vol.Id)
	}

	return ret, nil

}
