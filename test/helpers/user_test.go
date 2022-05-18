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
	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/aws/cluster-api-provider-cloudstack/test/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

var _ = Describe("Test helper methods", func() {

	Context("Domain Creation and Deletion Integ Tests", func() {
		var csClient *cloudstack.CloudStackClient
		ccPath := "../../cloud-config"
		var connectionErr error
		conf := cloud.Config{}
		if rawCfg, err := ini.Load("../../cloud-config"); err != nil {
			connectionErr = errors.Wrapf(err, "reading config at path %s:", ccPath)
		} else if g := rawCfg.Section("Global"); len(g.Keys()) == 0 {
			connectionErr = errors.New("section Global not found")
		} else if err = rawCfg.Section("Global").StrictMapTo(&conf); err != nil {
			connectionErr = errors.Wrapf(err, "parsing [Global] section from config at path %s:", ccPath)
		}

		csClient = cloudstack.NewAsyncClient(conf.APIURL, conf.APIKey, conf.SecretKey, conf.VerifySSL)

		// Get the root domain's ID.
		rootDomainID, err, found := helpers.GetDomainByPath(csClient, "ROOT/")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(rootDomainID).ShouldNot(BeEmpty())
		Ω(found).Should(BeTrue())

		BeforeEach(func() {
			if connectionErr != nil { // Only do these tests if an actual ACS instance is available via cloud-config.
				Skip("Could not connect to ACS instance.")
			}
		})

		AfterEach(func() {
			for _, path := range []string{"ROOT/someNewDomain", "ROOT/blah"} {
				// Delete any created domains.
				id, err, found := helpers.GetDomainByPath(csClient, path)
				Ω(err).ShouldNot(HaveOccurred())
				if found {
					Ω(helpers.DeleteDomain(csClient, id)).Should(Succeed())
				}
			}
		})

		It("Can get the ROOT domain's ID.", func() {
			id, err, found := helpers.GetDomainByPath(csClient, "ROOT/")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(id).ShouldNot(BeEmpty())
			Ω(found).Should(BeTrue())
		})

		It("Doesn't error when unable to get a domain's ID.", func() {
			id, err, found := helpers.GetDomainByPath(csClient, "ROOT/blahnotpresent")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(found).Should(BeFalse())
			Ω(id).Should(BeEmpty())
		})

		It("Can create a domain under a parent domain.", func() {
			id, err := helpers.CreateDomainUnderParent(csClient, rootDomainID, "someNewDomain")
			Ω(id).ShouldNot(BeEmpty())
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("Returns an appropriate error when the domain already exists.", func() {
			someDomain := &cloud.Domain{Name: "blah", Path: "blah"}
			Ω(helpers.GetOrCreateDomain(someDomain, csClient)).Should(Succeed())
			Ω(someDomain.Name).Should(Equal("blah"))
			Ω(someDomain.Path).Should(Equal("ROOT/blah"))
			Ω(someDomain.ID).ShouldNot(BeEmpty())
			_, err = helpers.CreateDomainUnderParent(csClient, rootDomainID, "blah")
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("already exists"))
		})

		It("Doesn't error if the domain already exists.", func() {
			someDomain := &cloud.Domain{Name: "blah", Path: "blah"}
			Ω(helpers.GetOrCreateDomain(someDomain, csClient)).Should(Succeed())
			Ω(someDomain.Name).Should(Equal("blah"))
			Ω(someDomain.Path).Should(Equal("ROOT/blah"))
			Ω(someDomain.ID).ShouldNot(BeEmpty())

			Ω(helpers.GetOrCreateDomain(someDomain, csClient)).Should(Succeed())
			Ω(someDomain.Name).Should(Equal("blah"))
			Ω(someDomain.Path).Should(Equal("ROOT/blah"))
			Ω(someDomain.ID).ShouldNot(BeEmpty())
		})

		It("Can create a wholly new multi-level sub-domain path.", func() {
			someDomain := &cloud.Domain{Name: "tooBlah", Path: "ROOT/someNewDomain/tooBlah"}
			Ω(helpers.GetOrCreateDomain(someDomain, csClient)).Should(Succeed())
			Ω(someDomain.Name).Should(Equal("tooBlah"))
			Ω(someDomain.Path).Should(Equal("ROOT/someNewDomain/tooBlah"))
			Ω(someDomain.ID).ShouldNot(BeEmpty())
		})
	})
})
