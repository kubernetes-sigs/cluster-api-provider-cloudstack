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

package cloud

import (
	"fmt"
	"os"
	"path"
	"testing"

	_ "github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

const (
	FixturePath = "test/fixtures/cloud-config-files"
)

func TestHelpers(t *testing.T) {

	t.Run("Getting API configuration", func(t *testing.T) {
		t.Run("Section Global not found", func(t *testing.T) {
			g := NewWithT(t)
			filepath := getConfigPath("cloud-config-no-global")
			expectedErr := fmt.Errorf("section Global not found")

			if _, _, _, err := ReadAPIConfig(filepath); err != nil {
				g.Expect(errors.Cause(err)).To(MatchError(expectedErr))
			} else {
				t.Error()
			}
		})

		t.Run("Good configuration", func(t *testing.T) {
			g := NewWithT(t)
			filepath := getConfigPath("cloud-config-good")

			if apiURL, apiKey, secretKey, err := ReadAPIConfig(filepath); err != nil {
				t.Error()
			} else {
				g.Expect(apiURL).To(Equal("api-url1"))
				g.Expect(apiKey).To(Equal("api-key1"))
				g.Expect(secretKey).To(Equal("secret-key1"))
			}
		})
	})
}

func getConfigPath(filename string) string {
	dir, _ := os.Getwd()
	return path.Join(dir, FixturePath, filename)
}
