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

package cloud_test

import (
	"strconv"

	csapi "github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
)

var _ = Describe("Network", func() {

	const (
		ipAddress    = "192.168.1.14"
		errorMessage = "Error"
	)

	fakeError := errors.New(errorMessage)
	var ( // Declare shared vars.
		mockCtrl   *gomock.Controller
		mockClient *csapi.CloudStackClient
		ns         *csapi.MockNetworkServiceIface
		nos        *csapi.MockNetworkOfferingServiceIface
		fs         *csapi.MockFirewallServiceIface
		as         *csapi.MockAddressServiceIface
		lbs        *csapi.MockLoadBalancerServiceIface
		rs         *csapi.MockResourcetagsServiceIface
		client     cloud.Client
	)

	BeforeEach(func() {
		// Setup new mock services.
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = csapi.NewMockClient(mockCtrl)
		ns = mockClient.Network.(*csapi.MockNetworkServiceIface)
		nos = mockClient.NetworkOffering.(*csapi.MockNetworkOfferingServiceIface)
		fs = mockClient.Firewall.(*csapi.MockFirewallServiceIface)
		as = mockClient.Address.(*csapi.MockAddressServiceIface)
		lbs = mockClient.LoadBalancer.(*csapi.MockLoadBalancerServiceIface)
		rs = mockClient.Resourcetags.(*csapi.MockResourcetagsServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient, nil)
		dummies.SetDummyVars()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("Get or Create Isolated network in CloudStack", func() {
		It("calls to create an isolated network when not found", func() {
			dummies.Zone1.Network = dummies.ISONet1
			dummies.Zone1.Network.ID = ""

			nos.EXPECT().GetNetworkOfferingID(gomock.Any()).Return("someOfferingID", 1, nil)
			ns.EXPECT().NewCreateNetworkParams(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&csapi.CreateNetworkParams{})
			ns.EXPECT().GetNetworkByName(dummies.ISONet1.Name).Return(nil, 0, nil)
			ns.EXPECT().GetNetworkByID(dummies.ISONet1.ID).Return(nil, 0, nil)
			ns.EXPECT().CreateNetwork(gomock.Any()).Return(&csapi.CreateNetworkResponse{Id: dummies.ISONet1.ID}, nil)
			as.EXPECT().NewListPublicIpAddressesParams().Return(&csapi.ListPublicIpAddressesParams{})
			as.EXPECT().ListPublicIpAddresses(gomock.Any()).
				Return(&csapi.ListPublicIpAddressesResponse{
					Count:             1,
					PublicIpAddresses: []*csapi.PublicIpAddress{{Id: dummies.PublicIPID, Ipaddress: "fakeIP"}}}, nil)
			as.EXPECT().NewAssociateIpAddressParams().Return(&csapi.AssociateIpAddressParams{})
			as.EXPECT().AssociateIpAddress(gomock.Any())
			fs.EXPECT().NewCreateEgressFirewallRuleParams(dummies.ISONet1.ID, gomock.Any()).
				DoAndReturn(func(networkid string, protocol string) *csapi.CreateEgressFirewallRuleParams {
					p := &csapi.CreateEgressFirewallRuleParams{}
					if protocol == "icmp" {
						p.SetIcmptype(-1)
						p.SetIcmpcode(-1)
					}
					return p
				}).Times(3)

			ruleParamsICMP := &csapi.CreateEgressFirewallRuleParams{}
			ruleParamsICMP.SetIcmptype(-1)
			ruleParamsICMP.SetIcmpcode(-1)
			gomock.InOrder(
				fs.EXPECT().CreateEgressFirewallRule(&csapi.CreateEgressFirewallRuleParams{}).
					Return(&csapi.CreateEgressFirewallRuleResponse{}, nil).Times(2),
				fs.EXPECT().CreateEgressFirewallRule(ruleParamsICMP).
					Return(&csapi.CreateEgressFirewallRuleResponse{}, nil))

			// Will add cluster tag once to Network and once to PublicIP.
			createdByResponse := &csapi.ListTagsResponse{Tags: []*csapi.Tag{{Key: cloud.CreatedByCAPCTagName, Value: "1"}}}
			gomock.InOrder(
				rs.EXPECT().NewListTagsParams().Return(&csapi.ListTagsParams{}),
				rs.EXPECT().ListTags(gomock.Any()).Return(createdByResponse, nil),
				rs.EXPECT().NewListTagsParams().Return(&csapi.ListTagsParams{}),
				rs.EXPECT().ListTags(gomock.Any()).Return(createdByResponse, nil))

			// Will add creation and cluster tags to network and PublicIP.
			rs.EXPECT().NewCreateTagsParams(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&csapi.CreateTagsParams{}).Times(4)
			rs.EXPECT().CreateTags(gomock.Any()).Return(&csapi.CreateTagsResponse{}, nil).Times(4)

			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).Return(
				&csapi.ListLoadBalancerRulesResponse{LoadBalancerRules: []*csapi.LoadBalancerRule{
					{Publicport: strconv.Itoa(int(dummies.EndPointPort)), Id: dummies.LBRuleID}}}, nil)

			Ω(client.GetOrCreateIsolatedNetwork(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
		})

		It("fails to get network offering from CloudStack", func() {
			ns.EXPECT().GetNetworkByName(dummies.ISONet1.Name).Return(nil, 0, nil)
			ns.EXPECT().GetNetworkByID(dummies.ISONet1.ID).Return(nil, 0, nil)
			nos.EXPECT().GetNetworkOfferingID(gomock.Any()).Return("", -1, fakeError)

			err := client.GetOrCreateIsolatedNetwork(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)
			Ω(err).ShouldNot(Succeed())
			Ω(err.Error()).Should(ContainSubstring("creating a new isolated network"))
		})
	})

	Context("for a closed firewall", func() {
		It("OpenFirewallRule asks CloudStack to open the firewall", func() {
			dummies.Zone1.Network = dummies.ISONet1
			fs.EXPECT().NewCreateEgressFirewallRuleParams(dummies.ISONet1.ID, gomock.Any()).
				DoAndReturn(func(networkid string, protocol string) *csapi.CreateEgressFirewallRuleParams {
					p := &csapi.CreateEgressFirewallRuleParams{}
					if protocol == "icmp" {
						p.SetIcmptype(-1)
						p.SetIcmpcode(-1)
					}
					return p
				}).Times(3)

			ruleParamsICMP := &csapi.CreateEgressFirewallRuleParams{}
			ruleParamsICMP.SetIcmptype(-1)
			ruleParamsICMP.SetIcmpcode(-1)
			gomock.InOrder(
				fs.EXPECT().CreateEgressFirewallRule(&csapi.CreateEgressFirewallRuleParams{}).
					Return(&csapi.CreateEgressFirewallRuleResponse{}, nil).Times(2),
				fs.EXPECT().CreateEgressFirewallRule(ruleParamsICMP).
					Return(&csapi.CreateEgressFirewallRuleResponse{}, nil))

			Ω(client.OpenFirewallRules(dummies.CSISONet1)).Should(Succeed())
		})
	})

	Context("for an open firewall", func() {
		It("OpenFirewallRule asks CloudStack to open the firewall anyway, but doesn't fail", func() {
			dummies.Zone1.Network = dummies.ISONet1

			fs.EXPECT().NewCreateEgressFirewallRuleParams(dummies.ISONet1.ID, gomock.Any()).
				DoAndReturn(func(networkid string, protocol string) *csapi.CreateEgressFirewallRuleParams {
					p := &csapi.CreateEgressFirewallRuleParams{}
					if protocol == "icmp" {
						p.SetIcmptype(-1)
						p.SetIcmpcode(-1)
					}
					return p
				}).Times(3)

			ruleParamsICMP := &csapi.CreateEgressFirewallRuleParams{}
			ruleParamsICMP.SetIcmptype(-1)
			ruleParamsICMP.SetIcmpcode(-1)
			gomock.InOrder(
				fs.EXPECT().CreateEgressFirewallRule(&csapi.CreateEgressFirewallRuleParams{}).
					Return(&csapi.CreateEgressFirewallRuleResponse{}, nil).Times(2),
				fs.EXPECT().CreateEgressFirewallRule(ruleParamsICMP).
					Return(&csapi.CreateEgressFirewallRuleResponse{}, nil))

			Ω(client.OpenFirewallRules(dummies.CSISONet1)).Should(Succeed())
		})
	})

	Context("in an isolated network with public IPs available", func() {
		It("will resolve public IP details given an endpoint spec", func() {
			as.EXPECT().NewListPublicIpAddressesParams().Return(&csapi.ListPublicIpAddressesParams{})
			as.EXPECT().ListPublicIpAddresses(gomock.Any()).
				Return(&csapi.ListPublicIpAddressesResponse{
					Count:             1,
					PublicIpAddresses: []*csapi.PublicIpAddress{{Id: "PublicIPID", Ipaddress: ipAddress}},
				}, nil)
			publicIPAddress, err := client.GetPublicIP(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)
			Ω(err).Should(Succeed())
			Ω(publicIPAddress).ShouldNot(BeNil())
			Ω(publicIPAddress.Ipaddress).Should(Equal(ipAddress))
		})
	})

	Context("In an isolated network with all public IPs allocated", func() {
		It("No public IP addresses available", func() {
			as.EXPECT().NewListPublicIpAddressesParams().Return(&csapi.ListPublicIpAddressesParams{})
			as.EXPECT().ListPublicIpAddresses(gomock.Any()).
				Return(&csapi.ListPublicIpAddressesResponse{
					Count:             0,
					PublicIpAddresses: []*csapi.PublicIpAddress{},
				}, nil)
			publicIPAddress, err := client.GetPublicIP(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)
			Ω(publicIPAddress).Should(BeNil())
			Ω(err.Error()).Should(ContainSubstring("no public addresses found in available networks"))
		})

		It("All Public IPs allocated", func() {
			as.EXPECT().NewListPublicIpAddressesParams().Return(&csapi.ListPublicIpAddressesParams{})
			as.EXPECT().ListPublicIpAddresses(gomock.Any()).
				Return(&csapi.ListPublicIpAddressesResponse{
					Count: 2,
					PublicIpAddresses: []*csapi.PublicIpAddress{{
						State:               "Allocated",
						Allocated:           "true",
						Associatednetworkid: "1",
					},
						{
							State:               "Allocated",
							Allocated:           "true",
							Associatednetworkid: "1",
						}},
				}, nil)
			publicIPAddress, err := client.GetPublicIP(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)
			Ω(publicIPAddress).Should(BeNil())
			Ω(err.Error()).Should(ContainSubstring("all Public IP Address(es) found were already allocated"))
		})
	})

	Context("Associate Public IP address to Network", func() {
		It("Successfully Associated Public IP to provided isolated network", func() {
			as.EXPECT().NewListPublicIpAddressesParams().Return(&csapi.ListPublicIpAddressesParams{})
			as.EXPECT().ListPublicIpAddresses(gomock.Any()).
				Return(&csapi.ListPublicIpAddressesResponse{
					Count:             1,
					PublicIpAddresses: []*csapi.PublicIpAddress{{Id: "PublicIPID", Ipaddress: ipAddress}},
				}, nil)
			aip := &csapi.AssociateIpAddressParams{}
			as.EXPECT().NewAssociateIpAddressParams().Return(aip)
			as.EXPECT().AssociateIpAddress(aip).Return(&csapi.AssociateIpAddressResponse{}, nil)
			// Will add cluster tag once to Network and once to PublicIP.
			createdByResponse := &csapi.ListTagsResponse{Tags: []*csapi.Tag{{Key: cloud.CreatedByCAPCTagName, Value: "1"}}}
			gomock.InOrder(
				rs.EXPECT().NewListTagsParams().Return(&csapi.ListTagsParams{}),
				rs.EXPECT().ListTags(gomock.Any()).Return(createdByResponse, nil))

			// Will add creation and cluster tags to network and PublicIP.
			rs.EXPECT().NewCreateTagsParams(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&csapi.CreateTagsParams{}).Times(2)
			rs.EXPECT().CreateTags(gomock.Any()).Return(&csapi.CreateTagsResponse{}, nil).Times(2)

			Ω(client.AssociatePublicIPAddress(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
		})

		It("Failure Associating Public IP to Isolated network", func() {
			as.EXPECT().NewListPublicIpAddressesParams().Return(&csapi.ListPublicIpAddressesParams{})
			as.EXPECT().ListPublicIpAddresses(gomock.Any()).
				Return(&csapi.ListPublicIpAddressesResponse{
					Count:             1,
					PublicIpAddresses: []*csapi.PublicIpAddress{{Id: "PublicIPID", Ipaddress: ipAddress}},
				}, nil)
			aip := &csapi.AssociateIpAddressParams{}
			as.EXPECT().NewAssociateIpAddressParams().Return(aip)
			as.EXPECT().AssociateIpAddress(aip).Return(nil, errors.New("Failed to allocate IP address"))
			Ω(client.AssociatePublicIPAddress(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster).Error()).Should(ContainSubstring("associating public IP address with ID"))
		})
	})

	Context("The specific load balancer rule does exist", func() {
		It("resolves the rule's ID", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).Return(
				&csapi.ListLoadBalancerRulesResponse{LoadBalancerRules: []*csapi.LoadBalancerRule{
					{Publicport: strconv.Itoa(int(dummies.EndPointPort)), Id: dummies.LBRuleID}}}, nil)

			dummies.CSISONet1.Status.LBRuleID = ""
			Ω(client.ResolveLoadBalancerRuleDetails(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSISONet1.Status.LBRuleID).Should(Equal(dummies.LBRuleID))
		})

		It("Failed to resolve LB rule details", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).Return(
				&csapi.ListLoadBalancerRulesResponse{LoadBalancerRules: []*csapi.LoadBalancerRule{
					{Publicport: "differentPublicPort", Id: dummies.LBRuleID}}}, nil)

			dummies.CSISONet1.Status.LBRuleID = ""
			Ω(client.ResolveLoadBalancerRuleDetails(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster).Error()).
				Should(Equal("no load balancer rule found"))
		})

		It("Failed to list LB rules", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).Return(
				nil, fakeError)

			dummies.CSISONet1.Status.LBRuleID = ""
			Ω(client.ResolveLoadBalancerRuleDetails(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster).Error()).
				Should(ContainSubstring("listing load balancer rules"))
		})

		It("doesn't create a new load balancer rule on create", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).
				Return(&csapi.ListLoadBalancerRulesResponse{
					LoadBalancerRules: []*csapi.LoadBalancerRule{
						{Publicport: strconv.Itoa(int(dummies.EndPointPort)), Id: dummies.LBRuleID}}}, nil)

			Ω(client.GetOrCreateLoadBalancerRule(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSISONet1.Status.LBRuleID).Should(Equal(dummies.LBRuleID))
		})
	})

	Context("Assign VM to Load Balancer rule", func() {
		It("Associates VM to LB rule", func() {
			dummies.CSISONet1.Status.LBRuleID = "lbruleid"
			lbip := &csapi.ListLoadBalancerRuleInstancesParams{}
			albp := &csapi.AssignToLoadBalancerRuleParams{}
			lbs.EXPECT().NewListLoadBalancerRuleInstancesParams(dummies.CSISONet1.Status.LBRuleID).
				Return(lbip)
			lbs.EXPECT().ListLoadBalancerRuleInstances(lbip).Return(&csapi.ListLoadBalancerRuleInstancesResponse{}, nil)
			lbs.EXPECT().NewAssignToLoadBalancerRuleParams(dummies.CSISONet1.Status.LBRuleID).Return(albp)
			lbs.EXPECT().AssignToLoadBalancerRule(albp).Return(&csapi.AssignToLoadBalancerRuleResponse{}, nil)

			Ω(client.AssignVMToLoadBalancerRule(dummies.CSISONet1, *dummies.CSMachine1.Spec.InstanceID)).Should(Succeed())
		})

		It("Associating VM to LB rule fails", func() {
			dummies.CSISONet1.Status.LBRuleID = "lbruleid"
			lbip := &csapi.ListLoadBalancerRuleInstancesParams{}
			albp := &csapi.AssignToLoadBalancerRuleParams{}
			lbs.EXPECT().NewListLoadBalancerRuleInstancesParams(dummies.CSISONet1.Status.LBRuleID).
				Return(lbip)
			lbs.EXPECT().ListLoadBalancerRuleInstances(lbip).Return(&csapi.ListLoadBalancerRuleInstancesResponse{}, nil)
			lbs.EXPECT().NewAssignToLoadBalancerRuleParams(dummies.CSISONet1.Status.LBRuleID).Return(albp)
			lbs.EXPECT().AssignToLoadBalancerRule(albp).Return(nil, fakeError)

			Ω(client.AssignVMToLoadBalancerRule(dummies.CSISONet1, *dummies.CSMachine1.Spec.InstanceID)).ShouldNot(Succeed())
		})

		It("LB Rule already assigned to VM", func() {
			dummies.CSISONet1.Status.LBRuleID = "lbruleid"
			lbip := &csapi.ListLoadBalancerRuleInstancesParams{}
			lbs.EXPECT().NewListLoadBalancerRuleInstancesParams(dummies.CSISONet1.Status.LBRuleID).
				Return(lbip)
			lbs.EXPECT().ListLoadBalancerRuleInstances(lbip).Return(&csapi.ListLoadBalancerRuleInstancesResponse{
				Count: 1,
				LoadBalancerRuleInstances: []*csapi.VirtualMachine{{
					Id: *dummies.CSMachine1.Spec.InstanceID,
				}},
			}, nil)

			Ω(client.AssignVMToLoadBalancerRule(dummies.CSISONet1, *dummies.CSMachine1.Spec.InstanceID)).Should(Succeed())
		})
	})

	Context("load balancer rule does not exist", func() {
		It("calls cloudstack to create a new load balancer rule.", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).
				Return(&csapi.ListLoadBalancerRulesResponse{
					LoadBalancerRules: []*csapi.LoadBalancerRule{{Publicport: "7443", Id: dummies.LBRuleID}}}, nil)
			lbs.EXPECT().NewCreateLoadBalancerRuleParams(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&csapi.CreateLoadBalancerRuleParams{})
			lbs.EXPECT().CreateLoadBalancerRule(gomock.Any()).
				Return(&csapi.CreateLoadBalancerRuleResponse{Id: "2ndLBRuleID"}, nil)

			Ω(client.GetOrCreateLoadBalancerRule(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSISONet1.Status.LBRuleID).Should(Equal("2ndLBRuleID"))
		})

		It("Fails to resolve load balancer rule details", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).
				Return(nil, fakeError)
			err := client.GetOrCreateLoadBalancerRule(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)
			Ω(err).ShouldNot(Succeed())
			Ω(err.Error()).Should(ContainSubstring(errorMessage))
		})

		It("Fails to create a new load balancer rule.", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).
				Return(&csapi.ListLoadBalancerRulesResponse{
					LoadBalancerRules: []*csapi.LoadBalancerRule{{Publicport: "7443", Id: dummies.LBRuleID}}}, nil)
			lbs.EXPECT().NewCreateLoadBalancerRuleParams(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&csapi.CreateLoadBalancerRuleParams{})
			lbs.EXPECT().CreateLoadBalancerRule(gomock.Any()).
				Return(nil, fakeError)
			err := client.GetOrCreateLoadBalancerRule(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)
			Ω(err).ShouldNot(Succeed())
			Ω(err.Error()).Should(Equal(errorMessage))

		})
	})

	Context("Delete Network", func() {
		It("Calls CloudStack to delete network", func() {
			dnp := &csapi.DeleteNetworkParams{}
			ns.EXPECT().NewDeleteNetworkParams(dummies.ISONet1.ID).Return(dnp)
			ns.EXPECT().DeleteNetwork(dnp).Return(&csapi.DeleteNetworkResponse{}, nil)

			Ω(client.DeleteNetwork(dummies.ISONet1)).Should(Succeed())
		})

		It("Network deletion failure", func() {
			dnp := &csapi.DeleteNetworkParams{}
			ns.EXPECT().NewDeleteNetworkParams(dummies.ISONet1.ID).Return(dnp)
			ns.EXPECT().DeleteNetwork(dnp).Return(nil, fakeError)
			err := client.DeleteNetwork(dummies.ISONet1)
			Ω(err).ShouldNot(Succeed())
			Ω(err.Error()).Should(ContainSubstring("deleting network with id " + dummies.ISONet1.ID))
		})
	})

	Context("Dispose or cleanup isolate network resources", func() {
		It("delete all isolated network resources when not managed by CAPC", func() {
			dummies.CSISONet1.Status.PublicIPID = "publicIpId"
			rtlp := &csapi.ListTagsParams{}
			rs.EXPECT().NewListTagsParams().Return(rtlp).Times(4)
			rs.EXPECT().ListTags(rtlp).Return(&csapi.ListTagsResponse{}, nil).Times(4)
			as.EXPECT().GetPublicIpAddressByID(dummies.CSISONet1.Status.PublicIPID).Return(&csapi.PublicIpAddress{}, 1, nil)

			Ω(client.DisposeIsoNetResources(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
		})

		It("delete all isolated network resources when managed by CAPC", func() {
			dummies.CSISONet1.Status.PublicIPID = "publicIpId"
			rtdp := &csapi.DeleteTagsParams{}
			rtlp := &csapi.ListTagsParams{}
			dap := &csapi.DisassociateIpAddressParams{}
			createdByCAPCResponse := &csapi.ListTagsResponse{Tags: []*csapi.Tag{{Key: cloud.CreatedByCAPCTagName, Value: "1"}}}
			rs.EXPECT().NewDeleteTagsParams(gomock.Any(), gomock.Any()).Return(rtdp).Times(2)
			rs.EXPECT().DeleteTags(rtdp).Return(&csapi.DeleteTagsResponse{}, nil).Times(2)
			rs.EXPECT().NewListTagsParams().Return(rtlp).Times(4)
			rs.EXPECT().ListTags(rtlp).Return(createdByCAPCResponse, nil).Times(3)
			rs.EXPECT().ListTags(rtlp).Return(&csapi.ListTagsResponse{}, nil).Times(1)
			as.EXPECT().GetPublicIpAddressByID(dummies.CSISONet1.Status.PublicIPID).Return(&csapi.PublicIpAddress{}, 1, nil)
			as.EXPECT().NewDisassociateIpAddressParams(dummies.CSISONet1.Status.PublicIPID).Return(dap)
			as.EXPECT().DisassociateIpAddress(dap).Return(&csapi.DisassociateIpAddressResponse{}, nil)

			Ω(client.DisposeIsoNetResources(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
		})

		It("disassociate IP address fails due to failure in deleting a resource i.e., disassociate Public IP", func() {
			dummies.CSISONet1.Status.PublicIPID = "publicIpId"
			rtdp := &csapi.DeleteTagsParams{}
			rtlp := &csapi.ListTagsParams{}
			dap := &csapi.DisassociateIpAddressParams{}
			createdByCAPCResponse := &csapi.ListTagsResponse{Tags: []*csapi.Tag{{Key: cloud.CreatedByCAPCTagName, Value: "1"}}}
			rs.EXPECT().NewDeleteTagsParams(gomock.Any(), gomock.Any()).Return(rtdp).Times(2)
			rs.EXPECT().DeleteTags(rtdp).Return(&csapi.DeleteTagsResponse{}, nil).Times(2)
			rs.EXPECT().NewListTagsParams().Return(rtlp).Times(2)
			rs.EXPECT().ListTags(rtlp).Return(createdByCAPCResponse, nil).Times(2)
			as.EXPECT().GetPublicIpAddressByID(dummies.CSISONet1.Status.PublicIPID).Return(&csapi.PublicIpAddress{}, 1, nil)
			as.EXPECT().NewDisassociateIpAddressParams(dummies.CSISONet1.Status.PublicIPID).Return(dap)
			as.EXPECT().DisassociateIpAddress(dap).Return(nil, fakeError)

			Ω(client.DisposeIsoNetResources(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).ShouldNot(Succeed())
		})

	})

	Context("Networking Integ Tests", Label("integ"), func() {
		BeforeEach(func() {
			client = realCloudClient
			// Delete any existing tags
			existingTags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.Net1.ID)
			if err != nil {
				Fail("Failed to get existing tags. Error: " + err.Error())
			}
			if len(existingTags) != 0 {
				err = client.DeleteTags(cloud.ResourceTypeNetwork, dummies.Net1.ID, existingTags)
				if err != nil {
					Fail("Failed to delete existing tags. Error: " + err.Error())
				}
			}
			dummies.SetDummyVars()

			// Setup Isolated Network Dummy Vars.
			dummies.CSISONet1.Spec.ID = ""                        // Make CAPC methods resolve this.
			dummies.CSCluster.Spec.ControlPlaneEndpoint.Host = "" // Make CAPC methods resolve this.
			dummies.CSFailureDomain1.Spec.Zone.ID = ""            // Make CAPC methods resolve this.

			FetchIntegTestResources()
		})

		It("fetches an isolated network", func() {
			dummies.SetDummyIsoNetToNameOnly()
			dummies.SetClusterSpecToNet(&dummies.ISONet1)

			Ω(client.ResolveNetwork(&dummies.ISONet1)).Should(Succeed())
			Ω(dummies.ISONet1.ID).ShouldNot(BeEmpty())
			Ω(dummies.ISONet1.Type).Should(Equal(cloud.NetworkTypeIsolated))
		})

		It("fetches a public IP", func() {
			dummies.Zone1.ID = ""
			dummies.SetDummyIsoNetToNameOnly()
			dummies.SetClusterSpecToNet(&dummies.ISONet1)
			dummies.CSCluster.Spec.ControlPlaneEndpoint.Host = ""
			Ω(client.ResolveNetwork(&dummies.ISONet1)).Should(Succeed())
		})

		It("adds an isolated network and doesn't fail when asked to GetOrCreateIsolatedNetwork multiple times", func() {
			Ω(client.GetOrCreateIsolatedNetwork(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
			Ω(client.GetOrCreateIsolatedNetwork(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())

			// Network should now exist if it didn't at the start.
			Ω(client.ResolveNetwork(&dummies.ISONet1)).Should(Succeed())

			// Do once more.
			Ω(client.GetOrCreateIsolatedNetwork(dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
		})
	})
})
