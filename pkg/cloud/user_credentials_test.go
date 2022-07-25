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
	. "github.com/onsi/ginkgo/v2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/helpers"

	. "github.com/onsi/gomega"
)

var _ = Describe("User Credentials", func() {

	BeforeEach(func() {
		dummies.SetDummyVars()
		dummies.SetDummyClusterStatus()
		dummies.SetDummyCSMachineStatuses()
	})

	AfterEach(func() {
	})

	Context("UserCred Integ Tests", Label("integ"), func() {
		var domain cloud.Domain
		var account cloud.Account
		var user cloud.User

		BeforeEach(func() {
			client = realCloudClient

			domain = cloud.Domain{Path: testDomainPath}
			account = cloud.Account{Domain: domain}
			user = cloud.User{Name: helpers.TempUserName, Account: account}
		})

		It("can resolve a domain from the path", func() {
			Ω(client.ResolveDomain(&domain)).Should(Succeed())
			Ω(domain.ID).ShouldNot(BeEmpty())
		})

		It("can resolve an account from the domain path and account name", func() {
			Ω(client.ResolveAccount(&account)).Should(Succeed())
			Ω(account.ID).ShouldNot(BeEmpty())
		})

		It("can resolve a user from the domain path, account name, and user name", func() {
			Ω(client.ResolveUser(&user)).Should(Succeed())
			Ω(user.ID).ShouldNot(BeEmpty())
		})

		It("can get sub-domain user's credentials", func() {
			Ω(client.ResolveUserKeys(&user)).Should(Succeed())

			Ω(user.APIKey).ShouldNot(BeEmpty())
			Ω(user.SecretKey).ShouldNot(BeEmpty())
		})

		It("can get an arbitrary user with keys from domain and account specifications alone", func() {
			found, err := client.GetUserWithKeys(&user)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(found).Should(BeTrue())
			Ω(user.APIKey).ShouldNot(BeEmpty())
		})

		It("can get create a new client as another user", func() {
			found, err := client.GetUserWithKeys(&user)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(found).Should(BeTrue())
			Ω(user.APIKey).ShouldNot(BeEmpty())
			newClient, err := client.NewClientInDomainAndAccount(user.Account.Name, user.Account.Domain.Name)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(newClient).ShouldNot(BeNil())
		})
	})
})
