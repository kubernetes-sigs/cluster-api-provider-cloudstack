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
			client.GetOrCreateDomain()
			//id, err := client.FindDomain("blah/blah/subsub")
			//Ω(err).ShouldNot(HaveOccurred())
			////user, err := client.GetUser("SuperNested", *id)
			//Ω(err).ShouldNot(HaveOccurred())

			fmt.Println(user)
			fmt.Println(user.Apikey)
			fmt.Println(user.Apikey)
			fmt.Println(user.Apikey)
			fmt.Println(user.Secretkey)
			fmt.Println(user.Secretkey)
			fmt.Println(user.Secretkey)

			//Ω(client.ChangeUser(user.Apikey, user.Secretkey)).Should(Succeed())
			//Ω(client.SetUserToOriginal()).Should(Succeed())
		})
	})
})
