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
	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
)

func FetchClusterInfo(cs *cloudstack.CloudStackClient, csCluster *infrav1.CloudStackCluster) (retErr error) {
	csCluster.Status.ZoneID, _, retErr = cs.Zone.GetZoneID(csCluster.Spec.Zone)
	if retErr != nil {
		return retErr
	}
	csCluster.Status.NetworkID, _, retErr = cs.Network.GetNetworkID(csCluster.Spec.Network)
	if retErr != nil {
		return retErr
	}
	csCluster.Status.Ready = true
	return retErr
}
