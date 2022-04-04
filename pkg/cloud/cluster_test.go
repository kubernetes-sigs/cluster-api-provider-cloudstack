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
	capcv1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Cluster", func() {
	var (
		client     cloud.Client
		mockCtrl   *gomock.Controller
		mockClient *csapi.CloudStackClient
		zs         *csapi.MockZoneServiceIface
		ds         *csapi.MockDomainServiceIface
		as         *csapi.MockAccountServiceIface
		ns         *csapi.MockNetworkServiceIface
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = csapi.NewMockClient(mockCtrl)
		zs = mockClient.Zone.(*csapi.MockZoneServiceIface)
		ds = mockClient.Domain.(*csapi.MockDomainServiceIface)
		as = mockClient.Account.(*csapi.MockAccountServiceIface)
		ns = mockClient.Network.(*csapi.MockNetworkServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
		dummies.SetDummyVars()
		dummies.SetDummyDomainAndAccount()
		dummies.SetDummyCSApiResponse()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("an existing abstract dummies.CSCluster", func() {
		It("handles zone not found.", func() {
			expectedErr := fmt.Errorf("Not found")
			zs.EXPECT().GetZoneID(dummies.Zone1.Name).Return("", -1, expectedErr)
			zs.EXPECT().GetZoneByID(dummies.Zone1.ID).Return(nil, -1, expectedErr)

			err := client.GetOrCreateCluster(dummies.CSCluster)
			Expect(errors.Cause(err)).To(MatchError(expectedErr))
		})

		It("handles multiple zone IDs returned", func() {
			zs.EXPECT().GetZoneID(dummies.Zone1.Name).Return(dummies.Zone1.ID, 2, nil)
			zs.EXPECT().GetZoneByID(dummies.Zone1.ID).Return(nil, -1, fmt.Errorf("Not found"))

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).Should(MatchError(And(
				ContainSubstring("expected 1 Zone with name "+dummies.Zone1.Name+", but got 2"),
				ContainSubstring("could not get Zone by ID "+dummies.Zone1.ID+": Not found"))))
		})

		It("resolves domain and account when both are specified", func() {
			zs.EXPECT().GetZoneID(dummies.Zone1.Name).Return(dummies.Zone1.ID, 1, nil)
			zs.EXPECT().GetZoneByID(dummies.Zone1.ID).Return(dummies.CAPCZoneToCSAPIZone(&dummies.Zone1), 1, nil)
			ds.EXPECT().NewListDomainsParams().Return(dummies.ListDomainsParams)
			ds.EXPECT().ListDomains(dummies.ListDomainsParams).Return(dummies.ListDomainsResp, nil)
			as.EXPECT().NewListAccountsParams().Return(dummies.ListAccountsParams)
			as.EXPECT().ListAccounts(dummies.ListAccountsParams).Return(dummies.ListAccountsResp, nil)
			ns.EXPECT().GetNetworkByName(dummies.Net1.Name).Return(dummies.CAPCNetToCSAPINet(&dummies.Net1), 1, nil)

			// Limit test to single zone.
			dummies.CSCluster.Spec.Zones = []capcv1.Zone{dummies.Zone1}
			dummies.CSCluster.Status.Zones = capcv1.ZoneStatusMap{}

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.DomainID).Should(Equal(dummies.DomainID))
		})

		It("resolves domain and account when none are specified", func() {
			zs.EXPECT().GetZoneID(dummies.Zone1.Name).Return(dummies.Zone1.ID, 1, nil)
			zs.EXPECT().GetZoneByID(dummies.Zone1.ID).Return(dummies.CAPCZoneToCSAPIZone(&dummies.Zone1), 1, nil)
			ns.EXPECT().GetNetworkByName(dummies.Net1.Name).Return(dummies.CAPCNetToCSAPINet(&dummies.Net1), 1, nil)

			// Limit test to single zone.
			dummies.CSCluster.Spec.Zones = []capcv1.Zone{dummies.Zone1}
			dummies.CSCluster.Status.Zones = capcv1.ZoneStatusMap{}

			dummies.CSCluster.Spec.Domain = ""
			dummies.CSCluster.Spec.Account = ""

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.DomainID).Should(Equal(""))
		})

		It("fails when only one of domain or account is specified", func() {
			zs.EXPECT().GetZoneID(dummies.Zone1.Name).Return(dummies.Zone1.ID, 1, nil).AnyTimes()
			zs.EXPECT().GetZoneByID(dummies.Zone1.ID).Return(dummies.CAPCZoneToCSAPIZone(&dummies.Zone1), 1, nil).AnyTimes()
			ns.EXPECT().GetNetworkByName(dummies.Net1.Name).Return(dummies.CAPCNetToCSAPINet(&dummies.Net1), 1, nil).AnyTimes()

			// Limit test to single zone.
			dummies.CSCluster.Spec.Zones = []capcv1.Zone{dummies.Zone1}
			dummies.CSCluster.Status.Zones = capcv1.ZoneStatusMap{}

			domainBackup := dummies.CSCluster.Spec.Domain
			dummies.CSCluster.Spec.Domain = ""

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).ShouldNot(Succeed())

			dummies.CSCluster.Spec.Domain = domainBackup
			dummies.CSCluster.Spec.Account = ""

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).ShouldNot(Succeed())
		})
	})
})
