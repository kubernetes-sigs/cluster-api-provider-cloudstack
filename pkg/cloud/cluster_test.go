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

/*
	These tests assume a running CloudStack instance and associated cloud-config file.

	They are not proper, but they're enough for now.

	TODO Use mocks and testing frameworks.
*/
package cloud

import (
	"fmt"
	"testing"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	_ "github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	_ "github.com/onsi/gomega"
	"github.com/pkg/errors"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
)

func TestClusterInfo(t *testing.T) {
	zoneName := "zone"
	zoneID := "zone-id"
	networkName := "network"
	networkID := "network-id"

	mockCtrl := gomock.NewController(t)
	mockClient := cloudstack.NewMockClient(mockCtrl)
	cluster := &infrav1.CloudStackCluster{
		Spec: infrav1.CloudStackClusterSpec{
			Zone:    zoneName,
			Network: networkName}}

	t.Run("Fetching Cluster Information", func(t *testing.T) {
		t.Run("Zone not found", func(t *testing.T) {
			g := NewWithT(t)
			zoneName := "zone"
			expectedErr := fmt.Errorf("Not found")
			zs := mockClient.Zone.(*cloudstack.MockZoneServiceIface)
			zs.EXPECT().GetZoneID(zoneName).Return("", -1, expectedErr)

			if err := FetchClusterInfo(mockClient, cluster); err != nil {
				g.Expect(errors.Cause(err)).To(MatchError(expectedErr))
			} else {
				t.Error()
			}
		})

		t.Run("Network not found", func(t *testing.T) {
			g := NewWithT(t)
			zs := mockClient.Zone.(*cloudstack.MockZoneServiceIface)
			zs.EXPECT().GetZoneID(zoneName).Return(zoneID, 1, nil)

			expectedErr := fmt.Errorf("Not found")
			ns := mockClient.Network.(*cloudstack.MockNetworkServiceIface)
			ns.EXPECT().GetNetworkID(networkName).Return("", -1, expectedErr)

			if err := FetchClusterInfo(mockClient, cluster); err != nil {
				g.Expect(errors.Cause(err)).To(MatchError(expectedErr))
			} else {
				t.Error()
			}
		})

		t.Run("Zone and network IDs are fetched", func(t *testing.T) {
			g := NewWithT(t)
			zs := mockClient.Zone.(*cloudstack.MockZoneServiceIface)
			zs.EXPECT().GetZoneID(zoneName).Return(zoneID, 1, nil)

			ns := mockClient.Network.(*cloudstack.MockNetworkServiceIface)
			ns.EXPECT().GetNetworkID(networkName).Return(networkID, 1, nil)

			if err := FetchClusterInfo(mockClient, cluster); err != nil {
				t.Error()
			} else {
				g.Expect(cluster.Status.ZoneID).To(Equal(zoneID))
				g.Expect(cluster.Status.NetworkID).To(Equal(networkID))
			}
		})
	})
}
