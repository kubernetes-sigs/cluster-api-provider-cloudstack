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

/*
	These tests assume a running CloudStack instance and associated cloud-config file.

	They are not proper, but they're enough for now.

	TODO Use mocks and testing frameworks.
*/
package cloud

import (
	"encoding/json"
	"fmt"
	"testing"

	_ "github.com/golang/mock/gomock"
	_ "github.com/onsi/gomega"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
)

func TestClusterInfo(t *testing.T) {
	cs := newCSClient()
	t.Run("Get Cluster Information", func(t *testing.T) {
		c := &infrav1.CloudStackCluster{
			Spec: infrav1.CloudStackClusterSpec{
				Zone:    "zone1",
				Network: "guestNet1"}}

		if err := FetchClusterInfo(cs, c); err != nil {
			t.Error(err)
		}
		cJSON, _ := json.MarshalIndent(c, "", "  ")
		fmt.Println(string(cJSON))
	})
}
