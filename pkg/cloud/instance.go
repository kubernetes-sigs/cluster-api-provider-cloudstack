/*
Copyright 2022.

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
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

const AntiAffinityValue = "anti"

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

func (c *client) ResolveServiceOffering(csMachine *infrav1.CloudStackMachine) (offeringID string, retErr error) {
	offeringID, count, err := c.cs.ServiceOffering.GetServiceOfferingID(csMachine.Spec.Offering)
	if err != nil {
		retErr = multierror.Append(retErr, errors.Wrapf(
			err, "Could not get Service Offering ID from %s.", csMachine.Spec.Offering))
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"Expected 1 Service Offering with name %s, but got %d.", csMachine.Spec.Offering, count))
	}

	if retErr != nil {
		if _, count, err := c.cs.ServiceOffering.GetServiceOfferingByID(csMachine.Spec.Offering); err != nil {
			return "", multierror.Append(retErr, errors.Wrapf(
				err, "Could not get Service Offering by ID %s.", csMachine.Spec.Offering))
		} else if count != 1 {
			return "", multierror.Append(retErr, errors.Errorf(
				"Expected 1 Service Offering with UUID %s, but got %d.", csMachine.Spec.Offering, count))
		} else {
			offeringID = csMachine.Spec.Offering
		}
	}

	return offeringID, nil
}

func (c *client) ResolveTemplate(csCluster *infrav1.CloudStackCluster, csMachine *infrav1.CloudStackMachine) (templateID string, retErr error) {
	templateID, count, err := c.cs.Template.GetTemplateID(csMachine.Spec.Template, "all", csCluster.Status.ZoneID)
	if err != nil {
		retErr = multierror.Append(retErr, errors.Wrapf(
			err, "Could not get Template ID from %s.", csMachine.Spec.Template))
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"Expected 1 Template with name %s, but got %d.", csMachine.Spec.Template, count))
	}

	if retErr != nil {
		if _, count, err := c.cs.Template.GetTemplateByID(csMachine.Spec.Template, "all"); err != nil {
			return "", multierror.Append(retErr, errors.Wrapf(
				err, "Could not get Template by ID %s.", csMachine.Spec.Template))
		} else if count != 1 {
			return "", multierror.Append(retErr, errors.Errorf(
				"Expected 1 Template with UUID %s, but got %d.", csMachine.Spec.Template, count))
		} else {
			templateID = csMachine.Spec.Template
		}
	}

	return templateID, nil
}

// GetOrCreateVMInstance CreateVMInstance will fetch or create a VM instance, and
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
	offeringID, err := c.ResolveServiceOffering(csMachine)
	if err != nil {
		return err
	}

	templateID, err := c.ResolveTemplate(csCluster, csMachine)
	if err != nil {
		return err
	}

	// Create VM instance.
	p := c.cs.VirtualMachine.NewDeployVirtualMachineParams(offeringID, templateID, csCluster.Status.ZoneID)
	p.SetNetworkids([]string{csCluster.Status.NetworkID})
	setIfNotEmpty(csMachine.Name, p.SetName)
	setIfNotEmpty(csMachine.Name, p.SetDisplayname)
	setIfNotEmpty(csMachine.Spec.SSHKey, p.SetKeypair)

	if compressedAndEncodedUserData, err := CompressAndEncodeString(userData); err != nil {
		return err
	} else {
		setIfNotEmpty(compressedAndEncodedUserData, p.SetUserdata)
	}

	if len(csMachine.Spec.AffinityGroupIds) > 0 {
		p.SetAffinitygroupids(csMachine.Spec.AffinityGroupIds)
	} else if strings.ToLower(csMachine.Spec.Affinity) != "no" && csMachine.Spec.Affinity != "" {
		ownerRef := csCtrlrUtils.GetManagementOwnerRef(machine)
		if ownerRef == nil {
			return errors.Errorf("Could not find management owner reference for %s/%s",
				csMachine.Namespace, csMachine.Name)
		}
		affinityType := AffinityGroupType
		if strings.ToLower(csMachine.Spec.Affinity) == AntiAffinityValue {
			affinityType = AntiAffinityGroupType
		}
		name := fmt.Sprintf("%sAffinity-%s-%s", affinityType, csCluster.ClusterName, ownerRef.UID)
		group := &AffinityGroup{Name: name, Type: affinityType}
		if err := c.GetOrCreateAffinityGroup(csCluster, group); err != nil {
			return err
		}
		p.SetAffinitygroupids([]string{group.Id})
	}
	setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
	setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)

	// If this VM instance is a control plane, consider setting it's IP.
	_, isControlPlanceMachine := machine.ObjectMeta.Labels["cluster.x-k8s.io/control-plane"]
	if isControlPlanceMachine && csCluster.Status.NetworkType == NetworkTypeShared {
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
		return err
	}
	csMachine.Spec.InstanceID = pointer.StringPtr(deployVMResp.Id)

	// Resolve uses a VM metrics request response to fill cloudstack machine status.
	// The deployment response is insufficient.
	return c.ResolveVMInstanceDetails(csMachine)

}

// DestroyVMInstance Destroy a VM instance. Assumes machine has been fetched prior and has an instance ID.
func (c *client) DestroyVMInstance(csMachine *infrav1.CloudStackMachine) error {

	p := c.cs.VirtualMachine.NewDestroyVirtualMachineParams(*csMachine.Spec.InstanceID)
	p.SetExpunge(true)
	_, err := c.cs.VirtualMachine.DestroyVirtualMachine(p)
	if err != nil && strings.Contains(err.Error(), "Unable to find UUID for id ") {
		// VM doesn't exist.  So the desired state is in effect.  Our work is done here.
		return nil
	}
	return err
}
