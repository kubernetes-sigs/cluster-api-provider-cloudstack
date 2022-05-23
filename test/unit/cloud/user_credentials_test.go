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
	"github.com/aws/cluster-api-provider-cloudstack-staging/test/unit/dummies"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	. "github.com/onsi/ginkgo/v2"

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

	Context("UserCred Semi-Integ Tests", func() {
		var domain cloud.Domain
		var account cloud.Account
		var user cloud.User

		BeforeEach(func() {
			client = realCloudClient

			// Setup dummies.
			// TODO: move these to the test dummies package.
			domain = cloud.Domain{Path: "ROOT/blah/blah/subsub"}
			account = cloud.Account{Name: "SuperNested", Domain: domain}
			user = cloud.User{Name: "SubSub", Account: account}
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
			cfg := cloud.Config{APIKey: user.APIKey, SecretKey: user.SecretKey}
			newClient, err := client.NewClientFromSpec(cfg)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(newClient).ShouldNot(BeNil())
		})
	})
})
