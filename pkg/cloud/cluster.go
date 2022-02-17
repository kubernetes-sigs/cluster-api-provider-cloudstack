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

package cloud

import (
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type ClusterIface interface {
	GetOrCreateCluster(*infrav1.CloudStackCluster) error
}

func (c *client) resolveZone(csCluster *infrav1.CloudStackCluster) (retErr error) {
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
	if retErr = c.resolveZone(csCluster); retErr != nil {
		return errors.Wrapf(retErr, "Error resolving Zone details for Cluster %s.", csCluster.Name)
	}

	// If provided, translate Domain name to Domain ID.
	if csCluster.Spec.Domain != "" {
		domainID, count, retErr := c.cs.Domain.GetDomainID(csCluster.Spec.Domain)
		if retErr != nil {
			return retErr
		} else if count != 1 {
			return errors.Errorf("Expected 1 Domain with name %s, but got %d.", csCluster.Spec.Domain, count)
		} else {
			csCluster.Status.DomainID = domainID
		}
	}

	// Get or create network and needed network constructs.
	if retErr = c.GetOrCreateNetwork(csCluster); retErr != nil {
		return retErr
	}

	if csCluster.Status.NetworkType == NetworkTypeIsolated {
		if retErr = c.OpenFirewallRules(csCluster); retErr != nil {
			return retErr
		}
		if csCluster.Status.PublicIPID == "" { // Don't try to get public IP again it's already been fetched.
			if retErr = c.AssociatePublicIpAddress(csCluster); retErr != nil {
				return retErr
			}
		}
		if retErr = c.GetOrCreateLoadBalancerRule(csCluster); retErr != nil {
			return retErr
		}
	}

	// Set cluster to ready to indicate readiness to CAPI.
	csCluster.Status.Ready = true
	return nil
}
