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
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
)

type ZoneIFace interface {
	ResolveZone(*infrav1.CloudStackZone) error
	ResolveNetworkForZone(*infrav1.CloudStackZone) error
}

func (c *client) ResolveZone(zone *infrav1.CloudStackZone) (retErr error) {
	if zoneID, count, err := c.cs.Zone.GetZoneID(zone.Spec.Name); err != nil {
		retErr = multierror.Append(retErr, errors.Wrapf(err, "could not get Zone ID from %v", zone))
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"expected 1 Zone with name %s, but got %d", zone.Spec.Name, count))
	} else {
		zone.Spec.ID = zoneID
	}

	if resp, count, err := c.cs.Zone.GetZoneByID(zone.Spec.ID); err != nil {
		return multierror.Append(retErr, errors.Wrapf(err, "could not get Zone by ID %v", zone.Spec.ID))
	} else if count != 1 {
		return multierror.Append(retErr, errors.Errorf(
			"expected 1 Zone with UUID %s, but got %d", zone.Spec.ID, count))
	} else {
		zone.Spec.Name = resp.Name
	}

	return nil
}

// ResolveNetworkForZone fetches details on Zone's specified network.
func (c *client) ResolveNetworkForZone(zone *infrav1.CloudStackZone) (retErr error) {
	// TODO rebuild this to consider cases with networks in many zones.
	// Use ListNetworks instead.
	netName := zone.Spec.Network.Name
	netDetails, count, err := c.cs.Network.GetNetworkByName(netName)
	if err != nil {
		retErr = multierror.Append(retErr, errors.Wrapf(err, "could not get Network ID from %v", netName))
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"expected 1 Network with name %s, but got %d", netName, count))
	} else { // Got netID from the network's name.
		zone.Spec.Network.ID = netDetails.Id
		zone.Spec.Network.Type = netDetails.Type
		return nil
	}

	// Now get network details.
	netDetails, count, err = c.cs.Network.GetNetworkByID(zone.Spec.Network.ID)
	if err != nil {
		return multierror.Append(retErr, errors.Wrapf(err, "could not get Network by ID %s", zone.Spec.Network.ID))
	} else if count != 1 {
		return multierror.Append(retErr, errors.Errorf("expected 1 Network with UUID %v, but got %d", zone.Spec.Network.ID, count))
	}
	zone.Spec.Network.Name = netDetails.Name
	zone.Spec.Network.ID = netDetails.Id
	zone.Spec.Network.Type = netDetails.Type
	return nil
}
