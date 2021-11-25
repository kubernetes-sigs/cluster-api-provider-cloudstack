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

package cloud_test

import (
	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
	"gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/pkg/cloud"
)

var _ = Describe("Network", func() {
	var (
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	csCluster := &infrav1.CloudStackCluster{
		Spec: infrav1.CloudStackClusterSpec{
			Zone:    "zone1",
			Network: "GuestNet1",
		},
	}

	Context("for an existing network", func() {
		It("does fetch a network", func() {
			netName := "GuestNet1"
			mockClient.Network.(*cloudstack.MockNetworkServiceIface).
				EXPECT().GetNetworkID(netName).Return("someNetID", 1, nil)

			Ω(cloud.FetchNetwork(mockClient, csCluster)).Should(Succeed())
			Ω(csCluster.Status.NetworkID).Should(Equal("someNetID"))
		})

		It("does not call to create a new network", func() {
			netName := "GuestNet1"
			mockClient.Network.(*cloudstack.MockNetworkServiceIface).
				EXPECT().GetNetworkID(netName).Return("fakeNetID", 1, nil)

			cloud.CreateNetwork(mockClient, csCluster)
		})
	})

	Context("for a non existent network", func() {
		It("When CreateNetwork is called it calls CloudStack to create a network", func() {
			netName := "GuestNet1"
			mockClient.Network.(*cloudstack.MockNetworkServiceIface).EXPECT().
				GetNetworkID(netName).
				Return("", -1, errors.New("No match found for blah."))
			mockClient.NetworkOffering.(*cloudstack.MockNetworkOfferingServiceIface).EXPECT().
				GetNetworkOfferingID(gomock.Any()).
				Return("someOfferingID", 1, nil)
			mockClient.Network.(*cloudstack.MockNetworkServiceIface).EXPECT().
				NewCreateNetworkParams(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&cloudstack.CreateNetworkParams{})
			mockClient.Network.(*cloudstack.MockNetworkServiceIface).EXPECT().
				CreateNetwork(gomock.Any()).
				Return(&cloudstack.CreateNetworkResponse{Id: "someNetID"}, nil)

			Ω(cloud.CreateNetwork(mockClient, csCluster)).Should(Succeed())

		})
	})

	Context("for a closed firewall", func() {
		netID := "someNetID"
		csCluster.Status.NetworkID = netID
		It("asks CloudStack to open the firewall", func() {
			mockClient.Firewall.(*cloudstack.MockFirewallServiceIface).EXPECT().
				NewCreateEgressFirewallRuleParams(netID, "tcp").
				Return(&cloudstack.CreateEgressFirewallRuleParams{})
			mockClient.Firewall.(*cloudstack.MockFirewallServiceIface).EXPECT().
				CreateEgressFirewallRule(&cloudstack.CreateEgressFirewallRuleParams{}).
				Return(&cloudstack.CreateEgressFirewallRuleResponse{}, nil)

			Ω(cloud.OpenFirewallRules(mockClient, csCluster)).Should(Succeed())
		})
	})

	Context("for an open firewall", func() {
		netID := "someNetID"
		csCluster.Status.NetworkID = netID
		It("asks CloudStack to open the firewall anyway, but doesn't fail", func() {
			mockClient.Firewall.(*cloudstack.MockFirewallServiceIface).EXPECT().
				NewCreateEgressFirewallRuleParams(netID, "tcp").
				Return(&cloudstack.CreateEgressFirewallRuleParams{})
			mockClient.Firewall.(*cloudstack.MockFirewallServiceIface).EXPECT().
				CreateEgressFirewallRule(&cloudstack.CreateEgressFirewallRuleParams{}).
				Return(&cloudstack.CreateEgressFirewallRuleResponse{}, errors.New("There is already a rule like this."))

			Ω(cloud.OpenFirewallRules(mockClient, csCluster)).Should(Succeed())
		})
	})

	Context("in an isolated network with public IPs available", func() {
		It("will fetch a public IP given an endpoint spec", func() {
			mockClient.Address.(*cloudstack.MockAddressServiceIface).EXPECT().
				NewListPublicIpAddressesParams().
				Return(&cloudstack.ListPublicIpAddressesParams{})
			mockClient.Address.(*cloudstack.MockAddressServiceIface).EXPECT().
				ListPublicIpAddresses(gomock.Any()).
				Return(&cloudstack.ListPublicIpAddressesResponse{
					Count:             1,
					PublicIpAddresses: []*cloudstack.PublicIpAddress{{Id: "PublicIPID", Ipaddress: "192.168.1.14"}},
				}, nil)
			Ω(cloud.FetchPublicIP(mockClient, csCluster)).Should(Succeed())
		})
	})

	Context("The specific load balancer rule does exist", func() {
		csCluster.Spec.ControlPlaneEndpoint.Port = int32(6443)
		It("fetches the rule's ID", func() {
			mockClient.LoadBalancer.(*cloudstack.MockLoadBalancerServiceIface).EXPECT().
				NewListLoadBalancerRulesParams().
				Return(&cloudstack.ListLoadBalancerRulesParams{})
			mockClient.LoadBalancer.(*cloudstack.MockLoadBalancerServiceIface).EXPECT().
				ListLoadBalancerRules(gomock.Any()).
				Return(&cloudstack.ListLoadBalancerRulesResponse{
					LoadBalancerRules: []*cloudstack.LoadBalancerRule{{Publicport: "6443", Id: "lbRuleID"}}}, nil)

			Ω(cloud.FetchLoadBalancerRule(mockClient, csCluster)).Should(Succeed())
			Ω(csCluster.Status.LBRuleID).Should(Equal("lbRuleID"))
		})

		It("doesn't create a new load blancer rule on create", func() {
			mockClient.LoadBalancer.(*cloudstack.MockLoadBalancerServiceIface).EXPECT().
				NewListLoadBalancerRulesParams().
				Return(&cloudstack.ListLoadBalancerRulesParams{})
			mockClient.LoadBalancer.(*cloudstack.MockLoadBalancerServiceIface).EXPECT().
				ListLoadBalancerRules(gomock.Any()).
				Return(&cloudstack.ListLoadBalancerRulesResponse{
					LoadBalancerRules: []*cloudstack.LoadBalancerRule{{Publicport: "6443", Id: "lbRuleID"}}}, nil)

			Ω(cloud.CreateLoadBalancerRule(mockClient, csCluster)).Should(Succeed())
			Ω(csCluster.Status.LBRuleID).Should(Equal("lbRuleID"))
		})
	})

	Context("load balancer rule does not exist", func() {
		csCluster.Spec.ControlPlaneEndpoint.Port = int32(6443)
		It("calls cloudstack to create a new load balancer rule.", func() {
			mockClient.LoadBalancer.(*cloudstack.MockLoadBalancerServiceIface).EXPECT().
				NewListLoadBalancerRulesParams().
				Return(&cloudstack.ListLoadBalancerRulesParams{})
			mockClient.LoadBalancer.(*cloudstack.MockLoadBalancerServiceIface).EXPECT().
				ListLoadBalancerRules(gomock.Any()).
				Return(&cloudstack.ListLoadBalancerRulesResponse{
					LoadBalancerRules: []*cloudstack.LoadBalancerRule{{Publicport: "7443", Id: "lbRuleID"}}}, nil)
			mockClient.LoadBalancer.(*cloudstack.MockLoadBalancerServiceIface).EXPECT().
				NewCreateLoadBalancerRuleParams(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&cloudstack.CreateLoadBalancerRuleParams{})
			mockClient.LoadBalancer.(*cloudstack.MockLoadBalancerServiceIface).EXPECT().
				CreateLoadBalancerRule(gomock.Any()).
				Return(&cloudstack.CreateLoadBalancerRuleResponse{Id: "randomID"}, nil)
			Ω(cloud.CreateLoadBalancerRule(mockClient, csCluster)).Should(Succeed())
			Ω(csCluster.Status.LBRuleID).Should(Equal("randomID"))
		})
	})
})
