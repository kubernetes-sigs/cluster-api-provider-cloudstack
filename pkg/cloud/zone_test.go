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
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
)

var _ = Describe("Zone", func() {
	const (
		errorMessage = "Error"
	)

	fakeError := errors.New(errorMessage)
	var (
		client     cloud.Client
		mockCtrl   *gomock.Controller
		mockClient *csapi.CloudStackClient
		zs         *csapi.MockZoneServiceIface
		ns         *csapi.MockNetworkServiceIface
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = csapi.NewMockClient(mockCtrl)
		zs = mockClient.Zone.(*csapi.MockZoneServiceIface)
		ns = mockClient.Network.(*csapi.MockNetworkServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
		dummies.SetDummyVars()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("an existing abstract dummies.CSCluster", func() {
		It("handles zone not found.", func() {
			expectedErr := fmt.Errorf("Not found")
			zs.EXPECT().GetZoneID(dummies.Zone1.Name).Return("", -1, expectedErr)
			zs.EXPECT().GetZoneByID(dummies.Zone1.ID).Return(nil, -1, expectedErr)

			err := client.ResolveZone(&dummies.CSFailureDomain1.Spec.Zone)
			Expect(errors.Cause(err)).To(MatchError(expectedErr))
		})

		It("handles multiple zone IDs returned", func() {
			zs.EXPECT().GetZoneID(dummies.Zone1.Name).Return(dummies.Zone1.ID, 2, nil)
			zs.EXPECT().GetZoneByID(dummies.Zone1.ID).Return(nil, -1, fmt.Errorf("Not found"))

			Ω(client.ResolveZone(&dummies.CSFailureDomain1.Spec.Zone)).Should(MatchError(And(
				ContainSubstring("expected 1 Zone with name "+dummies.Zone1.Name+", but got 2"),
				ContainSubstring("could not get Zone by ID "+dummies.Zone1.ID+": Not found"))))
		})

		It("returns multiple zones for a given id", func() {
			zs.EXPECT().GetZoneID(dummies.Zone1.Name).Return(dummies.Zone1.ID, 2, nil)
			zs.EXPECT().GetZoneByID(dummies.Zone1.ID).Return(&csapi.Zone{}, 2, nil)

			Ω(client.ResolveZone(&dummies.CSFailureDomain1.Spec.Zone).Error()).
				Should(ContainSubstring("expected 1 Zone with name " + dummies.Zone1.Name + ", but got 2"))
		})
	})

	Context("Resolve network for zone", func() {
		It("get network by name specfied in zone spec", func() {
			ns.EXPECT().GetNetworkByName(dummies.Zone1.Network.Name).Return(&csapi.Network{}, 1, nil)

			Ω(client.ResolveNetworkForZone(&dummies.CSFailureDomain1.Spec.Zone)).Should(Succeed())
		})

		It("get network by name specfied in zone spec returns > 1 network", func() {
			ns.EXPECT().GetNetworkByName(dummies.Zone2.Network.Name).Return(&csapi.Network{}, 2, nil)
			ns.EXPECT().GetNetworkByID(dummies.Zone2.Network.ID).Return(&csapi.Network{}, 2, nil)

			Ω(client.ResolveNetworkForZone(&dummies.CSFailureDomain2.Spec.Zone)).Should(MatchError(And(
				ContainSubstring(fmt.Sprintf("expected 1 Network with name %s, but got %d", dummies.Zone2.Network.Name, 2)),
				ContainSubstring(fmt.Sprintf("expected 1 Network with UUID %v, but got %d", dummies.Zone2.Network.ID, 2)))))
		})

		It("get network by id specfied in zone spec", func() {
			ns.EXPECT().GetNetworkByName(dummies.Zone2.Network.Name).Return(nil, -1, fakeError)
			ns.EXPECT().GetNetworkByID(dummies.Zone2.Network.ID).Return(&csapi.Network{}, 1, nil)
			Ω(client.ResolveNetworkForZone(&dummies.CSFailureDomain2.Spec.Zone)).Should(Succeed())
		})

		It("get network by id fails", func() {
			ns.EXPECT().GetNetworkByName(dummies.Zone2.Network.Name).Return(nil, -1, fakeError)
			ns.EXPECT().GetNetworkByID(dummies.Zone2.Network.ID).Return(nil, -1, fakeError)

			Ω(client.ResolveNetworkForZone(&dummies.CSFailureDomain2.Spec.Zone).Error()).Should(ContainSubstring(fmt.Sprintf("could not get Network by ID %s", dummies.Zone2.Network.ID)))
		})
	})
})
