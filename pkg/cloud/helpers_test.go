/*
Copyright 2022.

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
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path"

	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	FixturePath = "test/fixtures/cloud-config-files"
)

var _ = Describe("Helpers", func() {

	It("Gets API configuration", func() {
		Context("For a configuration with the 'Global' section missing", func() {
			filepath := getConfigPath("cloud-config-no-global")

			client, err := cloud.NewClient(filepath)
			Ω(client).Should(BeNil())
			Ω(err.Error()).Should(ContainSubstring("section Global not found"))
		})
	})

	It("should compress and encode string", func() {
		str := "Hello World"

		compressedAndEncodedData, err := cloud.CompressAndEncodeString(str)

		compressedData, _ := base64.StdEncoding.DecodeString(compressedAndEncodedData)
		reader, _ := gzip.NewReader(bytes.NewReader(compressedData))
		result, _ := ioutil.ReadAll(reader)

		Ω(err).Should(BeNil())
		Ω(string(result)).Should(Equal(str))
	})
})

func getConfigPath(filename string) string {
	dir, _ := os.Getwd()
	return path.Join(dir, FixturePath, filename)
}
