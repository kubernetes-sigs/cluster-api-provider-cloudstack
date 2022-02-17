/*
Copyright 2022.

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
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("Network", func() {
	var ( // Declare shared vars.
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient
		ns         *cloudstack.MockNetworkServiceIface
		nos        *cloudstack.MockNetworkOfferingServiceIface
		fs         *cloudstack.MockFirewallServiceIface
		as         *cloudstack.MockAddressServiceIface
		lbs        *cloudstack.MockLoadBalancerServiceIface
		rs         *cloudstack.MockResourcetagsServiceIface
		csCluster  *infrav1.CloudStackCluster
		client     cloud.Client
	)

	const (
		fakeNetId           = "fakeNetID"
		fakeNetName         = "fakeNetName"
		isolatedNetworkType = "Isolated"
		lbRuleId            = "lbRuleID"
		netId               = "someNetID"
		protocol            = "tcp"
		publicPort          = "6443"
	)

	BeforeEach(func() {
		// Setup new mock services.
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		ns = mockClient.Network.(*cloudstack.MockNetworkServiceIface)
		nos = mockClient.NetworkOffering.(*cloudstack.MockNetworkOfferingServiceIface)
		fs = mockClient.Firewall.(*cloudstack.MockFirewallServiceIface)
		as = mockClient.Address.(*cloudstack.MockAddressServiceIface)
		lbs = mockClient.LoadBalancer.(*cloudstack.MockLoadBalancerServiceIface)
		rs = mockClient.Resourcetags.(*cloudstack.MockResourcetagsServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)

		// Reset csCluster.
		csCluster = &infrav1.CloudStackCluster{
			Spec: infrav1.CloudStackClusterSpec{
				Zone:                 "zone1",
				Network:              fakeNetName,
				ControlPlaneEndpoint: clusterv1.APIEndpoint{Port: int32(6443)},
			},
			ObjectMeta: metav1.ObjectMeta{
				UID: "0",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	// Sets expectations network tag creation.  To be used by tests that get/create networks.
	expectNetworkTags := func(networkId string) {
		listTagsParams := &cloudstack.ListTagsParams{}
		createTagsParams := &cloudstack.CreateTagsParams{}
		rs.EXPECT().NewListTagsParams().Return(listTagsParams)
		rs.EXPECT().ListTags(listTagsParams).Return(&cloudstack.ListTagsResponse{}, nil)
		rs.EXPECT().NewCreateTagsParams([]string{networkId}, "network", gomock.Any()).Return(createTagsParams)
		rs.EXPECT().CreateTags(createTagsParams).Return(&cloudstack.CreateTagsResponse{}, nil)
	}

	Context("for an existing network", func() {
		It("resolves network details in cluster status", func() {
			ns.EXPECT().GetNetworkID(fakeNetName).Return(fakeNetId, 1, nil)
			ns.EXPECT().GetNetworkByID(fakeNetId).Return(&cloudstack.Network{Type: isolatedNetworkType}, 1, nil)
			Ω(client.ResolveNetwork(csCluster)).Should(Succeed())
			Ω(csCluster.Status.NetworkID).Should(Equal(fakeNetId))
			Ω(csCluster.Status.NetworkType).Should(Equal(isolatedNetworkType))
		})

		It("does not call to create a new network via GetOrCreateNetwork", func() {
			ns.EXPECT().GetNetworkID(fakeNetName).Return(fakeNetId, 1, nil)
			ns.EXPECT().GetNetworkByID(fakeNetId).Return(&cloudstack.Network{Type: isolatedNetworkType}, 1, nil)
			expectNetworkTags(fakeNetId)

			Ω(client.GetOrCreateNetwork(csCluster)).Should(Succeed())
		})

		It("resolves network details with network ID instead of network name", func() {
			ns.EXPECT().GetNetworkID(gomock.Any()).Return("", -1, errors.New("No match found for blah."))
			ns.EXPECT().GetNetworkByID(fakeNetId).Return(&cloudstack.Network{Type: isolatedNetworkType}, 1, nil)
			expectNetworkTags(fakeNetId)

			csCluster.Spec.Network = fakeNetId
			Ω(client.GetOrCreateNetwork(csCluster)).Should(Succeed())
		})
	})

	Context("for a non-existent network", func() {
		It("when GetOrCreateNetwork is called it calls CloudStack to create a network", func() {
			ns.EXPECT().GetNetworkID(gomock.Any()).Return("", -1, errors.New("No match found for blah."))
			ns.EXPECT().GetNetworkByID(gomock.Any()).Return(nil, -1, errors.New("No match found for blah."))
			nos.EXPECT().GetNetworkOfferingID(gomock.Any()).Return("someOfferingID", 1, nil)
			ns.EXPECT().NewCreateNetworkParams(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&cloudstack.CreateNetworkParams{})
			ns.EXPECT().CreateNetwork(gomock.Any()).Return(&cloudstack.CreateNetworkResponse{Id: netId}, nil)
			expectNetworkTags(netId)

			Ω(client.GetOrCreateNetwork(csCluster)).Should(Succeed())
		})
	})

	Context("for a closed firewall", func() {
		It("OpenFirewallRule asks CloudStack to open the firewall", func() {
			netID := netId
			csCluster.Status.NetworkID = netID
			fs.EXPECT().NewCreateEgressFirewallRuleParams(netID, protocol).
				Return(&cloudstack.CreateEgressFirewallRuleParams{})
			fs.EXPECT().CreateEgressFirewallRule(&cloudstack.CreateEgressFirewallRuleParams{}).
				Return(&cloudstack.CreateEgressFirewallRuleResponse{}, nil)

			Ω(client.OpenFirewallRules(csCluster)).Should(Succeed())
		})
	})

	Context("for an open firewall", func() {
		It("OpenFirewallRule asks CloudStack to open the firewall anyway, but doesn't fail", func() {
			netID := netId
			csCluster.Status.NetworkID = netID
			fs.EXPECT().NewCreateEgressFirewallRuleParams(netID, protocol).
				Return(&cloudstack.CreateEgressFirewallRuleParams{})
			fs.EXPECT().CreateEgressFirewallRule(&cloudstack.CreateEgressFirewallRuleParams{}).
				Return(&cloudstack.CreateEgressFirewallRuleResponse{}, errors.New("There is already a rule like this."))
			Ω(client.OpenFirewallRules(csCluster)).Should(Succeed())
		})
	})

	Context("in an isolated network with public IPs available", func() {
		It("will resolve public IP details given an endpoint spec", func() {
			ipAddress := "192.168.1.14"
			as.EXPECT().NewListPublicIpAddressesParams().Return(&cloudstack.ListPublicIpAddressesParams{})
			as.EXPECT().ListPublicIpAddresses(gomock.Any()).
				Return(&cloudstack.ListPublicIpAddressesResponse{
					Count:             1,
					PublicIpAddresses: []*cloudstack.PublicIpAddress{{Id: "PublicIPID", Ipaddress: ipAddress}},
				}, nil)
			publicIpAddress, err := client.ResolvePublicIPDetails(csCluster)
			Ω(err).Should(Succeed())
			Ω(publicIpAddress).ShouldNot(BeNil())
			Ω(publicIpAddress.Ipaddress).Should(Equal(ipAddress))
		})
	})

	Context("The specific load balancer rule does exist", func() {
		It("resolves the rule's ID", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&cloudstack.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).Return(
				&cloudstack.ListLoadBalancerRulesResponse{
					LoadBalancerRules: []*cloudstack.LoadBalancerRule{{Publicport: publicPort, Id: lbRuleId}}}, nil)
			Ω(client.ResolveLoadBalancerRuleDetails(csCluster)).Should(Succeed())
			Ω(csCluster.Status.LBRuleID).Should(Equal(lbRuleId))
		})

		It("doesn't create a new load blancer rule on create", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&cloudstack.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).
				Return(&cloudstack.ListLoadBalancerRulesResponse{
					LoadBalancerRules: []*cloudstack.LoadBalancerRule{{Publicport: publicPort, Id: lbRuleId}}}, nil)
			Ω(client.GetOrCreateLoadBalancerRule(csCluster)).Should(Succeed())
			Ω(csCluster.Status.LBRuleID).Should(Equal(lbRuleId))
		})
	})

	Context("load balancer rule does not exist", func() {
		It("calls cloudstack to create a new load balancer rule.", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&cloudstack.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).Return(&cloudstack.ListLoadBalancerRulesResponse{
				LoadBalancerRules: []*cloudstack.LoadBalancerRule{{Publicport: "7443", Id: lbRuleId}}}, nil)
			lbs.EXPECT().NewCreateLoadBalancerRuleParams(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&cloudstack.CreateLoadBalancerRuleParams{})
			const randomId = "randomID"
			lbs.EXPECT().CreateLoadBalancerRule(gomock.Any()).
				Return(&cloudstack.CreateLoadBalancerRuleResponse{Id: randomId}, nil)
			Ω(client.GetOrCreateLoadBalancerRule(csCluster)).Should(Succeed())
			Ω(csCluster.Status.LBRuleID).Should(Equal(randomId))
		})
	})
})
