package cloud

import (
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type ZoneIFace interface {
	ResolveZone(*infrav1.CloudStackZone) error
	ResolveNetworkForZone(*infrav1.CloudStackZone) error
}

func (c *client) ResolveZone(zone *infrav1.CloudStackZone) (retErr error) {
	if zoneID, count, err := c.cs.Zone.GetZoneID(zone.Name); err != nil {
		retErr = multierror.Append(retErr, errors.Wrapf(err, "could not get Zone ID from %v", zone))
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"expected 1 Zone with name %s, but got %d", zone.Name, count))
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
