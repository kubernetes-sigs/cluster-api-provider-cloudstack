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
	"errors"
	"fmt"

	csapi "github.com/apache/cloudstack-go/v2/cloudstack"
	gomock "go.uber.org/mock/gomock"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/helpers"

	"github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("User Credentials", func() {
	const (
		errorMessage = "Error"
	)

	fakeError := errors.New(errorMessage)
	var (
		mockCtrl   *gomock.Controller
		mockClient *csapi.CloudStackClient
		ds         *csapi.MockDomainServiceIface
		as         *csapi.MockAccountServiceIface
		us         *csapi.MockUserServiceIface
	)

	ginkgo.BeforeEach(func() {
		mockCtrl = gomock.NewController(ginkgo.GinkgoT())
		mockClient = csapi.NewMockClient(mockCtrl)
		ds = mockClient.Domain.(*csapi.MockDomainServiceIface)
		as = mockClient.Account.(*csapi.MockAccountServiceIface)
		us = mockClient.User.(*csapi.MockUserServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient, nil)
		dummies.SetDummyVars()
		// dummies.SetDummyClusterStatus()
		// dummies.SetDummyCSMachineStatuses()
		dummies.SetDummyCAPCClusterVars()
	})

	ginkgo.AfterEach(func() {
		mockCtrl.Finish()
	})

	ginkgo.Context("Get domain in CloudStack", func() {
		ginkgo.It("search for CloudStack domain", func() {
			dummies.Domain.Path = "domainPath1"
			dsp := &csapi.ListDomainsParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT/domainPath1",
			}}}, nil)

			gomega.Ω(client.ResolveDomain(&dummies.Domain)).Should(gomega.Succeed())
		})

		ginkgo.It("search for CloudStack domain with incorrect domain path", func() {
			dummies.Domain.Path = "/domainPath1"
			dsp := &csapi.ListDomainsParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT/domainPath1",
			}}}, nil)

			err := client.ResolveDomain(&dummies.Domain)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.Equal(fmt.Sprintf("domain Path %s did not match domain ID %s", dummies.Domain.Path, dummies.Domain.ID)))
		})

		ginkgo.It("search for CloudStack domain returns more than one domain", func() {
			dummies.Domain.Path = "domainPath1"
			dsp := &csapi.ListDomainsParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 2, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT/domainPath1",
			}}}, nil)

			err := client.ResolveDomain(&dummies.Domain)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.Equal(fmt.Sprintf("domain ID %s provided, expected exactly one domain, got %d", dummies.Domain.ID, 2)))
		})

		ginkgo.It("search for CloudStack domain when only domain Name is provided", func() {
			dummies.Domain.ID = ""
			dsp := &csapi.ListDomainsParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
			}}}, nil)

			gomega.Ω(client.ResolveDomain(&dummies.Domain)).Should(gomega.Succeed())
		})

		ginkgo.It("search for CloudStack domain when only domain Name is provided, but returns > 1 domain", func() {
			dummies.Domain.ID = ""
			dsp := &csapi.ListDomainsParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 2, Domains: []*csapi.Domain{{
				Name: "domainName",
			}}}, nil)

			err := client.ResolveDomain(&dummies.Domain)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.Equal(fmt.Sprintf("only domain name: %s provided, expected exactly one domain, got %d", dummies.Domain.Name, 2)))
		})
	})

	ginkgo.Context("Get Account in CloudStack", func() {
		ginkgo.It("search for account in CloudStack", func() {
			dsp := &csapi.ListDomainsParams{}
			asp := &csapi.ListAccountsParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT",
			}}}, nil)
			as.EXPECT().NewListAccountsParams().Return(asp)
			as.EXPECT().ListAccounts(asp).Return(&csapi.ListAccountsResponse{Count: 1, Accounts: []*csapi.Account{{
				Id:   dummies.AccountID,
				Name: dummies.AccountName,
			}}}, nil)

			gomega.Ω(client.ResolveAccount(&dummies.Account)).Should(gomega.Succeed())

		})

		ginkgo.It("no account found in CloudStack for the provided Account name", func() {
			dsp := &csapi.ListDomainsParams{}
			asp := &csapi.ListAccountsParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT",
			}}}, nil)
			as.EXPECT().NewListAccountsParams().Return(asp)
			as.EXPECT().ListAccounts(asp).Return(&csapi.ListAccountsResponse{Count: 0, Accounts: []*csapi.Account{}}, nil)

			err := client.ResolveAccount(&dummies.Account)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring("could not find account"))
		})

		ginkgo.It("More than one account found in the provided domain and account name", func() {
			dsp := &csapi.ListDomainsParams{}
			asp := &csapi.ListAccountsParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT",
			}}}, nil)
			as.EXPECT().NewListAccountsParams().Return(asp)
			as.EXPECT().ListAccounts(asp).Return(&csapi.ListAccountsResponse{Count: 2, Accounts: []*csapi.Account{}}, nil)

			err := client.ResolveAccount(&dummies.Account)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring("expected 1 Account with account name"))
		})

		ginkgo.It("fails to list accounts", func() {
			dsp := &csapi.ListDomainsParams{}
			asp := &csapi.ListAccountsParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT",
			}}}, nil)
			as.EXPECT().NewListAccountsParams().Return(asp)
			as.EXPECT().ListAccounts(asp).Return(nil, fakeError)

			gomega.Ω(client.ResolveAccount(&dummies.Account)).ShouldNot(gomega.Succeed())
		})
	})

	ginkgo.Context("Get User from CloudStack", func() {
		ginkgo.BeforeEach(func() {
			dummies.SetDummyUserVars()
		})

		ginkgo.It("search for user in CloudStack", func() {
			dsp := &csapi.ListDomainsParams{}
			asp := &csapi.ListAccountsParams{}
			usp := &csapi.ListUsersParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT",
			}}}, nil)
			as.EXPECT().NewListAccountsParams().Return(asp)
			as.EXPECT().ListAccounts(asp).Return(&csapi.ListAccountsResponse{Count: 1, Accounts: []*csapi.Account{{
				Id:   dummies.AccountID,
				Name: dummies.AccountName,
			}}}, nil)
			us.EXPECT().NewListUsersParams().Return(usp)
			us.EXPECT().ListUsers(usp).Return(&csapi.ListUsersResponse{
				Count: 1,
				Users: []*csapi.User{{
					Id:       dummies.UserID,
					Username: dummies.Username,
				}},
			}, nil)

			gomega.Ω(client.ResolveUser(&dummies.User)).Should(gomega.Succeed())
		})

		ginkgo.It("search for user fails while resolving account in CloudStack", func() {
			dsp := &csapi.ListDomainsParams{}
			asp := &csapi.ListAccountsParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT",
			}}}, nil)
			as.EXPECT().NewListAccountsParams().Return(asp)
			as.EXPECT().ListAccounts(asp).Return(nil, fakeError)

			err := client.ResolveUser(&dummies.User)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring("resolving account"))
		})

		ginkgo.It("search for user in CloudStack fails", func() {
			dsp := &csapi.ListDomainsParams{}
			asp := &csapi.ListAccountsParams{}
			usp := &csapi.ListUsersParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT",
			}}}, nil)
			as.EXPECT().NewListAccountsParams().Return(asp)
			as.EXPECT().ListAccounts(asp).Return(&csapi.ListAccountsResponse{Count: 1, Accounts: []*csapi.Account{{
				Id:   dummies.AccountID,
				Name: dummies.AccountName,
			}}}, nil)
			us.EXPECT().NewListUsersParams().Return(usp)
			us.EXPECT().ListUsers(usp).Return(nil, fakeError)

			gomega.Ω(client.ResolveUser(&dummies.User)).ShouldNot(gomega.Succeed())
		})

		ginkgo.It("search for user in CloudStack results in more than one user", func() {
			dsp := &csapi.ListDomainsParams{}
			asp := &csapi.ListAccountsParams{}
			usp := &csapi.ListUsersParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT",
			}}}, nil)
			as.EXPECT().NewListAccountsParams().Return(asp)
			as.EXPECT().ListAccounts(asp).Return(&csapi.ListAccountsResponse{Count: 1, Accounts: []*csapi.Account{{
				Id:   dummies.AccountID,
				Name: dummies.AccountName,
			}}}, nil)
			us.EXPECT().NewListUsersParams().Return(usp)
			us.EXPECT().ListUsers(usp).Return(&csapi.ListUsersResponse{
				Count: 2,
				Users: []*csapi.User{},
			}, nil)

			err := client.ResolveUser(&dummies.User)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring("expected 1 User with username"))
		})
	})

	ginkgo.Context("Get user keys in CloudStack", func() {
		initialCalls := func() {
			dsp := &csapi.ListDomainsParams{}
			asp := &csapi.ListAccountsParams{}

			ds.EXPECT().NewListDomainsParams().Return(dsp)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT",
			}}}, nil)
			as.EXPECT().NewListAccountsParams().Return(asp)
			as.EXPECT().ListAccounts(asp).Return(&csapi.ListAccountsResponse{Count: 1, Accounts: []*csapi.Account{{
				Id:   dummies.AccountID,
				Name: dummies.AccountName,
			}}}, nil)
		}

		ginkgo.It("get user keys", func() {
			initialCalls()
			usp := &csapi.ListUsersParams{}
			ukp := &csapi.GetUserKeysParams{}
			us.EXPECT().NewListUsersParams().Return(usp)
			us.EXPECT().ListUsers(usp).Return(&csapi.ListUsersResponse{
				Count: 1,
				Users: []*csapi.User{{
					Id:       dummies.UserID,
					Username: dummies.Username,
				}},
			}, nil)
			us.EXPECT().NewGetUserKeysParams(gomock.Any()).Return(ukp)
			us.EXPECT().GetUserKeys(ukp).Return(&csapi.GetUserKeysResponse{
				Apikey:    dummies.Apikey,
				Secretkey: dummies.SecretKey,
			}, nil)

			gomega.Ω(client.ResolveUserKeys(&dummies.User)).Should(gomega.Succeed())
		})

		ginkgo.It("get user keys fils when resolving user", func() {
			initialCalls()
			usp := &csapi.ListUsersParams{}

			us.EXPECT().NewListUsersParams().Return(usp)
			us.EXPECT().ListUsers(usp).Return(nil, fakeError)

			err := client.ResolveUserKeys(&dummies.User)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring("error encountered when resolving user details"))

		})

		ginkgo.It("get user keys fils when resolving user keys", func() {
			initialCalls()
			usp := &csapi.ListUsersParams{}
			ukp := &csapi.GetUserKeysParams{}

			us.EXPECT().NewListUsersParams().Return(usp)
			us.EXPECT().ListUsers(usp).Return(&csapi.ListUsersResponse{
				Count: 1,
				Users: []*csapi.User{{
					Id:       dummies.UserID,
					Username: dummies.Username,
				}},
			}, nil)
			us.EXPECT().NewGetUserKeysParams(gomock.Any()).Return(ukp)
			us.EXPECT().GetUserKeys(ukp).Return(nil, fakeError)

			err := client.ResolveUserKeys(&dummies.User)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring("error encountered when resolving user api keys"))
		})
	})

	ginkgo.Context("Get user with keys", func() {
		ginkgo.BeforeEach(func() {
			dummies.SetDummyUserVars()
		})

		initialCalls := func() {
			dsp := &csapi.ListDomainsParams{}
			ds.EXPECT().NewListDomainsParams().Return(dsp).Times(1)
			ds.EXPECT().ListDomains(dsp).Return(&csapi.ListDomainsResponse{Count: 1, Domains: []*csapi.Domain{{
				Name: "domainName",
				Path: "ROOT",
			}}}, nil)
		}
		ginkgo.It("get first user for given account and domain", func() {
			initialCalls()
			initialCalls()
			asp := &csapi.ListAccountsParams{}
			usp := &csapi.ListUsersParams{}
			ukp := &csapi.GetUserKeysParams{}

			as.EXPECT().NewListAccountsParams().Return(asp).Times(2)
			as.EXPECT().ListAccounts(asp).Return(&csapi.ListAccountsResponse{Count: 1, Accounts: []*csapi.Account{{
				Id:   dummies.AccountID,
				Name: dummies.AccountName,
			}}}, nil).Times(2)
			us.EXPECT().NewListUsersParams().Return(usp).Times(2)
			us.EXPECT().ListUsers(usp).Return(&csapi.ListUsersResponse{
				Count: 1,
				Users: []*csapi.User{{
					Id:       dummies.UserID,
					Username: dummies.Username,
				}},
			}, nil).Times(2)
			us.EXPECT().NewGetUserKeysParams(gomock.Any()).Return(ukp)
			us.EXPECT().GetUserKeys(ukp).Return(&csapi.GetUserKeysResponse{
				Apikey:    dummies.Apikey,
				Secretkey: dummies.SecretKey,
			}, nil)

			result, err := client.GetUserWithKeys(&dummies.User)
			gomega.Ω(err).Should(gomega.Succeed())
			gomega.Ω(result).Should(gomega.BeTrue())
		})

		ginkgo.It("fails to resolve accout", func() {
			initialCalls()
			asp := &csapi.ListAccountsParams{}
			as.EXPECT().NewListAccountsParams().Return(asp)
			as.EXPECT().ListAccounts(asp).Return(nil, fakeError)

			result, err := client.GetUserWithKeys(&dummies.User)
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring(fmt.Sprintf("resolving account %s details", dummies.User.Account.Name)))
			gomega.Ω(result).Should(gomega.BeFalse())
		})

		ginkgo.It("fails to resolve accout", func() {
			initialCalls()
			asp := &csapi.ListAccountsParams{}
			usp := &csapi.ListUsersParams{}

			as.EXPECT().NewListAccountsParams().Return(asp)
			as.EXPECT().ListAccounts(asp).Return(&csapi.ListAccountsResponse{Count: 1, Accounts: []*csapi.Account{{
				Id:   dummies.AccountID,
				Name: dummies.AccountName,
			}}}, nil)
			us.EXPECT().NewListUsersParams().Return(usp)
			us.EXPECT().ListUsers(usp).Return(nil, fakeError)

			result, err := client.GetUserWithKeys(&dummies.User)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(result).Should(gomega.BeFalse())
		})
	})

	ginkgo.Context("UserCred Integ Tests", ginkgo.Label("integ"), func() {
		var domain cloud.Domain
		var account cloud.Account
		var user cloud.User

		ginkgo.BeforeEach(func() {
			client = realCloudClient

			domain = cloud.Domain{Path: testDomainPath}
			account = cloud.Account{Domain: domain}
			user = cloud.User{Name: helpers.TempUserName, Account: account}
		})

		ginkgo.It("can resolve a domain from the path", func() {
			gomega.Ω(client.ResolveDomain(&domain)).Should(gomega.Succeed())
			gomega.Ω(domain.ID).ShouldNot(gomega.BeEmpty())
		})

		ginkgo.It("can resolve an account from the domain path and account name", func() {
			gomega.Ω(client.ResolveAccount(&account)).Should(gomega.Succeed())
			gomega.Ω(account.ID).ShouldNot(gomega.BeEmpty())
		})

		ginkgo.It("can resolve a user from the domain path, account name, and user name", func() {
			gomega.Ω(client.ResolveUser(&user)).Should(gomega.Succeed())
			gomega.Ω(user.ID).ShouldNot(gomega.BeEmpty())
		})

		ginkgo.It("can get sub-domain user's credentials", func() {
			gomega.Ω(client.ResolveUserKeys(&user)).Should(gomega.Succeed())

			gomega.Ω(user.APIKey).ShouldNot(gomega.BeEmpty())
			gomega.Ω(user.SecretKey).ShouldNot(gomega.BeEmpty())
		})

		ginkgo.It("can get an arbitrary user with keys from domain and account specifications alone", func() {
			found, err := client.GetUserWithKeys(&user)
			gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
			gomega.Ω(found).Should(gomega.BeTrue())
			gomega.Ω(user.APIKey).ShouldNot(gomega.BeEmpty())
		})

		ginkgo.It("can get create a new client as another user", func() {
			found, err := client.GetUserWithKeys(&user)
			gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
			gomega.Ω(found).Should(gomega.BeTrue())
			gomega.Ω(user.APIKey).ShouldNot(gomega.BeEmpty())
			newClient, err := client.NewClientInDomainAndAccount(user.Account.Domain.Name, user.Account.Name, "")
			gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
			gomega.Ω(newClient).ShouldNot(gomega.BeNil())
		})
	})
})
