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
	. "github.com/onsi/gomega"
	"gopkg.in/ini.v1"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
)

// Example cloud-config ini structure.
type Global struct {
	APIURL    string `ini:"api-url"`
	VerifySSL bool   `ini:"verify-ssl"`
}

var _ = Describe("Client", func() {

	var ()

	BeforeEach(func() {
		// This test fixture is useful for development, but the actual method of parsing is confinded to the client's
		// new client method. The parsing used here is more of a schema, and we don't need to test another library's
		// abilities to parse said schema.
	})

	AfterEach(func() {
	})

	Context("When fetching an INI config.", func() {
		It("Handles the positive case.", func() {
			cfg := &Global{}
			rawCfg, err := ini.Load("../../cloud-config")
			Ω(rawCfg.Section("Global")).ShouldNot(BeNil())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(rawCfg.Section("Global").MapTo(cfg)).Should(Succeed())
			Ω(cfg.VerifySSL).Should(BeFalse())
			Ω(cfg.APIURL).ShouldNot(BeEmpty())
		})
	})

	Context("Create Client and Get API Configuration", func() {
		It("For a configuration with the 'Global' section missing", func() {
			filepath := getConfigPath("cloud-config-no-global")

			client, err := cloud.NewClient(filepath)

			Ω(client).Should(BeNil())
			Ω(err.Error()).Should(ContainSubstring("section Global not found"))
		})

		It("fails to read config file", func() {
			filepath := getConfigPath("cloud-config-no-exist")
			client, err := cloud.NewClient(filepath)

			Ω(client).Should(BeNil())
			Ω(err.Error()).Should(ContainSubstring("reading config at path"))
		})

		It("Create CloudStack client", func() {
			filepath := getConfigPath("cloud-config-good")
			client, err := cloud.NewClient(filepath)

			Ω(client).ShouldNot(BeNil())
			Ω(err.Error()).Should(ContainSubstring("checking CloudStack API Client connectivity"))
		})
	})
})
