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
	"strconv"
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
)

type IsoNetworkIface interface {
	GetOrCreateIsolatedNetwork(*infrav1.CloudStackFailureDomain, *infrav1.CloudStackIsolatedNetwork, *infrav1.CloudStackCluster) error

	AssociatePublicIPAddress(*infrav1.CloudStackFailureDomain, *infrav1.CloudStackIsolatedNetwork, *infrav1.CloudStackCluster) error
	GetOrCreateLoadBalancerRule(*infrav1.CloudStackIsolatedNetwork, *infrav1.CloudStackCluster) error
	OpenFirewallRules(*infrav1.CloudStackIsolatedNetwork) error
	GetPublicIP(*infrav1.CloudStackFailureDomain, *infrav1.CloudStackCluster) (*cloudstack.PublicIpAddress, error)
	ResolveLoadBalancerRuleDetails(*infrav1.CloudStackIsolatedNetwork) error

	AssignVMToLoadBalancerRule(isoNet *infrav1.CloudStackIsolatedNetwork, instanceID string) error
	DeleteNetwork(infrav1.Network) error
	DisposeIsoNetResources(*infrav1.CloudStackIsolatedNetwork, *infrav1.CloudStackCluster) error
}

// getOfferingID fetches an offering id.
func (c *client) getOfferingID(offeringName string) (string, error) {
	offeringID, count, retErr := c.cs.NetworkOffering.GetNetworkOfferingID(offeringName)
	if retErr != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(retErr)
		return "", retErr
	} else if count != 1 {
		return "", errors.New("found more than one network offering")
	}
	return offeringID, nil
}

// AssociatePublicIPAddress Gets a PublicIP and associates the public IP to passed isolated network.
func (c *client) AssociatePublicIPAddress(
	fd *infrav1.CloudStackFailureDomain,
	isoNet *infrav1.CloudStackIsolatedNetwork,
	csCluster *infrav1.CloudStackCluster,
) (retErr error) {
	// Check specified IP address is available or get an unused one if not specified.
	publicAddress, err := c.GetPublicIP(fd, csCluster)
	if err != nil {
		return errors.Wrapf(err, "fetching a public IP address")
	}
	isoNet.Spec.ControlPlaneEndpoint.Host = publicAddress.Ipaddress
	csCluster.Spec.ControlPlaneEndpoint.Host = publicAddress.Ipaddress
	isoNet.Status.PublicIPID = publicAddress.Id

	// Check if the address is already associated with the network or VPC.
	if publicAddress.Associatednetworkid == isoNet.Spec.ID || (isoNet.Spec.VPC != nil && publicAddress.Vpcid == isoNet.Spec.VPC.ID) {
		return nil
	}

	// Public IP found, but not yet associated with network -- associate it.
	p := c.cs.Address.NewAssociateIpAddressParams()
	p.SetIpaddress(isoNet.Spec.ControlPlaneEndpoint.Host)
	p.SetNetworkid(isoNet.Spec.ID)
	if isoNet.Spec.VPC != nil && isoNet.Spec.VPC.ID != "" {
		p.SetVpcid(isoNet.Spec.VPC.ID)
	}
	setIfNotEmpty(c.user.Project.ID, p.SetProjectid)
	if _, err := c.cs.Address.AssociateIpAddress(p); err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return errors.Wrapf(err,
			"associating public IP address with ID %s to network with ID %s",
			publicAddress.Id, isoNet.Spec.ID)
	} else if err := c.AddClusterTag(ResourceTypeIPAddress, publicAddress.Id, csCluster); err != nil {
		return errors.Wrapf(err,
			"adding tag to public IP address with ID %s", publicAddress.Id)
	} else if err := c.AddCreatedByCAPCTag(ResourceTypeIPAddress, isoNet.Status.PublicIPID); err != nil {
		return errors.Wrapf(err,
			"adding tag to public IP address with ID %s", publicAddress.Id)
	}
	return nil
}

// CreateIsolatedNetwork creates an isolated network in the relevant FailureDomain per passed network specification.
func (c *client) CreateIsolatedNetwork(fd *infrav1.CloudStackFailureDomain, isoNet *infrav1.CloudStackIsolatedNetwork) (retErr error) {
	// Get network offering ID.
	offeringName := NetOffering
	// First, check if VPC is specified and handle it
	if isoNet.Spec.VPC != nil && (isoNet.Spec.VPC.Name != "" || isoNet.Spec.VPC.ID != "") {
		// Try to resolve or create the VPC
		err := c.ResolveVPC(isoNet.Spec.VPC)
		if err != nil { // No VPC found, create it
			err = c.CreateVPC(fd, isoNet.Spec.VPC)
			if err != nil {
				return errors.Wrap(err, "creating VPC with name "+isoNet.Spec.VPC.Name)
			}
		}
		offeringName = NetVPCOffering
	}

	// Get network offering ID.
	offeringID, err := c.getOfferingID(offeringName)
	if err != nil {
		return err
	}

	// Do isolated network creation.
	p := c.cs.Network.NewCreateNetworkParams(isoNet.Spec.Name, offeringID, fd.Spec.Zone.ID)
	p.SetDisplaytext(isoNet.Spec.Name)

	if isoNet.Spec.Gateway != "" {
		p.SetGateway(isoNet.Spec.Gateway)
	}

	if isoNet.Spec.Netmask != "" {
		p.SetNetmask(isoNet.Spec.Netmask)
	}

	setIfNotEmpty(c.user.Project.ID, p.SetProjectid)

	// If VPC is specified, set the VPC ID for the network
	if isoNet.Spec.VPC != nil && isoNet.Spec.VPC.ID != "" {
		p.SetVpcid(isoNet.Spec.VPC.ID)
	}

	resp, err := c.cs.Network.CreateNetwork(p)
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return errors.Wrapf(err, "creating network with name %s", isoNet.Spec.Name)
	}
	isoNet.Spec.ID = resp.Id
	isoNet.Spec.Gateway = resp.Gateway
	isoNet.Spec.Netmask = resp.Netmask
	return c.AddCreatedByCAPCTag(ResourceTypeNetwork, isoNet.Spec.ID)
}

// OpenFirewallRules opens a CloudStack egress firewall for an isolated network.
func (c *client) OpenFirewallRules(isoNet *infrav1.CloudStackIsolatedNetwork) (retErr error) {
	if isoNet.Spec.VPC != nil && isoNet.Spec.VPC.ID != "" {
		return nil
	}
	protocols := []string{NetworkProtocolTCP, NetworkProtocolUDP, NetworkProtocolICMP}
	for _, proto := range protocols {
		p := c.cs.Firewall.NewCreateEgressFirewallRuleParams(isoNet.Spec.ID, proto)

		if proto == "icmp" {
			p.SetIcmptype(-1)
			p.SetIcmpcode(-1)
		}

		_, err := c.cs.Firewall.CreateEgressFirewallRule(p)
		if err != nil &&
			// Ignore errors regarding already existing fw rules for TCP/UDP
			!strings.Contains(strings.ToLower(err.Error()), "there is already") &&
			// Ignore errors regarding already existing fw rule for ICMP
			!strings.Contains(strings.ToLower(err.Error()), "new rule conflicts with existing rule") {
			retErr = multierror.Append(retErr, errors.Wrapf(
				err, "failed creating egress firewall rule for network ID %s protocol %s", isoNet.Spec.ID, proto))
		}
	}
	c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(retErr)
	return retErr
}

// GetPublicIP gets a public IP with ID for cluster endpoint.
func (c *client) GetPublicIP(
	fd *infrav1.CloudStackFailureDomain,
	csCluster *infrav1.CloudStackCluster,
) (*cloudstack.PublicIpAddress, error) {
	ip := csCluster.Spec.ControlPlaneEndpoint.Host

	p := c.cs.Address.NewListPublicIpAddressesParams()
	p.SetAllocatedonly(false)
	p.SetZoneid(fd.Spec.Zone.ID)
	setIfNotEmpty(ip, p.SetIpaddress)
	setIfNotEmpty(c.user.Project.ID, p.SetProjectid)
	publicAddresses, err := c.cs.Address.ListPublicIpAddresses(p)
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return nil, err
	} else if ip != "" && publicAddresses.Count == 1 { // Endpoint specified and IP found.
		// Ignore already allocated here since the IP was specified.
		return publicAddresses.PublicIpAddresses[0], nil
	} else if publicAddresses.Count > 0 { // Endpoint not specified.
		for _, v := range publicAddresses.PublicIpAddresses { // Pick first available address.
			if v.Allocated == "" { // Found un-allocated Public IP.
				return v, nil
			}
		}
		return nil, errors.New("all Public IP Address(es) found were already allocated")
	}
	return nil, errors.New("no public addresses found in available networks")
}

// ResolveLoadBalancerRuleDetails resolves the details of a load balancer rule by PublicIPID and Port.
func (c *client) ResolveLoadBalancerRuleDetails(
	isoNet *infrav1.CloudStackIsolatedNetwork,
) error {
	p := c.cs.LoadBalancer.NewListLoadBalancerRulesParams()
	p.SetPublicipid(isoNet.Status.PublicIPID)
	setIfNotEmpty(c.user.Project.ID, p.SetProjectid)
	loadBalancerRules, err := c.cs.LoadBalancer.ListLoadBalancerRules(p)
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return errors.Wrap(err, "listing load balancer rules")
	}
	for _, rule := range loadBalancerRules.LoadBalancerRules {
		if rule.Publicport == strconv.Itoa(int(isoNet.Spec.ControlPlaneEndpoint.Port)) {

			isoNet.Status.LBRuleID = rule.Id
			return nil
		}
	}
	return errors.New("no load balancer rule found")
}

// GetOrCreateLoadBalancerRule Create a load balancer rule that can be assigned to instances.
func (c *client) GetOrCreateLoadBalancerRule(
	isoNet *infrav1.CloudStackIsolatedNetwork,
	csCluster *infrav1.CloudStackCluster,
) (retErr error) {
	// Check/set ports.
	// Prefer control plane endpoint. Take iso net port if CP missing. Set to default if both missing.
	if csCluster.Spec.ControlPlaneEndpoint.Port != 0 {
		isoNet.Spec.ControlPlaneEndpoint.Port = csCluster.Spec.ControlPlaneEndpoint.Port
	} else if isoNet.Spec.ControlPlaneEndpoint.Port != 0 { // Override default public port if endpoint port specified.
		csCluster.Spec.ControlPlaneEndpoint.Port = isoNet.Spec.ControlPlaneEndpoint.Port
	} else {
		csCluster.Spec.ControlPlaneEndpoint.Port = 6443
		isoNet.Spec.ControlPlaneEndpoint.Port = 6443
	}

	// Check if rule exists.
	if err := c.ResolveLoadBalancerRuleDetails(isoNet); err == nil ||
		!strings.Contains(strings.ToLower(err.Error()), "no load balancer rule found") {
		return errors.Wrap(err, "resolving load balancer rule details")
	}

	p := c.cs.LoadBalancer.NewCreateLoadBalancerRuleParams(
		"roundrobin", "Kubernetes_API_Server", K8sDefaultAPIPort, K8sDefaultAPIPort)
	p.SetPublicport(int(csCluster.Spec.ControlPlaneEndpoint.Port))
	p.SetNetworkid(isoNet.Spec.ID)

	p.SetPublicipid(isoNet.Status.PublicIPID)
	p.SetProtocol(NetworkProtocolTCP)
	resp, err := c.cs.LoadBalancer.CreateLoadBalancerRule(p)
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return err
	}
	isoNet.Status.LBRuleID = resp.Id
	return nil
}

// GetOrCreateIsolatedNetwork fetches or builds out the necessary structures for isolated network use.
func (c *client) GetOrCreateIsolatedNetwork(
	fd *infrav1.CloudStackFailureDomain,
	isoNet *infrav1.CloudStackIsolatedNetwork,
	csCluster *infrav1.CloudStackCluster,
) error {
	// Get or create the isolated network itself and resolve details into passed custom resources.
	net := isoNet.Network()
	if err := c.ResolveNetwork(net); err != nil { // Doesn't exist, create isolated network.
		if err = c.CreateIsolatedNetwork(fd, isoNet); err != nil {
			return errors.Wrap(err, "creating a new isolated network")
		}
	} else { // Network existed and was resolved. Set ID on isoNet CloudStackIsolatedNetwork in case it only had name set.
		isoNet.Spec.ID = net.ID
		isoNet.Spec.Gateway = net.Gateway
		isoNet.Spec.Netmask = net.Netmask
		if net.VPC != nil && net.VPC.ID != "" {
			isoNet.Spec.VPC = net.VPC
		}
	}

	// Tag the created network.
	networkID := isoNet.Spec.ID
	if err := c.AddClusterTag(ResourceTypeNetwork, networkID, csCluster); err != nil {
		return errors.Wrapf(err, "tagging network with id %s", networkID)
	}

	// Tag the created VPC.
	if net.VPC != nil && net.VPC.ID != "" {
		if err := c.AddClusterTag(ResourceTypeVPC, net.VPC.ID, csCluster); err != nil {
			return errors.Wrapf(err, "tagging VPC with id %s", net.VPC.ID)
		}
	}

	// Associate Public IP with CloudStackIsolatedNetwork
	if err := c.AssociatePublicIPAddress(fd, isoNet, csCluster); err != nil {
		return errors.Wrapf(err, "associating public IP address to csCluster")
	}

	// Setup a load balancing rule to map VMs to Public IP.
	if err := c.GetOrCreateLoadBalancerRule(isoNet, csCluster); err != nil {
		return errors.Wrap(err, "getting or creating load balancing rule")
	}

	//  Open the Isolated Network on endopint port.
	return errors.Wrap(c.OpenFirewallRules(isoNet), "opening the isolated network's firewall")
}

// AssignVMToLoadBalancerRule assigns a VM instance to a load balancing rule (specifying lb membership).
func (c *client) AssignVMToLoadBalancerRule(isoNet *infrav1.CloudStackIsolatedNetwork, instanceID string) (retErr error) {

	// Check that the instance isn't already in LB rotation.
	lbRuleInstances, retErr := c.cs.LoadBalancer.ListLoadBalancerRuleInstances(
		c.cs.LoadBalancer.NewListLoadBalancerRuleInstancesParams(isoNet.Status.LBRuleID))
	if retErr != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(retErr)
		return retErr
	}
	for _, instance := range lbRuleInstances.LoadBalancerRuleInstances {
		if instance.Id == instanceID { // Already assigned to load balancer..
			return nil
		}
	}

	// Assign to Load Balancer.
	p := c.cs.LoadBalancer.NewAssignToLoadBalancerRuleParams(isoNet.Status.LBRuleID)
	p.SetVirtualmachineids([]string{instanceID})
	_, retErr = c.cs.LoadBalancer.AssignToLoadBalancerRule(p)
	c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(retErr)
	return retErr
}

// DeleteNetwork deletes an isolated network.
func (c *client) DeleteNetwork(net infrav1.Network) error {
	_, err := c.cs.Network.DeleteNetwork(c.cs.Network.NewDeleteNetworkParams(net.ID))
	c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
	return errors.Wrapf(err, "deleting network with id %s", net.ID)
}

// DisposeIsoNetResources cleans up isolated network resources.
func (c *client) DisposeIsoNetResources(
	isoNet *infrav1.CloudStackIsolatedNetwork,
	csCluster *infrav1.CloudStackCluster,
) (retError error) {
	if isoNet.Status.PublicIPID != "" {
		if err := c.DeleteClusterTag(ResourceTypeIPAddress, isoNet.Status.PublicIPID, csCluster); err != nil {
			return err
		}
		if err := c.DisassociatePublicIPAddressIfNotInUse(isoNet); err != nil {
			return err
		}
	}
	if err := c.RemoveClusterTagFromNetwork(csCluster, *isoNet.Network()); err != nil {
		return err
	}
	if err := c.DeleteNetworkIfNotInUse(*isoNet.Network()); err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match found") {
		return err
	}
	if isoNet.Spec.VPC != nil && isoNet.Spec.VPC.ID != "" {
		if err := c.RemoveClusterTagFromVPC(csCluster, *isoNet.Spec.VPC); err != nil {
			return err
		}
		if err := c.DeleteVPCIfNotInUse(*isoNet.Spec.VPC); err != nil && !strings.Contains(strings.ToLower(err.Error()), "no match found") {
			return err
		}
	}
	return nil
}

// DeleteNetworkIfNotInUse deletes an isolated network if the network is no longer in use (indicated by in use tags).
func (c *client) DeleteNetworkIfNotInUse(net infrav1.Network) (retError error) {
	tags, err := c.GetTags(ResourceTypeNetwork, net.ID)
	if err != nil {
		return err
	}

	var clusterTagCount int
	for tagName := range tags {
		if strings.HasPrefix(tagName, ClusterTagNamePrefix) {
			clusterTagCount++
		}
	}

	if clusterTagCount == 0 && tags[CreatedByCAPCTagName] != "" {
		return c.DeleteNetwork(net)
	}

	return nil
}

// DisassociatePublicIPAddressIfNotInUse removes a CloudStack public IP association from passed isolated network
// if it is no longer in use (indicated by in use tags).
func (c *client) DisassociatePublicIPAddressIfNotInUse(isoNet *infrav1.CloudStackIsolatedNetwork) (retError error) {
	if tagsAllowDisposal, err := c.DoClusterTagsAllowDisposal(ResourceTypeIPAddress, isoNet.Status.PublicIPID); err != nil {
		return err
	} else if publicIP, _, err := c.cs.Address.GetPublicIpAddressByID(isoNet.Status.PublicIPID, cloudstack.WithProject(c.user.Project.ID)); err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return err
	} else if publicIP == nil || publicIP.Issourcenat { // Can't disassociate an address if it's the source NAT address.
		return nil
	} else if tagsAllowDisposal {
		return c.DisassociatePublicIPAddress(isoNet)
	}
	return nil
}

// DisassociatePublicIPAddress removes a CloudStack public IP association from passed isolated network.
func (c *client) DisassociatePublicIPAddress(isoNet *infrav1.CloudStackIsolatedNetwork) (retErr error) {
	// Remove the CAPC creation tag, so it won't be there the next time this address is associated.
	retErr = c.DeleteCreatedByCAPCTag(ResourceTypeIPAddress, isoNet.Status.PublicIPID)
	if retErr != nil {
		return retErr
	}

	p := c.cs.Address.NewDisassociateIpAddressParams(isoNet.Status.PublicIPID)
	_, retErr = c.cs.Address.DisassociateIpAddress(p)
	c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(retErr)
	return retErr
}
