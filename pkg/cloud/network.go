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
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const (
	NetOffering         = "DefaultIsolatedNetworkOfferingWithSourceNatService"
	K8sDefaultAPIPort   = 6443
	NetworkTypeIsolated = "Isolated"
	NetworkTypeShared   = "Shared"
	NetworkProtocolTCP  = "tcp"
)

func (c *client) ResolveNetwork(csCluster *infrav1.CloudStackCluster) (retErr error) {
	networkID, count, err := c.cs.Network.GetNetworkID(csCluster.Spec.Network)
	if err != nil {
		retErr = multierror.Append(retErr, errors.Wrapf(
			err, "Could not get Network ID from %s.", csCluster.Spec.Network))
		networkID = csCluster.Spec.Network
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"Expected 1 Network with name %s, but got %d.", csCluster.Spec.Network, count))
	}

	if networkDetails, count, err := c.cs.Network.GetNetworkByID(networkID); err != nil {
		return multierror.Append(retErr, errors.Wrapf(
			err, "Could not get Network by ID %s.", networkID))
	} else if count != 1 {
		return multierror.Append(retErr, errors.Errorf(
			"Expected 1 Network with UUID %s, but got %d.", networkID, count))
	} else {
		csCluster.Status.NetworkID = networkID
		csCluster.Status.NetworkType = networkDetails.Type
	}
	return nil
}

func (c *client) GetOrCreateNetwork(csCluster *infrav1.CloudStackCluster) (retErr error) {
	if retErr = c.ResolveNetwork(csCluster); retErr == nil { // Found network.
		return c.AddClusterTag(ResourceTypeNetwork, csCluster.Status.NetworkID, csCluster, false)
	} else if !strings.Contains(strings.ToLower(retErr.Error()), "no match found") { // Some other error.
		return retErr
	} // Network not found.

	// Create network since it wasn't found.
	offeringID, count, retErr := c.cs.NetworkOffering.GetNetworkOfferingID(NetOffering)
	if retErr != nil {
		return retErr
	} else if count != 1 {
		return errors.New("found more than one network offering")
	}
	p := c.cs.Network.NewCreateNetworkParams(
		csCluster.Spec.Network,
		csCluster.Spec.Network,
		offeringID,
		csCluster.Status.ZoneID)
	setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
	setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)
	resp, err := c.cs.Network.CreateNetwork(p)
	if err != nil {
		return err
	}
	csCluster.Status.NetworkID = resp.Id
	csCluster.Status.NetworkType = resp.Type

	return c.AddClusterTag(ResourceTypeNetwork, csCluster.Status.NetworkID, csCluster, true)
}

func (c *client) DisassociatePublicIpAddressIfNotInUse(csCluster *infrav1.CloudStackCluster) (retError error) {
	okayToDelete, err := c.DoClusterTagsAllowDisposal(ResourceTypeIpAddress, csCluster.Status.PublicIPID)
	if err != nil {
		return err
	}

	if okayToDelete {
		return c.DisassociatePublicIpAddress(csCluster)
	}

	return nil
}

func (c *client) DeleteNetworkIfNotInUse(csCluster *infrav1.CloudStackCluster) (retError error) {
	okayToDelete, err := c.DoClusterTagsAllowDisposal(ResourceTypeNetwork, csCluster.Status.NetworkID)
	if err != nil {
		return err
	}

	if okayToDelete {
		return c.DestroyNetwork(csCluster)
	}

	return nil
}

func (c *client) ResolvePublicIPDetails(csCluster *infrav1.CloudStackCluster) (*cloudstack.PublicIpAddress, error) {
	ip := csCluster.Spec.ControlPlaneEndpoint.Host

	p := c.cs.Address.NewListPublicIpAddressesParams()
	p.SetAllocatedonly(false)
	setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
	setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)
	if ip != "" {
		p.SetIpaddress(ip)
	}
	publicAddresses, err := c.cs.Address.ListPublicIpAddresses(p)

	if err != nil {
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
	return nil, errors.Errorf(`no public addresses found in network: "%s"`, csCluster.Spec.Network)
}

// AssociatePublicIPAddress Gets a PublicIP and associates it.
func (c *client) AssociatePublicIPAddress(csCluster *infrav1.CloudStackCluster) (retErr error) {
	publicAddress, err := c.ResolvePublicIPDetails(csCluster)
	if err != nil {
		return err
	}

	csCluster.Spec.ControlPlaneEndpoint.Host = publicAddress.Ipaddress
	csCluster.Status.PublicIPID = publicAddress.Id
	allocatedByCapc := publicAddress.Allocated == ""

	if publicAddress.Allocated != "" && publicAddress.Associatednetworkid == csCluster.Status.NetworkID {
		// Address already allocated to network. Allocated is a timestamp -- not a boolean.
		return c.AddClusterTag(ResourceTypeIpAddress, publicAddress.Id, csCluster, allocatedByCapc)
	} // Address not yet allocated. Allocate now.

	// Public IP found, but not yet allocated to network.
	p := c.cs.Address.NewAssociateIpAddressParams()
	p.SetNetworkid(csCluster.Status.NetworkID)
	p.SetIpaddress(csCluster.Spec.ControlPlaneEndpoint.Host)
	setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
	setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)
	if _, err := c.cs.Address.AssociateIpAddress(p); err != nil {
		return err
	}
	return c.AddClusterTag(ResourceTypeIpAddress, publicAddress.Id, csCluster, allocatedByCapc)
}

func (c *client) DisassociatePublicIpAddress(csCluster *infrav1.CloudStackCluster) (retErr error) {
	p := c.cs.Address.NewDisassociateIpAddressParams(csCluster.Status.PublicIPID)
	_, retErr = c.cs.Address.DisassociateIpAddress(p)
	return retErr
}

func (c *client) OpenFirewallRules(csCluster *infrav1.CloudStackCluster) (retErr error) {
	p := c.cs.Firewall.NewCreateEgressFirewallRuleParams(csCluster.Status.NetworkID, NetworkProtocolTCP)
	_, retErr = c.cs.Firewall.CreateEgressFirewallRule(p)
	if retErr != nil && strings.Contains(strings.ToLower(retErr.Error()), "there is already") { // Already a firewall rule here.
		retErr = nil
	}
	return retErr
}

func (c *client) ResolveLoadBalancerRuleDetails(csCluster *infrav1.CloudStackCluster) (retErr error) {
	p := c.cs.LoadBalancer.NewListLoadBalancerRulesParams()
	p.SetPublicipid(csCluster.Status.PublicIPID)
	setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
	setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)
	loadBalancerRules, err := c.cs.LoadBalancer.ListLoadBalancerRules(p)
	if err != nil {
		return err
	}
	for _, rule := range loadBalancerRules.LoadBalancerRules {
		if rule.Publicport == strconv.Itoa(int(csCluster.Spec.ControlPlaneEndpoint.Port)) {
			csCluster.Status.LBRuleID = rule.Id
			return nil
		}
	}
	return errors.New("no load balancer rule found")
}

// GetOrCreateLoadBalancerRule Create a load balancer rule that can be assigned to instances.
func (c *client) GetOrCreateLoadBalancerRule(csCluster *infrav1.CloudStackCluster) (retErr error) {
	// Check if rule exists.
	if err := c.ResolveLoadBalancerRuleDetails(csCluster); err == nil ||
		!strings.Contains(strings.ToLower(err.Error()), "no load balancer rule found") {
		return err
	}

	p := c.cs.LoadBalancer.NewCreateLoadBalancerRuleParams(
		"roundrobin", "Kubernetes_API_Server", K8sDefaultAPIPort, K8sDefaultAPIPort)
	p.SetNetworkid(csCluster.Status.NetworkID)
	if csCluster.Spec.ControlPlaneEndpoint.Port != 0 { // Override default public port if endpoint port specified.
		p.SetPublicport(int(csCluster.Spec.ControlPlaneEndpoint.Port))
	}
	p.SetPublicipid(csCluster.Status.PublicIPID)
	p.SetProtocol(NetworkProtocolTCP)
	setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
	setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)
	resp, err := c.cs.LoadBalancer.CreateLoadBalancerRule(p)
	if err != nil {
		return err
	}
	csCluster.Status.LBRuleID = resp.Id
	return nil
}

func (c *client) DestroyNetwork(csCluster *infrav1.CloudStackCluster) (retErr error) {
	_, retErr = c.cs.Network.DeleteNetwork(c.cs.Network.NewDeleteNetworkParams(csCluster.Status.NetworkID))
	return retErr
}

func (c *client) AssignVMToLoadBalancerRule(csCluster *infrav1.CloudStackCluster, instanceID string) (retErr error) {

	// Check that the instance isn't already in LB rotation.
	lbRuleInstances, retErr := c.cs.LoadBalancer.ListLoadBalancerRuleInstances(
		c.cs.LoadBalancer.NewListLoadBalancerRuleInstancesParams(csCluster.Status.LBRuleID))
	if retErr != nil {
		return retErr
	}
	for _, instance := range lbRuleInstances.LoadBalancerRuleInstances {
		if instance.Id == instanceID { // Already assigned to load balancer..
			return nil
		}
	}

	// Assign to Load Balancer.
	p := c.cs.LoadBalancer.NewAssignToLoadBalancerRuleParams(csCluster.Status.LBRuleID)
	p.SetVirtualmachineids([]string{instanceID})
	_, retErr = c.cs.LoadBalancer.AssignToLoadBalancerRule(p)
	return retErr
}
