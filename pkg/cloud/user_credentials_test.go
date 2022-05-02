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
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	. "github.com/onsi/ginkgo"
	"github.com/pkg/errors"

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
		client, connectionErr := cloud.NewClient("../../cloud-config")

		BeforeEach(func() {
			if connectionErr != nil { // Only do these tests if an actual ACS instance is available via cloud-config.
				Skip(errors.Wrapf(connectionErr, "Could not connect to ACS instance.").Error())
			}
		})

		It("can get sub-domain user's credentials", func() {
			domain := cloud.Domain{Path: "ROOT/blah/blah/subsub"}
			account := cloud.Account{Name: "SuperNested", Domain: domain}
			user := cloud.User{Name: "SubSub", Account: account}
			// Ω(client.ResolveDomain(&domain)).Should(Succeed())
			// Ω(client.ResolveAccount(&account)).Should(Succeed())
			// Ω(client.ResolveUser(&user)).Should(Succeed())
			Ω(client.ResolveUserKeys(&user)).Should(Succeed())

			Ω(user.APIKey).ShouldNot(BeEmpty())
			Ω(user.SecretKey).ShouldNot(BeEmpty())
		})
	})
})
