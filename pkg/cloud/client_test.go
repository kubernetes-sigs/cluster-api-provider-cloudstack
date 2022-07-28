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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/helpers"
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

	Context("When fetching a YAML config.", func() {
		It("Handles the positive case.", func() {
			// Create a real cloud client.
			var connectionErr error
			_, connectionErr = helpers.NewCSClient()
			Ω(connectionErr).ShouldNot(HaveOccurred())

			_, connectionErr = cloud.NewClientFromYamlPath(os.Getenv("PROJECT_DIR")+"/cloud-config.yaml", "myendpoint")
			Ω(connectionErr).ShouldNot(HaveOccurred())
		})
	})
})
