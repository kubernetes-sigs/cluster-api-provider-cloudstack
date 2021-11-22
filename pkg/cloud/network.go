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
	"strconv"
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
)

const NETOFFERING = "DefaultIsolatedNetworkOfferingWithSourceNatService"

func FetchNetwork(cs *cloudstack.CloudStackClient, csCluster *infrav1.CloudStackCluster) (retErr error) {
	// Get NetworkID.
	csCluster.Status.NetworkID, _, retErr = cs.Network.GetNetworkID(csCluster.Spec.Network)
	if retErr != nil {
		return retErr
	}
	return
}

func CreateNetwork(cs *cloudstack.CloudStackClient, csCluster *infrav1.CloudStackCluster) (retErr error) {
	if retErr = FetchNetwork(cs, csCluster); retErr == nil { // Found network.
		return nil
	} else if !strings.Contains(retErr.Error(), "No match found") { // Some other error.
		return retErr
	} // Network not found. Create it.

	// Create network since it wasn't found.
	offeringId, count, retErr := cs.NetworkOffering.GetNetworkOfferingID(NETOFFERING)
	if retErr != nil {
		return retErr
	} else if count != 1 {
		return errors.New("found more than one network offering.")
	}
	p := cs.Network.NewCreateNetworkParams(
		csCluster.Spec.Network,
		csCluster.Spec.Network,
		offeringId,
		csCluster.Status.ZoneID)
	resp, err := cs.Network.CreateNetwork(p)
	if err != nil {
		return err
	}
	csCluster.Status.NetworkID = resp.Id

	return nil
}

func FetchPublicIP(cs *cloudstack.CloudStackClient, csCluster *infrav1.CloudStackCluster) (retErr error) {
	p := cs.Address.NewListPublicIpAddressesParams()
	p.SetAllocatedonly(false)
	if ip := csCluster.Spec.ControlPlaneEndpoint.Host; ip != "" {
		p.SetIpaddress(ip)
	}
	publicAddresses, err := cs.Address.ListPublicIpAddresses(p)
	if err != nil {
		return err
	} else if publicAddresses.Count > 0 {
		csCluster.Spec.ControlPlaneEndpoint.Host = publicAddresses.PublicIpAddresses[0].Ipaddress
		csCluster.Status.PublicIPID = publicAddresses.PublicIpAddresses[0].Id
	} else {
		return errors.New("no public addresses found")
	}
	return nil
}

// Gets a PublicIP and associates it.
func AssociatePublicIpAddress(cs *cloudstack.CloudStackClient, csCluster *infrav1.CloudStackCluster) (retErr error) {
	if err := FetchPublicIP(cs, csCluster); err != nil {
		return err
	}

	if publicAddress, cnt, retErr := cs.Address.GetPublicIpAddressByID(csCluster.Status.PublicIPID); retErr != nil {
		return retErr
	} else if cnt != 1 { // Should probably never happen... Probably.
		return errors.New(fmt.Sprintf("Expected exactly one Public IP for ID %s", csCluster.Status.PublicIPID))
	} else if publicAddress.Allocated != "" && publicAddress.Associatednetworkid == csCluster.Status.NetworkID {
		// Address already allocated to network. Allocated is a timestamp -- not a boolean.
		return nil
	} // Address not yet allocated. Allocate now.

	// Public IP found, but not yet allocated to network.
	p := cs.Address.NewAssociateIpAddressParams()
	p.SetNetworkid(csCluster.Status.NetworkID)
	p.SetIpaddress(csCluster.Spec.ControlPlaneEndpoint.Host)
	if _, err := cs.Address.AssociateIpAddress(p); err != nil {
		return err
	}
	return nil
}

func OpenFirewallRules(cs *cloudstack.CloudStackClient, csCluster *infrav1.CloudStackCluster) (retErr error) {
	p := cs.Firewall.NewCreateEgressFirewallRuleParams(csCluster.Status.NetworkID, "tcp")
	_, retErr = cs.Firewall.CreateEgressFirewallRule(p)
	if strings.Contains(retErr.Error(), "There is already") {
		retErr = nil
	}
	return retErr
}

func FetchLoadBalancerRule(cs *cloudstack.CloudStackClient, csCluster *infrav1.CloudStackCluster) (retErr error) {
	// TODO make port configurable.
	port := 6443
	p := cs.LoadBalancer.NewListLoadBalancerRulesParams()
	p.SetPublicipid(csCluster.Status.PublicIPID)
	loadBalancerRules, err := cs.LoadBalancer.ListLoadBalancerRules(p)
	if err != nil {
		return err
	}
	for _, rule := range loadBalancerRules.LoadBalancerRules {
		if rule.Publicport == strconv.Itoa(port) {
			csCluster.Status.LBRuleID = rule.Id
			return nil
		}
	}
	return errors.New("no load balancer rule found")
}

// Create a load balancer rule that can be assigned to instances.
func CreateLoadBalancerRule(cs *cloudstack.CloudStackClient, csCluster *infrav1.CloudStackCluster) (retErr error) {
	port := 6443
	// Check if rule exists.
	if err := FetchLoadBalancerRule(cs, csCluster); err == nil ||
		!strings.Contains(err.Error(), "no load balancer rule found") {
		return err
	}

	p := cs.LoadBalancer.NewCreateLoadBalancerRuleParams("roundrobin", "Kubernetes_API_Server", port, port)
	fmt.Println(csCluster.Status)
	fmt.Println(csCluster.Status.PublicIPID)
	p.SetPublicipid(csCluster.Status.PublicIPID)
	p.SetPublicipid(csCluster.Status.PublicIPID)
	p.SetProtocol("tcp")
	resp, err := cs.LoadBalancer.CreateLoadBalancerRule(p)
	if err != nil {
		return err
	}
	csCluster.Status.LBRuleID = resp.Id
	return nil
}

func DestroyNetwork(cs *cloudstack.CloudStackClient, csCluster *infrav1.CloudStackCluster) (retErr error) {
	cs.Network.DeleteNetwork(cs.Network.NewDeleteNetworkParams(csCluster.Status.NetworkID))
	return retErr
}

func AssignVMToLoadBalancerRule(
	cs *cloudstack.CloudStackClient,
	csCluster *infrav1.CloudStackCluster,
	instanceId string) (retErr error) {
	p := cs.LoadBalancer.NewAssignToLoadBalancerRuleParams(csCluster.Status.LBRuleID)
	p.SetVirtualmachineids([]string{instanceId})
	_, retErr = cs.LoadBalancer.AssignToLoadBalancerRule(p)
	return retErr
}
