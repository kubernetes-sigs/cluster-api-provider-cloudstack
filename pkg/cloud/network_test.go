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
	"fmt"

	csapi "github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Network", func() {
	var ( // Declare shared vars.
		mockCtrl   *gomock.Controller
		mockClient *csapi.CloudStackClient
		ns         *csapi.MockNetworkServiceIface
		// nos        *csapi.MockNetworkOfferingServiceIface
		// fs         *csapi.MockFirewallServiceIface
		// as         *csapi.MockAddressServiceIface
		// lbs        *csapi.MockLoadBalancerServiceIface
		// rs     *csapi.MockResourcetagsServiceIface
		client cloud.Client
	)

	// const (
	// 	dummies.Net1.ID           = "dummies.Net1.ID"
	// 	dummies.Net1.Name         = "dummies.Net1.Name"
	// 	isolatedNetworkType = "Isolated"
	// 	lbRuleID            = "lbRuleID"
	// 	netID               = "someNetID"
	// 	protocol            = "tcp"
	// 	publicPort          = "6443"
	// )

	BeforeEach(func() {
		// Setup new mock services.
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = csapi.NewMockClient(mockCtrl)
		ns = mockClient.Network.(*csapi.MockNetworkServiceIface)
		// nos = mockClient.NetworkOffering.(*csapi.MockNetworkOfferingServiceIface)
		// fs = mockClient.Firewall.(*csapi.MockFirewallServiceIface)
		// as = mockClient.Address.(*csapi.MockAddressServiceIface)
		// lbs = mockClient.LoadBalancer.(*csapi.MockLoadBalancerServiceIface)
		// rs = mockClient.Resourcetags.(*csapi.MockResourcetagsServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
		dummies.SetDummyVars()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	// // Sets expectations network tag creation.  To be used by tests that get/create networks.
	// expectNetworkTags := func(networkID string) {
	// 	listTagsParams := &csapi.ListTagsParams{}
	// 	createTagsParams := &csapi.CreateTagsParams{}
	// 	rs.EXPECT().NewListTagsParams().Return(listTagsParams)
	// 	rs.EXPECT().ListTags(listTagsParams).Return(&csapi.ListTagsResponse{}, nil)
	// 	rs.EXPECT().NewCreateTagsParams([]string{networkID}, string(cloud.ResourceTypeNetwork), gomock.Any()).Return(createTagsParams)
	// 	rs.EXPECT().CreateTags(createTagsParams).Return(&csapi.CreateTagsResponse{}, nil)
	// }

	Context("for an existing network", func() {
		It("resolves network by ID", func() {
			ns.EXPECT().GetNetworkByName(dummies.ISONet1.Name).Return(nil, 0, nil)
			ns.EXPECT().GetNetworkByID(dummies.ISONet1.ID).Return(dummies.CAPCNetToCSAPINet(&dummies.ISONet1), 1, nil)

			Ω(client.ResolveNetwork(dummies.CSCluster, &dummies.ISONet1)).Should(Succeed())
		})

		It("resolves network by Name", func() {
			ns.EXPECT().GetNetworkByName(dummies.ISONet1.Name).Return(dummies.CAPCNetToCSAPINet(&dummies.ISONet1), 1, nil)

			Ω(client.ResolveNetwork(dummies.CSCluster, &dummies.ISONet1)).Should(Succeed())
		})
		It("resolves network details in cluster status", func() {
			// Gets Net1 by Name.
			ns.EXPECT().GetNetworkByName(dummies.Net1.Name).Return(dummies.CAPCNetToCSAPINet(&dummies.Net1), 1, nil)

			// Trys to get Net2 by name and doesn't find it. Then finds Net2 via ID.
			ns.EXPECT().GetNetworkByName(dummies.Net2.Name).Return(nil, 0, nil)
			ns.EXPECT().GetNetworkByID(dummies.Net2.ID).Return(dummies.CAPCNetToCSAPINet(&dummies.Net2), 1, nil)
			fmt.Println(dummies.CSCluster.Spec.Zones)

			Ω(client.ResolveNetworkStatuses(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.Zones[dummies.Zone1.ID].Network).Should(Equal(dummies.Net1))
			Ω(dummies.CSCluster.Status.Zones[dummies.Zone2.ID].Network).Should(Equal(dummies.Net2))
		})
	})

	// Isolated use case only!
	// Context("for a non-existent network", func() {
	// 	It("when GetOrCreateNetwork is called it calls CloudStack to create a network", func() {
	// 		ns.EXPECT().GetNetworkID(gomock.Any()).Return("", -1, errors.New("no match found for blah"))
	// 		ns.EXPECT().GetNetworkByID(gomock.Any()).Return(nil, -1, errors.New("no match found for blah"))
	// 		nos.EXPECT().GetNetworkOfferingID(gomock.Any()).Return("someOfferingID", 1, nil)
	// 		ns.EXPECT().NewCreateNetworkParams(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
	// 			Return(&csapi.CreateNetworkParams{})
	// 		ns.EXPECT().CreateNetwork(gomock.Any()).Return(&csapi.CreateNetworkResponse{Id: dummies.Net1.ID}, nil)

	// 		expectNetworkTags(dummies.Net1.ID)

	// 		Ω(client.ResolveNetworkStatuses(dummies.CSCluster)).Should(Succeed())
	// 	})
	// })

	// 	Context("for a closed firewall", func() {
	// 		It("OpenFirewallRule asks CloudStack to open the firewall", func() {
	// 			fs.EXPECT().NewCreateEgressFirewallRuleParams(netID, protocol).
	// 				Return(&csapi.CreateEgressFirewallRuleParams{})
	// 			fs.EXPECT().CreateEgressFirewallRule(&csapi.CreateEgressFirewallRuleParams{}).
	// 				Return(&csapi.CreateEgressFirewallRuleResponse{}, nil)

	// 			Ω(client.OpenFirewallRules(csCluster)).Should(Succeed())
	// 		})
	// 	})

	// 	Context("for an open firewall", func() {
	// 		It("OpenFirewallRule asks CloudStack to open the firewall anyway, but doesn't fail", func() {
	// 			fs.EXPECT().NewCreateEgressFirewallRuleParams(netID, protocol).
	// 				Return(&csapi.CreateEgressFirewallRuleParams{})
	// 			fs.EXPECT().CreateEgressFirewallRule(&csapi.CreateEgressFirewallRuleParams{}).
	// 				Return(&csapi.CreateEgressFirewallRuleResponse{}, errors.New("there is already a rule like this"))
	// 			Ω(client.OpenFirewallRules(csCluster)).Should(Succeed())
	// 		})
	// 	})

	// 	Context("in an isolated network with public IPs available", func() {
	// 		It("will resolve public IP details given an endpoint spec", func() {
	// 			ipAddress := "192.168.1.14"
	// 			as.EXPECT().NewListPublicIpAddressesParams().Return(&csapi.ListPublicIpAddressesParams{})
	// 			as.EXPECT().ListPublicIpAddresses(gomock.Any()).
	// 				Return(&csapi.ListPublicIpAddressesResponse{
	// 					Count:             1,
	// 					PublicIpAddresses: []*csapi.PublicIpAddress{{Id: "PublicIPID", Ipaddress: ipAddress}},
	// 				}, nil)
	// 			publicIPAddress, err := client.ResolvePublicIPDetails(csCluster)
	// 			Ω(err).Should(Succeed())
	// 			Ω(publicIPAddress).ShouldNot(BeNil())
	// 			Ω(publicIPAddress.Ipaddress).Should(Equal(ipAddress))
	// 		})
	// 	})

	// 	Context("The specific load balancer rule does exist", func() {
	// 		It("resolves the rule's ID", func() {
	// 			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
	// 			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).Return(
	// 				&csapi.ListLoadBalancerRulesResponse{
	// 					LoadBalancerRules: []*csapi.LoadBalancerRule{{Publicport: publicPort, Id: lbRuleID}}}, nil)
	// 			Ω(client.ResolveLoadBalancerRuleDetails(csCluster)).Should(Succeed())
	// 			Ω(csCluster.Status.LBRuleID).Should(Equal(lbRuleID))
	// 		})

	// 		It("doesn't create a new load balancer rule on create", func() {
	// 			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
	// 			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).
	// 				Return(&csapi.ListLoadBalancerRulesResponse{
	// 					LoadBalancerRules: []*csapi.LoadBalancerRule{{Publicport: publicPort, Id: lbRuleID}}}, nil)
	// 			Ω(client.GetOrCreateLoadBalancerRule(csCluster)).Should(Succeed())
	// 			Ω(csCluster.Status.LBRuleID).Should(Equal(lbRuleID))
	// 		})
	// 	})

	// 	Context("load balancer rule does not exist", func() {
	// 		It("calls cloudstack to create a new load balancer rule.", func() {
	// 			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
	// 			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).Return(&csapi.ListLoadBalancerRulesResponse{
	// 				LoadBalancerRules: []*csapi.LoadBalancerRule{{Publicport: "7443", Id: lbRuleID}}}, nil)
	// 			lbs.EXPECT().NewCreateLoadBalancerRuleParams(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
	// 				Return(&csapi.CreateLoadBalancerRuleParams{})
	// 			const randomID = "randomID"
	// 			lbs.EXPECT().CreateLoadBalancerRule(gomock.Any()).
	// 				Return(&csapi.CreateLoadBalancerRuleResponse{Id: randomID}, nil)
	// 			Ω(client.GetOrCreateLoadBalancerRule(csCluster)).Should(Succeed())
	// 			Ω(csCluster.Status.LBRuleID).Should(Equal(randomID))
	// 		})
	// 	})
})
