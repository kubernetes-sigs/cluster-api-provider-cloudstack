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
	capcv1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
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

			Ω(client.ResolveNetworkStatuses(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.Zones[dummies.Zone1.ID].Network).Should(Equal(dummies.Net1))
			Ω(dummies.CSCluster.Status.Zones[dummies.Zone2.ID].Network).Should(Equal(dummies.Net2))
		})

		It("correctly identifies the shared network use case", func() {
			Ω(cloud.UsesIsolatedNetwork(dummies.CSCluster)).Should(BeFalse())
		})
		It("correctly identifies the isolated network use case", func() {
			dummies.Zone1.Network = dummies.ISONet1
			dummies.CSCluster.Status.Zones = map[string]capcv1.Zone{dummies.Zone1.ID: dummies.Zone1}
			Ω(cloud.UsesIsolatedNetwork(dummies.CSCluster)).Should(BeTrue())
		})
		It("correctly identifies an existing network from a network status", func() {
			Ω(cloud.NetworkExists(dummies.CSCluster.Status.Zones.GetOne().Network)).Should(BeTrue())
		})
	})

	Context("for a non-existent network", func() {
		It("when ResolveNetworkStatuses is called it does not create a network", func() {
			ns.EXPECT().GetNetworkByName(gomock.Any()).Return(nil, -1, errors.New("no match found for blah"))
			ns.EXPECT().GetNetworkByID(gomock.Any()).Return(nil, -1, errors.New("no match found for blah"))

			Ω(client.ResolveNetworkStatuses(dummies.CSCluster)).ShouldNot(Succeed())
		})
	})

	It("calls to create an isolated network when not found", func() {
		dummies.Zone1.Network = dummies.ISONet1
		dummies.Zone1.Network.ID = ""
		dummies.CSCluster.Status.Zones = capcv1.ZoneStatusMap{dummies.Zone1.ID: dummies.Zone1}
		dummies.CSCluster.Status.PublicIPNetworkID = dummies.ISONet1.ID

		nos.EXPECT().GetNetworkOfferingID(gomock.Any()).Return("someOfferingID", 1, nil)
		ns.EXPECT().NewCreateNetworkParams(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&csapi.CreateNetworkParams{})
		ns.EXPECT().CreateNetwork(gomock.Any()).Return(&csapi.CreateNetworkResponse{Id: dummies.ISONet1.ID}, nil)
		as.EXPECT().NewListPublicIpAddressesParams().Return(&csapi.ListPublicIpAddressesParams{})
		as.EXPECT().ListPublicIpAddresses(gomock.Any()).
			Return(&csapi.ListPublicIpAddressesResponse{
				Count:             1,
				PublicIpAddresses: []*csapi.PublicIpAddress{{Id: dummies.PublicIPID, Ipaddress: "fakeIP"}}}, nil)
		as.EXPECT().NewAssociateIpAddressParams().Return(&csapi.AssociateIpAddressParams{})
		as.EXPECT().AssociateIpAddress(gomock.Any())

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

		Ω(client.GetOrCreateIsolatedNetwork(dummies.CSCluster)).Should(Succeed())
	})

	Context("for a closed firewall", func() {
		It("OpenFirewallRule asks CloudStack to open the firewall", func() {
			dummies.Zone1.Network = dummies.ISONet1
			dummies.CSCluster.Status.Zones = capcv1.ZoneStatusMap{dummies.Zone1.ID: dummies.Zone1}
			dummies.CSCluster.Status.PublicIPNetworkID = dummies.ISONet1.ID
			fs.EXPECT().NewCreateEgressFirewallRuleParams(dummies.ISONet1.ID, cloud.NetworkProtocolTCP).
				Return(&csapi.CreateEgressFirewallRuleParams{})
			fs.EXPECT().CreateEgressFirewallRule(&csapi.CreateEgressFirewallRuleParams{}).
				Return(&csapi.CreateEgressFirewallRuleResponse{}, nil)

			Ω(client.OpenFirewallRules(dummies.CSCluster)).Should(Succeed())
		})
	})

	Context("for an open firewall", func() {
		It("OpenFirewallRule asks CloudStack to open the firewall anyway, but doesn't fail", func() {
			dummies.Zone1.Network = dummies.ISONet1
			dummies.CSCluster.Status.Zones = capcv1.ZoneStatusMap{dummies.Zone1.ID: dummies.Zone1}
			dummies.CSCluster.Status.PublicIPNetworkID = dummies.ISONet1.ID

			fs.EXPECT().NewCreateEgressFirewallRuleParams(dummies.ISONet1.ID, "tcp").
				Return(&csapi.CreateEgressFirewallRuleParams{})
			fs.EXPECT().CreateEgressFirewallRule(&csapi.CreateEgressFirewallRuleParams{}).
				Return(&csapi.CreateEgressFirewallRuleResponse{}, errors.New("there is already a rule like this"))

			Ω(client.OpenFirewallRules(dummies.CSCluster)).Should(Succeed())
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
			publicIPAddress, err := client.FetchPublicIP(dummies.CSCluster)
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

			dummies.CSCluster.Status.LBRuleID = ""
			Ω(client.ResolveLoadBalancerRuleDetails(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.LBRuleID).Should(Equal(dummies.LBRuleID))
		})

		It("doesn't create a new load balancer rule on create", func() {
			lbs.EXPECT().NewListLoadBalancerRulesParams().Return(&csapi.ListLoadBalancerRulesParams{})
			lbs.EXPECT().ListLoadBalancerRules(gomock.Any()).
				Return(&csapi.ListLoadBalancerRulesResponse{
					LoadBalancerRules: []*csapi.LoadBalancerRule{
						{Publicport: strconv.Itoa(int(dummies.EndPointPort)), Id: dummies.LBRuleID}}}, nil)

			Ω(client.GetOrCreateLoadBalancerRule(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.LBRuleID).Should(Equal(dummies.LBRuleID))
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

			Ω(client.GetOrCreateLoadBalancerRule(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.LBRuleID).Should(Equal("2ndLBRuleID"))
		})
	})

	Context("Networking Integ Tests", func() {
		client, connectionErr := cloud.NewClient("../../cloud-config")

		BeforeEach(func() {
			if connectionErr != nil { // Only do these tests if an actual ACS instance is available via cloud-config.
				Skip("Could not connect to ACS instance.")
			}
			if err := client.ResolveNetwork(dummies.CSCluster, &dummies.Net1); err != nil {
				Skip("Could not find network.")
			}

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
		})

		It("fetches an isolated network", func() {
			dummies.SetDummyIsoNetToNameOnly()
			dummies.SetClusterSpecToNet(&dummies.ISONet1)

			Ω(client.ResolveNetwork(dummies.CSCluster, &dummies.ISONet1)).Should(Succeed())
			Ω(dummies.ISONet1.ID).ShouldNot(BeEmpty())
			Ω(dummies.ISONet1.Type).Should(Equal(cloud.NetworkTypeIsolated))
		})

		It("fetches a public IP", func() {
			dummies.Zone1.ID = ""
			dummies.SetDummyIsoNetToNameOnly()
			dummies.SetClusterSpecToNet(&dummies.ISONet1)
			dummies.CSCluster.Spec.ControlPlaneEndpoint.Host = ""
			Ω(client.ResolveZones(dummies.CSCluster)).Should(Succeed())
			Ω(client.ResolveNetwork(dummies.CSCluster, &dummies.ISONet1)).Should(Succeed())
		})
	})
})
