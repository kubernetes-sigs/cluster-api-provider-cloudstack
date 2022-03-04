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
	capcv1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type NetworkIface interface {
	ResolveNetworkStatuses(*capcv1.CloudStackCluster) error
	ResolveNetwork(*capcv1.CloudStackCluster, *capcv1.Network) error
	CreateIsolatedNetwork(*capcv1.CloudStackCluster) error
	OpenFirewallRules(*capcv1.CloudStackCluster) error
	ResolvePublicIPDetails(*capcv1.CloudStackCluster) (*cloudstack.PublicIpAddress, error)
	ResolveLoadBalancerRuleDetails(*capcv1.CloudStackCluster) error
	GetOrCreateLoadBalancerRule(*capcv1.CloudStackCluster) error
	GetOrCreateIsolatedNetwork(*capcv1.CloudStackCluster) error
	AssociatePublicIPAddress(*capcv1.CloudStackCluster) error
}

const (
	NetOffering         = "DefaultIsolatedNetworkOfferingWithSourceNatService"
	K8sDefaultAPIPort   = 6443
	NetworkTypeIsolated = "Isolated"
	NetworkTypeShared   = "Shared"
	NetworkProtocolTCP  = "tcp"
)

// usesIsolatedNetwork returns true if this cluster is specs an isolated network.
// Assumes that the a fetch has been done on network statuses prior.
func UsesIsolatedNetwork(csCluster *capcv1.CloudStackCluster) bool {
	// Check for Isolated network use case.
	if len(csCluster.Status.Zones) == 1 { // Where the only specced network
		zoneStatus := csCluster.Status.Zones.GetOne()
		if zoneStatus.Network.Type == "" || // doesn't exist or
			zoneStatus.Network.Type == NetworkTypeIsolated { // exists and is an isolated network.
			return true
		}
	}
	return false
}

// NetworkExists checks that the network already exists based on the presence of all fields.
// Assumes that the a fetch has been done on network statuses prior.
func NetworkExists(net capcv1.Network) bool {
	if net.Name != "" && net.Type != "" && net.ID != "" {
		return true
	}
	return false
}

// ResolveNetwork fetches networks' ID, Name, and Type.
func (c *client) ResolveNetwork(csCluster *capcv1.CloudStackCluster, net *capcv1.Network) (retErr error) {
	// TODO rebuild this to consider cases with networks in many zones.
	// Use ListNetworks instead.
	netName := net.Name
	netDetails, count, err := c.cs.Network.GetNetworkByName(netName)
	if err != nil {
		retErr = multierror.Append(retErr, errors.Wrapf(err, "could not get Network ID from %s", netName))
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"expected 1 Network with name %s, but got %d", netName, count))
	} else { // Got netID from the network's name.
		net.ID = netDetails.Id
		net.Type = netDetails.Type
		return nil
	}

	// Now get network details.
	netDetails, count, err = c.cs.Network.GetNetworkByID(net.ID)
	if err != nil {
		return multierror.Append(retErr, errors.Wrapf(err, "could not get Network by ID %s", net.ID))
	} else if count != 1 {
		return multierror.Append(retErr, errors.Errorf("expected 1 Network with UUID %s, but got %d", net.ID, count))
	}
	net.Name = netDetails.Name
	net.ID = netDetails.Id
	net.Type = netDetails.Type
	return nil
}

func generateNetworkTagName(csCluster *capcv1.CloudStackCluster) string {
	return clusterTagNamePrefix + string(csCluster.UID)
}

// getOfferingID fetches an offering id.
func (c *client) getOfferingID() (string, error) {
	offeringID, count, retErr := c.cs.NetworkOffering.GetNetworkOfferingID(NetOffering)
	if retErr != nil {
		return "", retErr
	} else if count != 1 {
		return "", errors.New("found more than one network offering")
	}
	return offeringID, nil
}

// CreateIsolatedNetwork creates an isolated network in the relevant Zone.
// Assumes that there is only the one zone in the cluster.
func (c *client) CreateIsolatedNetwork(csCluster *capcv1.CloudStackCluster) (retErr error) {
	zoneStatus := *csCluster.Status.Zones.GetOne() // Should only be the one...
	netStatus := zoneStatus.Network

	// Fetch offering ID.
	offeringID, err := c.getOfferingID()
	if err != nil {
		return err
	}

	// Do creation.
	p := c.cs.Network.NewCreateNetworkParams(netStatus.Name, netStatus.Name, offeringID, zoneStatus.ID)
	setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
	setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)
	resp, err := c.cs.Network.CreateNetwork(p)
	if err != nil {
		return err
	}
	if err := c.AddClusterTag(ResourceTypeNetwork, zoneStatus.Network.ID, csCluster); err != nil {
		return err
	}
	if err := c.AddCreatedByCAPCTag(ResourceTypeNetwork, zoneStatus.Network.ID); err != nil {
		return err
	}

	// Update Zone/Network status accordingly.
	netStatus.ID = resp.Id
	netStatus.Type = resp.Type
	zoneStatus.Network = netStatus
	csCluster.Status.Zones[zoneStatus.Name] = zoneStatus

	return nil
}

// ResolveNetworkStatuses fetches details on all networks specced, but will not modify ACS settings.
func (c *client) ResolveNetworkStatuses(csCluster *capcv1.CloudStackCluster) (retErr error) {
	// At this point network status should have been populated (copied) from the spec.
	for _, zoneStatus := range csCluster.Status.Zones {
		if retErr = c.ResolveNetwork(csCluster, &zoneStatus.Network); retErr == nil { // Found network
			csCluster.Status.Zones[zoneStatus.ID] = zoneStatus
		} else if !strings.Contains(retErr.Error(), "No match found") { // Some other error.
			return retErr
		}
	}

	return nil
}

func (c *client) RemoveClusterTagFromNetwork(csCluster *capcv1.CloudStackCluster, net capcv1.Network) (retError error) {

	tags, err := c.GetTags(ResourceTypeNetwork, net.ID)
	if err != nil {
		return err
	}

	clusterTagName := generateNetworkTagName(csCluster)
	if tagValue := tags[clusterTagName]; tagValue != "" {
		if err = c.DeleteTags(ResourceTypeNetwork, net.ID, map[string]string{clusterTagName: tagValue}); err != nil {
			return err
		}
	}

	return nil
}

func (c *client) DeleteNetworkIfNotInUse(csCluster *capcv1.CloudStackCluster, net capcv1.Network) (retError error) {
	tags, err := c.GetTags(ResourceTypeNetwork, net.ID)
	if err != nil {
		return err
	}

	var clusterTagCount int
	for tagName := range tags {
		if strings.HasPrefix(tagName, clusterTagNamePrefix) {
			clusterTagCount++
		}
	}

	if clusterTagCount == 0 && tags[createdByCAPCTagName] != "" {
		return c.DestroyNetwork(net)
	}

	return nil
}

func (c *client) ResolvePublicIPDetails(csCluster *capcv1.CloudStackCluster) (*cloudstack.PublicIpAddress, error) {
	ip := csCluster.Spec.ControlPlaneEndpoint.Host

	zoneStatus := csCluster.Status.Zones.GetOne()

	p := c.cs.Address.NewListPublicIpAddressesParams()
	p.SetAllocatedonly(false)
	p.SetZoneid(zoneStatus.ID)
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
		return nil, errors.New("all Public IP Adresse(s) found were already allocated")
	}
	return nil, errors.New("no public addresses found in available networks")
}

// AssociatePublicIPAddress Gets a PublicIP and associates it.
func (c *client) AssociatePublicIPAddress(csCluster *capcv1.CloudStackCluster) (retErr error) {
	publicAddress, err := c.ResolvePublicIPDetails(csCluster)
	if err != nil {
		return err
	}

	csCluster.Spec.ControlPlaneEndpoint.Host = publicAddress.Ipaddress
	csCluster.Status.PublicIPID = publicAddress.Id

	zoneStatus := csCluster.Status.Zones.GetOne()

	// Public IP found, but not yet allocated to network.
	p := c.cs.Address.NewAssociateIpAddressParams()
	p.SetIpaddress(csCluster.Spec.ControlPlaneEndpoint.Host)
	p.SetNetworkid(zoneStatus.Network.ID)
	setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
	setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)
	if _, err := c.cs.Address.AssociateIpAddress(p); err != nil {
		return err
	}
	if err := c.AddClusterTag(ResourceTypeIPAddress, publicAddress.Id, csCluster); err != nil {
		return err
	}
	return nil
}

func (c *client) OpenFirewallRules(csCluster *capcv1.CloudStackCluster) (retErr error) {
	p := c.cs.Firewall.NewCreateEgressFirewallRuleParams(csCluster.Status.PublicIPNetworkID, NetworkProtocolTCP)
	_, retErr = c.cs.Firewall.CreateEgressFirewallRule(p)
	if retErr != nil && strings.Contains(strings.ToLower(retErr.Error()), "there is already") { // Already a firewall rule here.
		retErr = nil
	}
	return retErr
}

func (c *client) DisassociatePublicIPAddress(csCluster *capcv1.CloudStackCluster) (retErr error) {
	// Remove the CAPC creation tag, so it won't be there the next time this address is associated.
	retErr = c.DeleteCreatedByCAPCTag(ResourceTypeIPAddress, csCluster.Status.PublicIPID)
	if retErr != nil {
		return retErr
	}

	p := c.cs.Address.NewDisassociateIpAddressParams(csCluster.Status.PublicIPID)
	_, retErr = c.cs.Address.DisassociateIpAddress(p)
	return retErr
}

func (c *client) DisassociatePublicIPAddressIfNotInUse(csCluster *capcv1.CloudStackCluster) (retError error) {
	tagsAllowDisposal, err := c.DoClusterTagsAllowDisposal(ResourceTypeIPAddress, csCluster.Status.PublicIPID)
	if err != nil {
		return err
	}

	// Can't disassociate an address if it's the source NAT address.
	publicIP, _, err := c.cs.Address.GetPublicIpAddressByID(csCluster.Status.PublicIPID)
	if err != nil {
		return err
	}
	sourceNAT := publicIP != nil && publicIP.Issourcenat

	if tagsAllowDisposal && !sourceNAT {
		return c.DisassociatePublicIPAddress(csCluster)
	}

	return nil
}

func (c *client) ResolveLoadBalancerRuleDetails(csCluster *capcv1.CloudStackCluster) (retErr error) {
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
func (c *client) GetOrCreateLoadBalancerRule(csCluster *capcv1.CloudStackCluster) (retErr error) {
	// Check if rule exists.
	if err := c.ResolveLoadBalancerRuleDetails(csCluster); err == nil ||
		!strings.Contains(strings.ToLower(err.Error()), "no load balancer rule found") {
		return err
	}

	p := c.cs.LoadBalancer.NewCreateLoadBalancerRuleParams(
		"roundrobin", "Kubernetes_API_Server", K8sDefaultAPIPort, K8sDefaultAPIPort)

	p.SetNetworkid(csCluster.Status.Zones.GetOne().Network.ID)
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

func (c *client) DestroyNetwork(net capcv1.Network) (retErr error) {
	_, retErr = c.cs.Network.DeleteNetwork(c.cs.Network.NewDeleteNetworkParams(net.ID))
	return retErr
}

func (c *client) AssignVMToLoadBalancerRule(csCluster *capcv1.CloudStackCluster, instanceID string) (retErr error) {

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

// GetOrCreateIsolatedNetwork fetches or builds out the necessary structures for isolated network use.
func (c *client) GetOrCreateIsolatedNetwork(csCluster *capcv1.CloudStackCluster) error {
	onlyNetStatus := csCluster.Status.Zones.GetOne().Network
	if !NetworkExists(onlyNetStatus) { // create isolated network.
		if err := c.CreateIsolatedNetwork(csCluster); err != nil {
			return err
		}
	}

	if csCluster.Status.PublicIPID == "" { // Don't try to get public IP again it's already been fetched.
		if err := c.AssociatePublicIPAddress(csCluster); err != nil {
			return err
		}
	}
	if err := c.GetOrCreateLoadBalancerRule(csCluster); err != nil {
		return err
	}
	return nil
}
