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
	csapi "github.com/apache/cloudstack-go/v2/cloudstack"
	capcv1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cluster", func() {
	var (
		client     cloud.Client
		mockCtrl   *gomock.Controller
		mockClient *csapi.CloudStackClient
		ds         *csapi.MockDomainServiceIface
		as         *csapi.MockAccountServiceIface
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = csapi.NewMockClient(mockCtrl)
		ds = mockClient.Domain.(*csapi.MockDomainServiceIface)
		as = mockClient.Account.(*csapi.MockAccountServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
		dummies.SetDummyVars()
		dummies.SetDummyDomainAndAccount()
		dummies.SetDummyCSApiResponse()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("an existing abstract dummies.CSCluster", func() {

		It("resolves domain and account when both are specified", func() {
			ds.EXPECT().NewListDomainsParams().Return(dummies.ListDomainsParams)
			ds.EXPECT().ListDomains(dummies.ListDomainsParams).Return(dummies.ListDomainsResp, nil)
			as.EXPECT().NewListAccountsParams().Return(dummies.ListAccountsParams)
			as.EXPECT().ListAccounts(dummies.ListAccountsParams).Return(dummies.ListAccountsResp, nil)
			// Limit test to single zone.
			dummies.CSCluster.Spec.Zones = []capcv1.Zone{dummies.Zone1}
			dummies.CSCluster.Status.Zones = capcv1.ZoneStatusMap{}

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.DomainID).Should(Equal(dummies.DomainID))
		})

		It("resolves domain when ROOT domain is specified", func() {
			ds.EXPECT().NewListDomainsParams().Return(dummies.ListDomainsParams)
			ds.EXPECT().ListDomains(dummies.ListDomainsParams).Return(dummies.ListDomainsResp, nil)
			as.EXPECT().NewListAccountsParams().Return(dummies.ListAccountsParams)
			as.EXPECT().ListAccounts(dummies.ListAccountsParams).Return(dummies.ListAccountsResp, nil)

			// Limit test to single zone.
			dummies.CSCluster.Spec.Zones = []capcv1.Zone{dummies.Zone1}
			dummies.CSCluster.Status.Zones = capcv1.ZoneStatusMap{}

			dummies.CSCluster.Spec.Domain = dummies.RootDomain

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.DomainID).Should(Equal(dummies.RootDomainID))
		})

		It("resolves domain when domain is a fully qualified name", func() {
			ds.EXPECT().NewListDomainsParams().Return(dummies.ListDomainsParams)
			ds.EXPECT().ListDomains(dummies.ListDomainsParams).Return(dummies.ListDomainsResp, nil)
			as.EXPECT().NewListAccountsParams().Return(dummies.ListAccountsParams)
			as.EXPECT().ListAccounts(dummies.ListAccountsParams).Return(dummies.ListAccountsResp, nil)

			dummies.CSCluster.Spec.Domain = dummies.Level2Domain

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.DomainID).Should(Equal(dummies.Level2DomainID))
		})

		It("fails to resolve domain when domain path does not match", func() {
			ds.EXPECT().NewListDomainsParams().Return(dummies.ListDomainsParams)
			ds.EXPECT().ListDomains(dummies.ListDomainsParams).Return(dummies.ListDomainsResp, nil)
			as.EXPECT().NewListAccountsParams().Return(dummies.ListAccountsParams)
			as.EXPECT().ListAccounts(dummies.ListAccountsParams).Return(dummies.ListAccountsResp, nil)

			dummies.CSCluster.Spec.Domain = dummies.Level2Domain

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.DomainID).Should(Equal(dummies.Level2DomainID))
		})

		It("doesn't fail when none are specified", func() {
			dummies.CSCluster.Spec.Domain = ""
			dummies.CSCluster.Spec.Account = ""

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).Should(Succeed())
			Ω(dummies.CSCluster.Status.DomainID).Should(Equal(""))
		})

		It("fails when only one of domain or account is specified", func() {
			domainBackup := dummies.CSCluster.Spec.Domain
			dummies.CSCluster.Spec.Domain = ""

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).ShouldNot(Succeed())

			dummies.CSCluster.Spec.Domain = domainBackup
			dummies.CSCluster.Spec.Account = ""

			Ω(client.GetOrCreateCluster(dummies.CSCluster)).ShouldNot(Succeed())
		})
	})
})
