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

package cloud

import (
	infrav1 "github.com/aws/cluster-api-provider-cloudstack-staging/api/v1alpha3"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

func (c *client) ResolveZone(csCluster *infrav1.CloudStackCluster) (retErr error) {
	if zoneID, count, err := c.cs.Zone.GetZoneID(csCluster.Spec.Zone); err != nil {
		retErr = multierror.Append(retErr, errors.Wrapf(
			err, "Could not get Zone ID from %s.", csCluster.Spec.Zone))
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"Expected 1 Zone with name %s, but got %d.", csCluster.Spec.Zone, count))
	} else {
		csCluster.Status.ZoneID = zoneID
	}

	if retErr != nil {
		if _, count, err := c.cs.Zone.GetZoneByID(csCluster.Spec.Zone); err != nil {
			return multierror.Append(retErr, errors.Wrapf(
				err, "Could not get Zone by ID %s.", csCluster.Spec.Zone))
		} else if count != 1 {
			return multierror.Append(retErr, errors.Errorf(
				"Expected 1 Zone with UUID %s, but got %d.", csCluster.Spec.Zone, count))
		} else {
			csCluster.Status.ZoneID = csCluster.Spec.Zone
		}
	}

	return nil
}

func (c *client) GetOrCreateCluster(csCluster *infrav1.CloudStackCluster) (retErr error) {
	if retErr = c.ResolveZone(csCluster); retErr != nil {
		return errors.Wrapf(retErr, "Error resolving Zone details for Cluster %s.", csCluster.Name)
	}

	// Get or create network and needed network constructs.
	if retErr = c.GetOrCreateNetwork(csCluster); retErr != nil {
		return retErr
	}

	if csCluster.Status.NetworkType == NetworkTypeIsolated {
		if retErr = c.OpenFirewallRules(csCluster); retErr != nil {
			return retErr
		}
		if retErr = c.AssociatePublicIpAddress(csCluster); retErr != nil {
			return retErr
		}
		if retErr = c.GetOrCreateLoadBalancerRule(csCluster); retErr != nil {
			return retErr
		}
	}

	// Set cluster to ready to indicate readiness to CAPI.
	csCluster.Status.Ready = true
	return nil
}
