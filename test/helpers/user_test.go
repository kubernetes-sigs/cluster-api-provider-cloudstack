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

package helpers_test

import (
	"github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/helpers"
)

var _ = ginkgo.Describe("Test helper methods", func() {
	csClient, err := helpers.NewCSClient()
	gomega.Ω(err).ShouldNot(gomega.HaveOccurred())

	// Get the root domain's ID.
	rootDomainID, err, found := helpers.GetDomainByPath(csClient, "ROOT/")
	gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
	gomega.Ω(rootDomainID).ShouldNot(gomega.BeEmpty())
	gomega.Ω(found).Should(gomega.BeTrue())

	ginkgo.AfterEach(func() {
		for _, path := range []string{"ROOT/someNewDomain", "ROOT/blah"} {
			// Delete any created domains.
			id, err, found := helpers.GetDomainByPath(csClient, path)
			gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
			if found {
				gomega.Ω(helpers.DeleteDomain(csClient, id)).Should(gomega.Succeed())
			}
		}
	})

	ginkgo.Context("Domain Creation and Deletion.", func() {
		ginkgo.It("Can get the ROOT domain's ID.", func() {
			id, err, found := helpers.GetDomainByPath(csClient, "ROOT/")
			gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
			gomega.Ω(id).ShouldNot(gomega.BeEmpty())
			gomega.Ω(found).Should(gomega.BeTrue())
		})

		ginkgo.It("Doesn't error when unable to get a domain's ID.", func() {
			id, err, found := helpers.GetDomainByPath(csClient, "ROOT/blahnotpresent")
			gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
			gomega.Ω(found).Should(gomega.BeFalse())
			gomega.Ω(id).Should(gomega.BeEmpty())
		})

		ginkgo.It("Can create a domain under a parent domain.", func() {
			id, err := helpers.CreateDomainUnderParent(csClient, rootDomainID, "someNewDomain")
			gomega.Ω(id).ShouldNot(gomega.BeEmpty())
			gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("Returns an appropriate error when the domain already exists.", func() {
			someDomain := &cloud.Domain{Name: "blah", Path: "blah"}
			gomega.Ω(helpers.GetOrCreateDomain(csClient, someDomain)).Should(gomega.Succeed())
			gomega.Ω(someDomain.Name).Should(gomega.Equal("blah"))
			gomega.Ω(someDomain.Path).Should(gomega.Equal("ROOT/blah"))
			gomega.Ω(someDomain.ID).ShouldNot(gomega.BeEmpty())
			_, err = helpers.CreateDomainUnderParent(csClient, rootDomainID, "blah")
			gomega.Ω(err).Should(gomega.HaveOccurred())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring("already exists"))
		})

		ginkgo.It("Doesn't error if the domain already exists.", func() {
			someDomain := &cloud.Domain{Name: "blah", Path: "blah"}
			gomega.Ω(helpers.GetOrCreateDomain(csClient, someDomain)).Should(gomega.Succeed())
			gomega.Ω(someDomain.Name).Should(gomega.Equal("blah"))
			gomega.Ω(someDomain.Path).Should(gomega.Equal("ROOT/blah"))
			gomega.Ω(someDomain.ID).ShouldNot(gomega.BeEmpty())

			gomega.Ω(helpers.GetOrCreateDomain(csClient, someDomain)).Should(gomega.Succeed())
			gomega.Ω(someDomain.Name).Should(gomega.Equal("blah"))
			gomega.Ω(someDomain.Path).Should(gomega.Equal("ROOT/blah"))
			gomega.Ω(someDomain.ID).ShouldNot(gomega.BeEmpty())
		})

		ginkgo.It("Can create a wholly new multi-level sub-domain path.", func() {
			someDomain := &cloud.Domain{Name: "tooBlah", Path: "ROOT/someNewDomain/tooBlah"}
			gomega.Ω(helpers.GetOrCreateDomain(csClient, someDomain)).Should(gomega.Succeed())
			gomega.Ω(someDomain.Name).Should(gomega.Equal("tooBlah"))
			gomega.Ω(someDomain.Path).Should(gomega.Equal("ROOT/someNewDomain/tooBlah"))
			gomega.Ω(someDomain.ID).ShouldNot(gomega.BeEmpty())
		})
	})

	ginkgo.Context("Account Creation.", func() {
		ginkgo.It("Can create a new account in a new domain.", func() {
			domain := cloud.Domain{Path: "ROOT/someNewDomain/tooBlah"}
			account := cloud.Account{Name: "TempTestAccount", Domain: domain}
			gomega.Ω(helpers.GetOrCreateAccount(csClient, &account)).Should(gomega.Succeed())
		})
		// already exists
		ginkgo.It("Doesn't fail if the account already exists.", func() {
			domain := cloud.Domain{Path: "ROOT/someNewDomain/tooBlah"}
			account := cloud.Account{Name: "TempTestAccount", Domain: domain}
			gomega.Ω(helpers.GetOrCreateAccount(csClient, &account)).Should(gomega.Succeed())
			gomega.Ω(helpers.GetOrCreateAccount(csClient, &account)).Should(gomega.Succeed())
		})
	})

	ginkgo.Context("User Creation w/Keys.", func() {
		ginkgo.It("Can create a new user with keys.", func() {
			domain := cloud.Domain{Path: "ROOT/someNewDomain/tooBlah"}
			account := cloud.Account{Name: "TempTestAccount", Domain: domain}
			user := cloud.User{Account: account}
			gomega.Ω(helpers.GetOrCreateUserWithKey(csClient, &user)).Should(gomega.Succeed())
			gomega.Ω(user.ID).ShouldNot(gomega.BeEmpty())
			gomega.Ω(user.APIKey).ShouldNot(gomega.BeEmpty())
		})
	})
})
