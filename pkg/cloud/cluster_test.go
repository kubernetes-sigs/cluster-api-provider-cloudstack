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
