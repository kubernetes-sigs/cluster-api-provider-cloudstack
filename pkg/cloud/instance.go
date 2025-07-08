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

package cloud

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"

	netpkg "net"
)

type VMIface interface {
	GetOrCreateVMInstance(*infrav1.CloudStackMachine, *clusterv1.Machine, *infrav1.CloudStackCluster, *infrav1.CloudStackFailureDomain, *infrav1.CloudStackAffinityGroup, string) error
	ResolveVMInstanceDetails(*infrav1.CloudStackMachine) error
	DestroyVMInstance(*infrav1.CloudStackMachine) error
}

// Set infrastructure spec and status from the CloudStack API's virtual machine metrics type.
func setMachineDataFromVMMetrics(vmResponse *cloudstack.VirtualMachinesMetric, csMachine *infrav1.CloudStackMachine) {
	csMachine.Spec.ProviderID = ptr.To(fmt.Sprintf("cloudstack:///%s", vmResponse.Id))
	// InstanceID is later used as required parameter to destroy VM.
	csMachine.Spec.InstanceID = ptr.To(vmResponse.Id)
	csMachine.Status.Addresses = []corev1.NodeAddress{}
	for _, nic := range vmResponse.Nic {
		if nic.Ipaddress != "" {
			csMachine.Status.Addresses = append(csMachine.Status.Addresses, corev1.NodeAddress{
				Type:    corev1.NodeInternalIP,
				Address: nic.Ipaddress,
			})
		}
	}
	newInstanceState := vmResponse.State
	if newInstanceState != csMachine.Status.InstanceState || (newInstanceState != "" && csMachine.Status.InstanceStateLastUpdated.IsZero()) {
		csMachine.Status.InstanceState = newInstanceState
		csMachine.Status.InstanceStateLastUpdated = metav1.Now()
	}
}

// ResolveVMInstanceDetails Retrieves VM instance details by csMachine.Spec.InstanceID or csMachine.Name, and
// sets infrastructure machine spec and status if VM instance is found.
func (c *client) ResolveVMInstanceDetails(csMachine *infrav1.CloudStackMachine) error {
	// Attempt to fetch by ID.
	if csMachine.Spec.InstanceID != nil {
		vmResp, count, err := c.cs.VirtualMachine.GetVirtualMachinesMetricByID(*csMachine.Spec.InstanceID, cloudstack.WithProject(c.user.Project.ID))
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match found") {
			c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
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
		vmResp, count, err := c.cs.VirtualMachine.GetVirtualMachinesMetricByName(csMachine.Name, cloudstack.WithProject(c.user.Project.ID))
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match") {
			c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
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

func (c *client) ResolveServiceOffering(csMachine *infrav1.CloudStackMachine, zoneID string) (offering cloudstack.ServiceOffering, retErr error) {
	if len(csMachine.Spec.Offering.ID) > 0 {
		csOffering, count, err := c.cs.ServiceOffering.GetServiceOfferingByID(csMachine.Spec.Offering.ID, cloudstack.WithProject(c.user.Project.ID))
		if err != nil {
			c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
			return cloudstack.ServiceOffering{}, errors.Wrapf(
				err, "could not get Service Offering by ID %s", csMachine.Spec.Offering.ID)
		} else if count != 1 {
			return *csOffering, errors.Errorf(
				"expected 1 Service Offering with UUID %s, but got %d", csMachine.Spec.Offering.ID, count)
		}

		if len(csMachine.Spec.Offering.Name) > 0 && csMachine.Spec.Offering.Name != csOffering.Name {
			return *csOffering, errors.Errorf(
				"offering name %s does not match name %s returned using UUID %s", csMachine.Spec.Offering.Name, csOffering.Name, csMachine.Spec.Offering.ID)
		}
		return *csOffering, nil
	}
	csOffering, count, err := c.cs.ServiceOffering.GetServiceOfferingByName(csMachine.Spec.Offering.Name, cloudstack.WithZone(zoneID), cloudstack.WithProject(c.user.Project.ID))
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return cloudstack.ServiceOffering{}, errors.Wrapf(
			err, "could not get Service Offering ID from %s in zone %s", csMachine.Spec.Offering.Name, zoneID)
	} else if count != 1 {
		return *csOffering, errors.Errorf(
			"expected 1 Service Offering with name %s in zone %s, but got %d", csMachine.Spec.Offering.Name, zoneID, count)
	}
	return *csOffering, nil
}

func (c *client) ResolveTemplate(
	csMachine *infrav1.CloudStackMachine,
	zoneID string,
) (templateID string, retErr error) {
	if len(csMachine.Spec.Template.ID) > 0 {
		csTemplate, count, err := c.cs.Template.GetTemplateByID(csMachine.Spec.Template.ID, "executable", cloudstack.WithProject(c.user.Project.ID))
		if err != nil {
			c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
			return "", errors.Wrapf(err, "could not get Template by ID %s", csMachine.Spec.Template.ID)
		} else if count != 1 {
			return "", errors.Errorf(
				"expected 1 Template with UUID %s, but got %d", csMachine.Spec.Template.ID, count)
		}

		if len(csMachine.Spec.Template.Name) > 0 && csMachine.Spec.Template.Name != csTemplate.Name {
			return "", errors.Errorf(
				"template name %s does not match name %s returned using UUID %s", csMachine.Spec.Template.Name, csTemplate.Name, csMachine.Spec.Template.ID)
		}
		return csMachine.Spec.Template.ID, nil
	}
	templateID, count, err := c.cs.Template.GetTemplateID(csMachine.Spec.Template.Name, "executable", zoneID, cloudstack.WithProject(c.user.Project.ID))
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return "", errors.Wrapf(err, "could not get Template ID from %s", csMachine.Spec.Template.Name)
	} else if count != 1 {
		return "", errors.Errorf(
			"expected 1 Template with name %s, but got %d", csMachine.Spec.Template.Name, count)
	}
	return templateID, nil
}

// ResolveDiskOffering Retrieves diskOffering by using disk offering ID if ID is provided and confirm returned
// disk offering name matches name provided in spec.
// If disk offering ID is not provided, the disk offering name is used to retrieve disk offering ID.
func (c *client) ResolveDiskOffering(csMachine *infrav1.CloudStackMachine, zoneID string) (diskOfferingID string, retErr error) {
	diskOfferingID = csMachine.Spec.DiskOffering.ID
	if len(csMachine.Spec.DiskOffering.Name) > 0 {
		diskID, count, err := c.cs.DiskOffering.GetDiskOfferingID(csMachine.Spec.DiskOffering.Name, cloudstack.WithZone(zoneID), cloudstack.WithProject(c.user.Project.ID))
		if err != nil {
			c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
			return "", errors.Wrapf(err, "could not get DiskOffering ID from %s", csMachine.Spec.DiskOffering.Name)
		} else if count != 1 {
			return "", errors.Errorf(
				"expected 1 DiskOffering with name %s in zone %s, but got %d", csMachine.Spec.DiskOffering.Name, zoneID, count)
		} else if len(csMachine.Spec.DiskOffering.ID) > 0 && diskID != csMachine.Spec.DiskOffering.ID {
			return "", errors.Errorf(
				"diskOffering ID %s does not match ID %s returned using name %s in zone %s",
				csMachine.Spec.DiskOffering.ID, diskID, csMachine.Spec.DiskOffering.Name, zoneID)
		} else if len(diskID) == 0 {
			return "", errors.Errorf(
				"empty diskOffering ID %s returned using name %s in zone %s",
				diskID, csMachine.Spec.DiskOffering.Name, zoneID)
		}
		diskOfferingID = diskID
	}
	if len(diskOfferingID) == 0 {
		return "", nil
	}

	return verifyDiskoffering(csMachine, c, diskOfferingID)
}

func verifyDiskoffering(csMachine *infrav1.CloudStackMachine, c *client, diskOfferingID string) (string, error) {
	csDiskOffering, count, err := c.cs.DiskOffering.GetDiskOfferingByID(diskOfferingID, cloudstack.WithProject(c.user.Project.ID))
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return "", errors.Wrapf(err, "could not get DiskOffering by ID %s", diskOfferingID)
	} else if count != 1 {
		return "", errors.Errorf(
			"expected 1 DiskOffering with UUID %s, but got %d", diskOfferingID, count)
	}

	if csDiskOffering.Iscustomized && csMachine.Spec.DiskOffering.CustomSize == 0 {
		return "", errors.Errorf(
			"diskOffering with UUID %s is customized, disk size can not be 0 GB",
			diskOfferingID)
	}

	if !csDiskOffering.Iscustomized && csMachine.Spec.DiskOffering.CustomSize > 0 {
		return "", errors.Errorf(
			"diskOffering with UUID %s is not customized, disk size can not be specified",
			diskOfferingID)
	}
	return diskOfferingID, nil
}

// CheckAccountLimits Checks the account's limit of VM, CPU & Memory
func (c *client) CheckAccountLimits(offering *cloudstack.ServiceOffering) error {
	if c.user.Account.CPUAvailable != "Unlimited" {
		cpuAvailable, err := strconv.ParseInt(c.user.Account.CPUAvailable, 10, 0)
		if err == nil && int64(offering.Cpunumber) > cpuAvailable {
			return fmt.Errorf("CPU available (%d) in account can't fulfil the requirement: %d", cpuAvailable, offering.Cpunumber)
		}
	}

	if c.user.Account.MemoryAvailable != "Unlimited" {
		memoryAvailable, err := strconv.ParseInt(c.user.Account.MemoryAvailable, 10, 0)
		if err == nil && int64(offering.Memory) > memoryAvailable {
			return fmt.Errorf("memory available (%d) in account can't fulfil the requirement: %d", memoryAvailable, offering.Memory)
		}
	}

	if c.user.Account.VMAvailable != "Unlimited" {
		vmAvailable, err := strconv.ParseInt(c.user.Account.VMAvailable, 10, 0)
		if err == nil && vmAvailable < 1 {
			return fmt.Errorf("VM Limit in account has reached it's maximum value")
		}
	}
	return nil
}

// CheckDomainLimits Checks the domain's limit of VM, CPU & Memory
func (c *client) CheckDomainLimits(offering *cloudstack.ServiceOffering) error {
	if c.user.Account.Domain.CPUAvailable != "Unlimited" {
		cpuAvailable, err := strconv.ParseInt(c.user.Account.Domain.CPUAvailable, 10, 0)
		if err == nil && int64(offering.Cpunumber) > cpuAvailable {
			return fmt.Errorf("CPU available (%d) in domain can't fulfil the requirement: %d", cpuAvailable, offering.Cpunumber)
		}
	}

	if c.user.Account.Domain.MemoryAvailable != "Unlimited" {
		memoryAvailable, err := strconv.ParseInt(c.user.Account.Domain.MemoryAvailable, 10, 0)
		if err == nil && int64(offering.Memory) > memoryAvailable {
			return fmt.Errorf("memory available (%d) in domain can't fulfil the requirement: %d", memoryAvailable, offering.Memory)
		}
	}

	if c.user.Account.Domain.VMAvailable != "Unlimited" {
		vmAvailable, err := strconv.ParseInt(c.user.Account.Domain.VMAvailable, 10, 0)
		if err == nil && vmAvailable < 1 {
			return fmt.Errorf("VM Limit in domain has reached it's maximum value")
		}
	}
	return nil
}

// CheckProjectLimits Checks the project's limit of VM, CPU & Memory
func (c *client) CheckProjectLimits(offering *cloudstack.ServiceOffering) error {
	if c.user.Project.ID == "" {
		return nil
	}

	if c.user.Project.CPUAvailable != "Unlimited" {
		cpuAvailable, err := strconv.ParseInt(c.user.Project.CPUAvailable, 10, 0)
		if err == nil && int64(offering.Cpunumber) > cpuAvailable {
			return fmt.Errorf("CPU available (%d) in project can't fulfil the requirement: %d", cpuAvailable, offering.Cpunumber)
		}
	}

	if c.user.Project.MemoryAvailable != "Unlimited" {
		memoryAvailable, err := strconv.ParseInt(c.user.Project.MemoryAvailable, 10, 0)
		if err == nil && int64(offering.Memory) > memoryAvailable {
			return fmt.Errorf("memory available (%d) in project can't fulfil the requirement: %d", memoryAvailable, offering.Memory)
		}
	}

	if c.user.Project.VMAvailable != "Unlimited" {
		vmAvailable, err := strconv.ParseInt(c.user.Project.VMAvailable, 10, 0)
		if err == nil && vmAvailable < 1 {
			return fmt.Errorf("VM Limit in project has reached it's maximum value")
		}
	}
	return nil
}

// CheckLimits will check the account & domain limits
func (c *client) CheckLimits(
	offering *cloudstack.ServiceOffering,
) error {
	err := c.CheckAccountLimits(offering)
	if err != nil {
		return err
	}

	err = c.CheckDomainLimits(offering)
	if err != nil {
		return err
	}

	err = c.CheckProjectLimits(offering)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) isIpAvailableInNetwork(ip, networkID string) (bool, error) {
	params := c.cs.Address.NewListPublicIpAddressesParams()
	params.SetNetworkid(networkID)
	params.SetIpaddress(ip)
	params.SetAllocatedonly(false)
	params.SetForvirtualnetwork(false)
	params.SetListall(true)

	resp, err := c.cs.Address.ListPublicIpAddresses(params)
	if err != nil {
		return false, errors.Wrapf(err, "failed to list public IP addresses for network %q", networkID)
	}

	for _, addr := range resp.PublicIpAddresses {
		if addr.State == "Free" {
			return true, nil
		}
	}

	return false, nil
}

func (c *client) hasFreeIPInNetwork(resolvedNet *cloudstack.Network) (bool, error) {
	params := c.cs.Address.NewListPublicIpAddressesParams()
	params.SetNetworkid(resolvedNet.Id)
	params.SetAllocatedonly(false)
	params.SetForvirtualnetwork(false)
	params.SetListall(true)

	resp, err := c.cs.Address.ListPublicIpAddresses(params)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check free IPs for network %q", resolvedNet.Id)
	}

	for _, addr := range resp.PublicIpAddresses {
		if addr.State == "Free" {
			return true, nil
		}
	}

	return false, nil
}

func (c *client) buildStaticIPEntry(ip, networkID string, resolvedNet *cloudstack.Network) (map[string]string, error) {
	if err := c.validateIPInCIDR(ip, resolvedNet, networkID); err != nil {
		return nil, err
	}

	if resolvedNet.Type == "Shared" {
		isAvailable, err := c.isIpAvailableInNetwork(ip, networkID)
		if err != nil {
			return nil, err
		}
		if !isAvailable {
			return nil, errors.Errorf("IP %q is already allocated in network %q or out of range", ip, networkID)
		}
	}

	return map[string]string{
		"networkid": networkID,
		"ip":        ip,
	}, nil
}

func (c *client) buildDynamicIPEntry(resolvedNet *cloudstack.Network) (map[string]string, error) {
	if resolvedNet.Type == "Shared" {
		freeIPExists, err := c.hasFreeIPInNetwork(resolvedNet)
		if err != nil {
			return nil, err
		}
		if !freeIPExists {
			return nil, errors.Errorf("no free IPs available in network %q", resolvedNet.Id)
		}
	}

	return map[string]string{
		"networkid": resolvedNet.Id,
	}, nil
}

func (c *client) resolveNetworkByName(name string) (*cloudstack.Network, error) {
	net, count, err := c.cs.Network.GetNetworkByName(name, cloudstack.WithProject(c.user.Project.ID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to look up network %q", name)
	}
	if count != 1 {
		return nil, errors.Errorf("expected 1 network named %q, but got %d", name, count)
	}
	return net, nil
}

func (c *client) buildIPToNetworkList(csMachine *infrav1.CloudStackMachine) ([]map[string]string, error) {
	var ipToNetworkList []map[string]string

	for _, net := range csMachine.Spec.Networks {
		networkID, resolvedNet, err := c.resolveNetworkReference(net)
		if err != nil {
			return nil, err
		}

		var entry map[string]string
		if net.IP != "" {
			entry, err = c.buildStaticIPEntry(net.IP, networkID, resolvedNet)
			if err != nil {
				return nil, err
			}
		} else {
			entry, err = c.buildDynamicIPEntry(resolvedNet)
			if err != nil {
				return nil, err
			}
		}

		ipToNetworkList = append(ipToNetworkList, entry)
	}

	return ipToNetworkList, nil
}

func (c *client) resolveNetworkReference(net infrav1.NetworkSpec) (string, *cloudstack.Network, error) {
	if net.ID == "" {
		resolvedNet, err := c.resolveNetworkByName(net.Name)
		if err != nil {
			return "", nil, err
		}
		return resolvedNet.Id, resolvedNet, nil
	}

	resolvedNet, _, err := c.cs.Network.GetNetworkByID(net.ID, cloudstack.WithProject(c.user.Project.ID))
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to get network %q by ID", net.ID)
	}
	return net.ID, resolvedNet, nil
}

func (c *client) validateIPInCIDR(ipStr string, net *cloudstack.Network, netID string) error {
	if net == nil {
		return errors.Errorf("network details not found for validation")
	}

	ip := netpkg.ParseIP(ipStr)
	if ip == nil {
		return errors.Errorf("invalid IP address %q", ipStr)
	}

	_, cidr, err := netpkg.ParseCIDR(net.Cidr)
	if err != nil {
		return errors.Wrapf(err, "invalid CIDR %q for network %q", net.Cidr, netID)
	}

	if !cidr.Contains(ip) {
		return errors.Errorf("IP %q is not within network CIDR %q", ipStr, net.Cidr)
	}

	return nil
}

// DeployVM will create a VM instance,
// and sets the infrastructure machine spec and status accordingly.
func (c *client) DeployVM(
	csMachine *infrav1.CloudStackMachine,
	capiMachine *clusterv1.Machine,
	fd *infrav1.CloudStackFailureDomain,
	affinity *infrav1.CloudStackAffinityGroup,
	offering *cloudstack.ServiceOffering,
	userData string,
) error {
	templateID, err := c.ResolveTemplate(csMachine, fd.Spec.Zone.ID)
	if err != nil {
		return err
	}
	diskOfferingID, err := c.ResolveDiskOffering(csMachine, fd.Spec.Zone.ID)
	if err != nil {
		return err
	}

	p := c.cs.VirtualMachine.NewDeployVirtualMachineParams(offering.Id, templateID, fd.Spec.Zone.ID)

	if len(csMachine.Spec.Networks) == 0 && fd.Spec.Zone.Network.ID != "" {
		p.SetNetworkids([]string{fd.Spec.Zone.Network.ID})
	} else {
		firstNetwork := csMachine.Spec.Networks[0]
		zoneNet := fd.Spec.Zone.Network

		if zoneNet.ID != "" && firstNetwork.ID != "" && firstNetwork.ID != zoneNet.ID {
			return errors.Errorf("first network ID %q does not match zone network ID %q", firstNetwork.ID, zoneNet.ID)
		}
		if zoneNet.Name != "" && firstNetwork.Name != "" && firstNetwork.Name != zoneNet.Name {
			return errors.Errorf("first network name %q does not match zone network name %q", firstNetwork.Name, zoneNet.Name)
		}

		ipToNetworkList, err := c.buildIPToNetworkList(csMachine)
		if err != nil {
			return err
		}
		p.SetIptonetworklist(ipToNetworkList)
	}

	setIfNotEmpty(csMachine.Name, p.SetName)
	setIfNotEmpty(capiMachine.Name, p.SetDisplayname)
	setIfNotEmpty(diskOfferingID, p.SetDiskofferingid)
	setIfNotEmpty(c.user.Project.ID, p.SetProjectid)
	setIntIfPositive(csMachine.Spec.DiskOffering.CustomSize, p.SetSize)

	setIfNotEmpty(csMachine.Spec.SSHKey, p.SetKeypair)

	if csMachine.CompressUserdata() {
		userData, err = compress(userData)
		if err != nil {
			return err
		}
	}
	userData = base64.StdEncoding.EncodeToString([]byte(userData))
	setIfNotEmpty(userData, p.SetUserdata)

	if len(csMachine.Spec.AffinityGroupIDs) > 0 {
		p.SetAffinitygroupids(csMachine.Spec.AffinityGroupIDs)
	} else if strings.ToLower(csMachine.Spec.Affinity) != "no" && csMachine.Spec.Affinity != "" {
		p.SetAffinitygroupids([]string{affinity.Spec.ID})
	}

	if csMachine.Spec.Details != nil {
		p.SetDetails(csMachine.Spec.Details)
	}

	deployVMResp, err := c.cs.VirtualMachine.DeployVirtualMachine(p)
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)

		// CloudStack may have created the VM even though it reported an error. We attempt to
		// retrieve the VM so we can populate the CloudStackMachine for the user to manually
		// clean up.
		vm, findErr := findVirtualMachine(c.cs.VirtualMachine, templateID, fd, csMachine, c.user.Project.ID)
		if findErr != nil {
			c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(findErr)
			return fmt.Errorf("%v; find virtual machine: %v", err, findErr)
		}

		// We didn't find a VM so return the original error.
		if vm == nil {
			return err
		}

		csMachine.Spec.InstanceID = ptr.To(vm.Id)
		csMachine.Status.InstanceState = vm.State

		return fmt.Errorf("incomplete vm deployment (vm_id=%v): %w", vm.Id, err)
	}

	csMachine.Spec.InstanceID = ptr.To(deployVMResp.Id)
	csMachine.Status.Status = ptr.To(metav1.StatusSuccess)

	return nil
}

// GetOrCreateVMInstance CreateVMInstance will fetch or create a VM instance, and
// sets the infrastructure machine spec and status accordingly.
func (c *client) GetOrCreateVMInstance(
	csMachine *infrav1.CloudStackMachine,
	capiMachine *clusterv1.Machine,
	_ *infrav1.CloudStackCluster,
	fd *infrav1.CloudStackFailureDomain,
	affinity *infrav1.CloudStackAffinityGroup,
	userData string,
) error {
	// Check if VM instance already exists.
	if err := c.ResolveVMInstanceDetails(csMachine); err == nil ||
		!strings.Contains(strings.ToLower(err.Error()), "no match") {
		return err
	}

	offering, err := c.ResolveServiceOffering(csMachine, fd.Spec.Zone.ID)
	if err != nil {
		return err
	}

	err = c.CheckLimits(&offering)
	if err != nil {
		return err
	}

	if err := c.DeployVM(csMachine, capiMachine, fd, affinity, &offering, userData); err != nil {
		return err
	}

	// Resolve uses a VM metrics request response to fill cloudstack machine status.
	// The deployment response is insufficient.
	return c.ResolveVMInstanceDetails(csMachine)
}

// findVirtualMachine retrieves a virtual machine by matching its expected name, template, failure
// domain zone and failure domain network. If no virtual machine is found it returns nil, nil.
func findVirtualMachine(
	client cloudstack.VirtualMachineServiceIface,
	templateID string,
	failureDomain *infrav1.CloudStackFailureDomain,
	machine *infrav1.CloudStackMachine, projectID string,
) (*cloudstack.VirtualMachine, error) {
	params := client.NewListVirtualMachinesParams()
	params.SetTemplateid(templateID)
	params.SetZoneid(failureDomain.Spec.Zone.ID)
	params.SetNetworkid(failureDomain.Spec.Zone.Network.ID)
	params.SetName(machine.Name)
	setIfNotEmpty(projectID, params.SetProjectid)

	response, err := client.ListVirtualMachines(params)
	if err != nil {
		return nil, err
	}

	if response.Count == 0 {
		return nil, nil
	}

	return response.VirtualMachines[0], nil
}

// DestroyVMInstance Destroys a VM instance. Assumes machine has been fetched prior and has an instance ID.
func (c *client) DestroyVMInstance(csMachine *infrav1.CloudStackMachine) error {
	p := c.cs.Configuration.NewListCapabilitiesParams()
	capabilities, err := c.cs.Configuration.ListCapabilities(p)
	expunge := true
	if err == nil {
		expunge = capabilities.Capabilities.Allowuserexpungerecovervm
	}

	// Attempt deletion regardless of machine state.
	p2 := c.csAsync.VirtualMachine.NewDestroyVirtualMachineParams(*csMachine.Spec.InstanceID)
	volIDs, err := c.listVMInstanceDatadiskVolumeIDs(*csMachine.Spec.InstanceID)
	if err != nil {
		return err
	}
	p2.SetExpunge(expunge)
	setArrayIfNotEmpty(volIDs, p2.SetVolumeids)
	if _, err := c.csAsync.VirtualMachine.DestroyVirtualMachine(p2); err != nil &&
		strings.Contains(strings.ToLower(err.Error()), "unable to find uuid for id") {
		// VM doesn't exist. Success...
		return nil
	} else if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
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

func (c *client) listVMInstanceDatadiskVolumeIDs(instanceID string) ([]string, error) {
	p := c.cs.Volume.NewListVolumesParams()
	p.SetVirtualmachineid(instanceID)
	// VM root volumes are destroyed automatically, no need to explicitly include
	p.SetType("DATADISK")
	setIfNotEmpty(c.user.Project.ID, p.SetProjectid)

	listVOLResp, err := c.csAsync.Volume.ListVolumes(p)
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return nil, err
	}

	var ret []string
	for _, vol := range listVOLResp.Volumes {
		ret = append(ret, vol.Id)
	}

	return ret, nil
}
