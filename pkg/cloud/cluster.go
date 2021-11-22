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
	"github.com/pkg/errors"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
)

func CreateCluster(cs *cloudstack.CloudStackClient, csCluster *infrav1.CloudStackCluster) (retErr error) {
	var count int

	// Translate zone name to  zone ID.
	csCluster.Status.ZoneID, count, retErr = cs.Zone.GetZoneID(csCluster.Spec.Zone)
	if retErr != nil {
		return retErr
	} else if count != 1 {
		return errors.Errorf("Expected 1 zone with name %s, but got %d.", csCluster.Spec.Zone, count)
	}

	// Get or create network and needed network constructs.
	if retErr = CreateNetwork(cs, csCluster); retErr != nil {
		return retErr
	}
	if retErr = OpenFirewallRules(cs, csCluster); retErr != nil {
		return retErr
	}
	if retErr = AssociatePublicIpAddress(cs, csCluster); retErr != nil {
		return retErr
	}
	if retErr = CreateLoadBalancerRule(cs, csCluster); retErr != nil {
		return retErr
	}

	// Set cluster to ready to indicate readiness to CAPI.
	csCluster.Status.Ready = true
	return nil
}
