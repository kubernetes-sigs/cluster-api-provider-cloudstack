package cloud

import (
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
	"k8s.io/utils/pointer"
)

// Set infrastructure spec and status from the CloudStack API's virtual machine metrics type.
func setMachineDataFromVMMetrics(vmResponse *cloudstack.VirtualMachinesMetric, csMachine *infrav1.CloudStackMachine) {
	csMachine.Spec.ProviderID = pointer.StringPtr(fmt.Sprintf("cloudstack:///%s", vmResponse.Id))
	csMachine.Spec.InstanceID = pointer.StringPtr(vmResponse.Id)
	csMachine.Status.Addresses = []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: vmResponse.Ipaddress}}
}

// Fetch retrieves a VM instance by csMachine.Spec.InstanceID or csMachine.Name, and
// sets infrastructure machine spec and status if VM instance is found.
func FetchVMInstance(cs *cloudstack.CloudStackClient, csMachine *infrav1.CloudStackMachine) error {
	// Attempt to fetch by ID.
	if csMachine.Spec.InstanceID != nil {
		vmResp, count, err := cs.VirtualMachine.GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match found") {
			return err
		} else if count > 1 {
			return errors.New(fmt.Sprintf("Found more than one VM Instance with ID %s.", *csMachine.Spec.InstanceID))
		} else if err == nil {
			setMachineDataFromVMMetrics(vmResp, csMachine)
			return nil
		}
	}

	// Attempt fetch by name.
	if csMachine.Name != "" {
		vmResp, count, err := cs.VirtualMachine.GetVirtualMachinesMetricByName(csMachine.Name) // add opts usage
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match found") {
			return err
		} else if count > 1 {
			return errors.New(fmt.Sprintf("Found more than one VM Instance with name %s.", csMachine.Name))
		} else if err == nil {
			setMachineDataFromVMMetrics(vmResp, csMachine)
			return nil
		}
	}
	return errors.New("machine not found")
}

// CreateVMInstance will fetch or create a VM instance, and
// sets the infrastructure machine spec and status accordingly.
func CreateVMInstance(
	cs *cloudstack.CloudStackClient,
	csMachine *infrav1.CloudStackMachine,
	csCluster *infrav1.CloudStackCluster) error {

	// Check if VM instance already exists.
	if err := FetchVMInstance(cs, csMachine); err == nil || !strings.Contains(err.Error(), "machine not found") {
		return err
	}

	// Get machine offering ID from name.
	offeringID, count, err := cs.ServiceOffering.GetServiceOfferingID(csMachine.Spec.Offering)
	if err != nil {
		return err
	} else if count != 1 {
		return errors.New(fmt.Sprintf(
			"Did not find exactly one machine offering with the name %s", csMachine.Spec.Offering))
	}

	// Get template ID from name.
	templateID, count, err := cs.Template.GetTemplateID(csMachine.Spec.Template, "all", csCluster.Status.ZoneID)
	if err != nil {
		return err
	} else if count != 1 {
		return errors.New(fmt.Sprintf(
			"Did not find exactly one template with the name %s", csMachine.Spec.Template))
	}

	// Create VM instance.
	p := cs.VirtualMachine.NewDeployVirtualMachineParams(offeringID, templateID, csCluster.Status.ZoneID)
	setIfNotEmpty(csMachine.Name, p.SetName)
	setIfNotEmpty(csMachine.Name, p.SetDisplayname)
	setIfNotEmpty(csMachine.Spec.SSHKey, p.SetKeypair)

	deployVMResp, err := cs.VirtualMachine.DeployVirtualMachine(p)
	if err != nil {
		return err
	}
	csMachine.Spec.InstanceID = pointer.StringPtr(deployVMResp.Id)

	// Fetch will now find the VM and fill csMachine.
	// Fetch uses a VM metrics request response to fill cloudstack machine status.
	// The deployment response is insufficient.
	if err = FetchVMInstance(cs, csMachine); err == nil {
		csMachine.Status.Ready = true
	}
	return err
}

// Destroy a VM instane. Assumes machine has been fetched prior and has an instance ID.
func DestroyVMInstance(cs *cloudstack.CloudStackClient, csMachine *infrav1.CloudStackMachine) error {

	p := cs.VirtualMachine.NewDestroyVirtualMachineParams(*csMachine.Spec.InstanceID)
	p.SetExpunge(true)
	_, err := cs.VirtualMachine.DestroyVirtualMachine(p)

	return err
}
