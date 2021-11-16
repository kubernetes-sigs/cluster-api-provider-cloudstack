package cloud

// type Network struct {
// 	Cluster          *infrav1.CloudStackCluster
// 	CAPINetwork      *infrav1.CloudStackNetwork
// 	PublicIP         *cloudstack.PublicIpAddress
// 	LoadBalancerRule *cloudstack.LoadBalancerRule
// }

// func (N *Network) LBRuleFromLBRuleResponse(lbR *cloudstack.CreateLoadBalancerRuleResponse) {
// 	*N.LoadBalancerRule = cloudstack.LoadBalancerRule{
// 		Account:     lbR.Account,
// 		Algorithm:   lbR.Algorithm,
// 		Cidrlist:    lbR.Cidrlist,
// 		Description: lbR.Description,
// 		Domain:      lbR.Domain,
// 		Domainid:    lbR.Domainid,
// 		Fordisplay:  lbR.Fordisplay,
// 		Id:          lbR.Id,
// 		JobID:       lbR.JobID,
// 		Jobstatus:   lbR.Jobstatus,
// 		Name:        lbR.Name,
// 		Networkid:   lbR.Networkid,
// 		Privateport: lbR.Privateport,
// 		Project:     lbR.Project,
// 		Projectid:   lbR.Projectid,
// 		Protocol:    lbR.Protocol,
// 		Publicip:    lbR.Publicip,
// 		Publicipid:  lbR.Publicipid,
// 		Publicport:  lbR.Publicport,
// 		State:       lbR.State,
// 		Tags:        lbR.Tags,
// 		Zoneid:      lbR.Zoneid,
// 		Zonename:    lbR.Zonename,
// 	}
// }

// func (N *Network) Fetch(cs *cloudstack.CloudStackClient) error {
// 	network, count, err := cs.Network.GetNetworkByName(N.CAPINetwork.Spec.Name)
// 	if err != nil {
// 		return err
// 	} else if count != 1 {
// 		return errors.New("did not find network")
// 	}

// 	N.CAPINetwork.Status.FromCSNetwork(network)
// 	return nil
// }

// func (N *Network) Create(cs *cloudstack.CloudStackClient) error {
// 	if err := N.Fetch(cs); err != nil {
// 		return err
// 	}

// 	offeringId, count, err := cs.NetworkOffering.GetNetworkOfferingID("DefaultIsolatedNetworkOfferringWithSourceNatService")
// 	if err != nil {
// 		return err
// 	} else if count != 1 {
// 		return errors.New("found more than one network offering.")
// 	}
// 	p := cs.Network.NewCreateNetworkParams(N.CAPINetwork.Spec.Name, N.CAPINetwork.Spec.Name, offeringId, N.Cluster.Status.Zoneid)
// 	network, err := cs.Network.CreateNetwork(p)
// 	if err != nil {
// 		return err
// 	}
// 	N.CAPINetwork.Status.FromCSCreateNetworkRespone(network)

// 	return nil
// }

// func (N *Network) FetchPublicIP(cs *cloudstack.CloudStackClient) error {
// 	p := cs.Address.NewListPublicIpAddressesParams()
// 	// setIfNotEmpty(ipAddr, p.SetIpaddress)
// 	publicAddresses, err := cs.Address.ListPublicIpAddresses(p)
// 	if err != nil {
// 		return err
// 	} else if publicAddresses.Count > 0 {
// 		N.PublicIP = publicAddresses.PublicIpAddresses[0]
// 	} else {
// 		return errors.New("no public addresses found")
// 	}
// 	return nil
// }

// // Gets a PublicIP and associates it.
// func (N *Network) AssociatePublicIpAddress(cs *cloudstack.CloudStackClient) error {
// 	if err := N.FetchPublicIP(cs); err != nil {
// 		return err
// 	}
// 	p := cs.Address.NewAssociateIpAddressParams()
// 	p.SetIpaddress(N.PublicIP.Ipaddress)
// 	p.SetNetworkid(N.PublicIP.Networkid)
// 	if _, err := cs.Address.AssociateIpAddress(p); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (N *Network) OpenFirewallRules(cs *cloudstack.CloudStackClient) error {
// 	params := cs.Firewall.NewCreateEgressFirewallRuleParams(N.CAPINetwork.Status.Id, "tcp")
// 	_, err := cs.Firewall.CreateEgressFirewallRule(params)
// 	return err
// }

// func (N *Network) FetchLoadBalancerRule(cs *cloudstack.CloudStackClient) error {
// 	port := 6443
// 	p := cs.LoadBalancer.NewListLoadBalancerRulesParams()
// 	p.SetPublicipid(N.CAPINetwork.Status.Id)
// 	loadBalancerRules, err := cs.LoadBalancer.ListLoadBalancerRules(p)
// 	if err != nil {
// 		return err
// 	}
// 	for _, rule := range loadBalancerRules.LoadBalancerRules {
// 		if rule.Publicport == strconv.Itoa(port) {
// 			N.LoadBalancerRule = rule
// 			return nil
// 		}
// 	}
// 	return errors.New("no load balancer rule found")
// }

// // Create a load balancer rule that can be assigned to instances.
// func (N *Network) CreateLoadBalancerRule(cs *cloudstack.CloudStackClient) error {
// 	port := 6443
// 	// Check if rule exists.
// 	if err := N.FetchLoadBalancerRule(cs); err == nil || !strings.Contains(err.Error(), "no load balancer rule found") {
// 		return err
// 	}

// 	p := cs.LoadBalancer.NewCreateLoadBalancerRuleParams("roundrobin", "Kubernetes_API_Server", port, port)
// 	p.SetPublicipid(N.PublicIP.Id)
// 	p.SetProtocol("tcp")
// 	creationResp, err := cs.LoadBalancer.CreateLoadBalancerRule(p)
// 	if err != nil {
// 		return err
// 	}
// 	N.LBRuleFromLBRuleResponse(creationResp)
// 	return nil
// }

// func (N *Network) AssignVMToLoadBalancerRule(cs *cloudstack.CloudStackClient, instanceId string) error {
// 	p := cs.LoadBalancer.NewAssignToLoadBalancerRuleParams(N.LoadBalancerRule.Id)
// 	p.SetVirtualmachineids([]string{instanceId})
// 	_, err := cs.LoadBalancer.AssignToLoadBalancerRule(p)
// 	return err
// }
