/*
Copyright 2021.

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
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/pkg/cloud"
)

const (
	FixturePath = "test/fixtures/cloud-config-files"
)

var _ = Describe("Helpers", func() {

	It("Gets API configuration", func() {
		Context("For a configuration with the 'Global' section missing", func() {
			filepath := getConfigPath("cloud-config-no-global")
			expectedErr := fmt.Errorf("section Global not found")

			_, _, _, err := cloud.ReadAPIConfig(filepath)
			Ω(err).Should(Equal(expectedErr))
		})

		Context("For a good configuration", func() {
			filepath := getConfigPath("cloud-config-good")

			apiURL, apiKey, secretKey, err := cloud.ReadAPIConfig(filepath)
			Ω(err).Should(BeNil())
			Ω(apiURL).Should(Equal("api-url1"))
			Ω(apiKey).Should(Equal("api-key1"))
			Ω(secretKey).Should(Equal("secret-key1"))
		})
	})
})

func getConfigPath(filename string) string {
	dir, _ := os.Getwd()
	return path.Join(dir, FixturePath, filename)
}
