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
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
)

var _ = Describe("Network", func() {
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
		client = cloud.NewClientFromCSAPIClient(mockClient)
		dummies.SetDummyVars()
		dummies.SetDummyClusterStatus()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("calls to create an isolated network when not found", func() {
		dummies.Zone1.Network = dummies.ISONet1
		dummies.Zone1.Network.ID = ""
		dummies.CSCluster.Status.Zones = infrav1.ZoneStatusMap{dummies.Zone1.ID: dummies.Zone1}
		dummies.CSCluster.Status.PublicIPNetworkID = dummies.ISONet1.ID

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
		fs.EXPECT().NewCreateEgressFirewallRuleParams(dummies.ISONet1.ID, cloud.NetworkProtocolTCP).
			Return(&csapi.CreateEgressFirewallRuleParams{})
		fs.EXPECT().CreateEgressFirewallRule(&csapi.CreateEgressFirewallRuleParams{}).
			Return(&csapi.CreateEgressFirewallRuleResponse{}, nil)

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

		Ω(client.GetOrCreateIsolatedNetwork(dummies.CSZone1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
	})

	Context("for a closed firewall", func() {
		It("OpenFirewallRule asks CloudStack to open the firewall", func() {
			dummies.Zone1.Network = dummies.ISONet1
			dummies.CSCluster.Status.Zones = infrav1.ZoneStatusMap{dummies.Zone1.ID: dummies.Zone1}
			dummies.CSCluster.Status.PublicIPNetworkID = dummies.ISONet1.ID
			fs.EXPECT().NewCreateEgressFirewallRuleParams(dummies.ISONet1.ID, cloud.NetworkProtocolTCP).
				Return(&csapi.CreateEgressFirewallRuleParams{})
			fs.EXPECT().CreateEgressFirewallRule(&csapi.CreateEgressFirewallRuleParams{}).
				Return(&csapi.CreateEgressFirewallRuleResponse{}, nil)

			Ω(client.OpenFirewallRules(dummies.CSISONet1)).Should(Succeed())
		})
	})

	Context("for an open firewall", func() {
		It("OpenFirewallRule asks CloudStack to open the firewall anyway, but doesn't fail", func() {
			dummies.Zone1.Network = dummies.ISONet1
			dummies.CSCluster.Status.Zones = infrav1.ZoneStatusMap{dummies.Zone1.ID: dummies.Zone1}
			dummies.CSCluster.Status.PublicIPNetworkID = dummies.ISONet1.ID

			fs.EXPECT().NewCreateEgressFirewallRuleParams(dummies.ISONet1.ID, "tcp").
				Return(&csapi.CreateEgressFirewallRuleParams{})
			fs.EXPECT().CreateEgressFirewallRule(&csapi.CreateEgressFirewallRuleParams{}).
				Return(&csapi.CreateEgressFirewallRuleResponse{}, errors.New("there is already a rule like this"))

			Ω(client.OpenFirewallRules(dummies.CSISONet1)).Should(Succeed())
		})
	})

	Context("in an isolated network with public IPs available", func() {
		It("will resolve public IP details given an endpoint spec", func() {
			ipAddress := "192.168.1.14"
			as.EXPECT().NewListPublicIpAddressesParams().Return(&csapi.ListPublicIpAddressesParams{})
			as.EXPECT().ListPublicIpAddresses(gomock.Any()).
				Return(&csapi.ListPublicIpAddressesResponse{
					Count:             1,
					PublicIpAddresses: []*csapi.PublicIpAddress{{Id: "PublicIPID", Ipaddress: ipAddress}},
				}, nil)
			publicIPAddress, err := client.GetPublicIP(dummies.CSZone1, dummies.CSISONet1, dummies.CSCluster)
			Ω(err).Should(Succeed())
			Ω(publicIPAddress).ShouldNot(BeNil())
			Ω(publicIPAddress.Ipaddress).Should(Equal(ipAddress))
		})
	})

	Context("The specific load balancer rule does exist", func() {
		It("resolves the rule's ID", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).Return(
				&csapi.ListLoadBalancerRulesResponse{LoadBalancerRules: []*csapi.LoadBalancerRule{
					{Publicport: strconv.Itoa(int(dummies.EndPointPort)), Id: dummies.LBRuleID}}}, nil)

			dummies.CSISONet1.Status.LBRuleID = ""
			Ω(client.ResolveLoadBalancerRuleDetails(dummies.CSZone1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSISONet1.Status.LBRuleID).Should(Equal(dummies.LBRuleID))
		})

		It("doesn't create a new load balancer rule on create", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).
				Return(&csapi.ListLoadBalancerRulesResponse{
					LoadBalancerRules: []*csapi.LoadBalancerRule{
						{Publicport: strconv.Itoa(int(dummies.EndPointPort)), Id: dummies.LBRuleID}}}, nil)

			Ω(client.GetOrCreateLoadBalancerRule(dummies.CSZone1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSISONet1.Status.LBRuleID).Should(Equal(dummies.LBRuleID))
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

			Ω(client.GetOrCreateLoadBalancerRule(dummies.CSZone1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSISONet1.Status.LBRuleID).Should(Equal("2ndLBRuleID"))
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
			dummies.CSZone1.Spec.ID = ""                          // Make CAPC methods resolve this.
			dummies.CSCluster.Status.Zones = infrav1.ZoneStatusMap{}

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
			Ω(client.GetOrCreateIsolatedNetwork(dummies.CSZone1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
			Ω(client.GetOrCreateIsolatedNetwork(dummies.CSZone1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())

			// Network should now exist if it didn't at the start.
			Ω(client.ResolveNetwork(&dummies.ISONet1)).Should(Succeed())

			// Do once more.
			Ω(client.GetOrCreateIsolatedNetwork(dummies.CSZone1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
		})
	})
})
