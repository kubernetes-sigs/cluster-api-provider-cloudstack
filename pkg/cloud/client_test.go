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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/ini.v1"
)

// Example cloud-config ini structure.
type Global struct {
	APIURL    string `ini:"api-url"`
	VerifySSL bool   `ini:"verify-ssl"`
}

var _ = Describe("Instance", func() {

	var ()

	BeforeEach(func() {
		// This test fixture is useful for development, but the actual method of parsing is confinded to the client's
		// new client method. The parsing used here is more of a schema, and we don't need to test another library's
		// abilities to parse said schema.
		Skip("Dev test suite.")
	})

	AfterEach(func() {
	})

	Context("When fetching an INI config.", func() {
		It("Handles the positive case.", func() {
			cfg := &Global{}
			rawCfg, err := ini.Load("../../cloud-config")
			Ω(rawCfg.Section("Global")).ShouldNot(BeNil())
			fmt.Println(rawCfg.Section("Global"))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(rawCfg.Section("Global").MapTo(cfg)).Should(Succeed())
			Ω(cfg.VerifySSL).Should(BeFalse())
			Ω(cfg.APIURL).ShouldNot(BeEmpty())
		})
	})
})
