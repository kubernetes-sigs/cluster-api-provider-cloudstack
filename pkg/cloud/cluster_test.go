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
	"fmt"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Cluster", func() {
	var (
		client     cloud.Client
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient
		zs         *cloudstack.MockZoneServiceIface
		ds         *cloudstack.MockDomainServiceIface
		ns         *cloudstack.MockNetworkServiceIface
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		zs = mockClient.Zone.(*cloudstack.MockZoneServiceIface)
		ds = mockClient.Domain.(*cloudstack.MockDomainServiceIface)
		ns = mockClient.Network.(*cloudstack.MockNetworkServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("an existing abstract cluster", func() {
		zoneName := "zoneName"
		zoneID := "zoneID"
		netName := "netName"

		cluster := &infrav1.CloudStackCluster{
			Spec: infrav1.CloudStackClusterSpec{
				Zone:    zoneName,
				Network: netName}}

		It("handles zone not found.", func() {
			expectedErr := fmt.Errorf("Not found")
			zs.EXPECT().GetZoneID(zoneName).Return("", -1, expectedErr)
			zs.EXPECT().GetZoneByID(zoneName).Return(nil, -1, expectedErr)

			err := client.GetOrCreateCluster(cluster)
			Expect(errors.Cause(err)).To(MatchError(expectedErr))
		})

		It("handles multiple zone IDs returned", func() {
			zs.EXPECT().GetZoneID(zoneName).Return(zoneID, 2, nil)
			zs.EXPECT().GetZoneByID(zoneName).Return(nil, -1, fmt.Errorf("Not found"))

			err := client.GetOrCreateCluster(cluster)
			Expect(err.Error()).To(ContainSubstring("Expected 1 Zone with name zoneName, but got 2."))
			Expect(err.Error()).To(ContainSubstring("Could not get Zone by ID zoneName.: Not found"))
		})

		It("translates Domain to DomainID when Domain is set", func() {
			cluster.Spec.Domain = "FakeDomain"
			cluster.Spec.Network = "FakeNetwork"
			domainID := "FakeDomainID"
			zs.EXPECT().GetZoneID(zoneName).Return(zoneID, 1, nil)
			ds.EXPECT().GetDomainID(cluster.Spec.Domain).Return(domainID, 1, nil)

			// End the fetching with a fake network error here.
			// Only trying to test domain functions.
			// TODO: turn the pkg/cloud/client.go client into a composition of interfaces such that the
			// individual services can be mocked.
			ns.EXPECT().GetNetworkID(cluster.Spec.Network).Return("", -1, fmt.Errorf("FakeError"))
			ns.EXPECT().GetNetworkByID(cluster.Spec.Network).Return(&cloudstack.Network{}, -1, fmt.Errorf("FakeError"))

			Ω(client.GetOrCreateCluster(cluster)).ShouldNot(Succeed())
			Ω(cluster.Status.DomainID).Should(Equal(domainID))
		})
	})
})
