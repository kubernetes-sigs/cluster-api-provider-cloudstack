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
	"testing"

	_ "github.com/golang/mock/gomock"
	_ "github.com/onsi/gomega"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
)

// Create a network.
func TestCreateNetwork(t *testing.T) {
	t.Run("Create Instance", func(t *testing.T) {
		cs := newCSClient()

		csCluster := &infrav1.CloudStackCluster{
			Spec: infrav1.CloudStackClusterSpec{
				Zone:    "zone1",
				Network: "GuestNet1",
			},
		}
		err := FetchNetwork(cs, csCluster)
		if err != nil {
			t.Error(err)
		}

		csCluster.Spec.Network = "EphemeralNet"

		// ZoneID required for net creation.
		FetchClusterInfo(cs, csCluster)

		err = CreateNetwork(cs, csCluster)
		if err != nil {
			t.Error(err)
		}

		err = OpenFirewallRules(cs, csCluster)
		if err != nil {
			t.Error(err)
		}

		err = FetchPublicIP(cs, csCluster)
		if err != nil {
			t.Error(err)
		}

		err = CreateLoadBalancerRule(cs, csCluster)
		if err != nil {
			t.Error(err)
		}

		err = DestroyNetwork(cs, csCluster)
		if err != nil {
			t.Error(err)
		}
	})
}
