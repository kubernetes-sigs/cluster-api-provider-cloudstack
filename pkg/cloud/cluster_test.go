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
	infrav1 "github.com/aws/cluster-api-provider-cloudstack-staging/api/v1alpha3"
	"github.com/aws/cluster-api-provider-cloudstack-staging/pkg/cloud"
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
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		zs = mockClient.Zone.(*cloudstack.MockZoneServiceIface)
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

		// This will take more extensive mocking to completely test now that is does so much more.
		// It("should fetch cluster information.", func() {
		// 	zs := mockClient.Zone.(*cloudstack.MockZoneServiceIface)
		// 	zs.EXPECT().GetZoneID(zoneName).Return(zoneID, 1, nil)

		// 	ns := mockClient.Network.(*cloudstack.MockNetworkServiceIface)
		// 	ns.EXPECT().GetNetworkID(netName).Return(netID, 1, nil)

		// 	Ω(cloud.CreateCluster(mockClient, cluster)).Should(Succeed())
		// 	Ω(cluster.Status.ZoneID).Should(Equal(zoneID))
		// 	Ω(cluster.Status.NetworkID).Should(Equal(netID))
		// })

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
	})
})
