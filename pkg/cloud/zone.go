package cloud

import (
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type ZoneIFace interface {
	ResolveZone(*infrav1.CloudStackZone) error
}

func (c *client) ResolveZone(zone *infrav1.CloudStackZone) (retErr error) {
	if zoneID, count, err := c.cs.Zone.GetZoneID(zone.Name); err != nil {
		retErr = multierror.Append(retErr, errors.Wrapf(err, "could not get Zone ID from %s", zone))
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"expected 1 Zone with name %s, but got %d", zone.Name, count))
	} else {
		zone.Spec.ID = zoneID
	}

	if resp, count, err := c.cs.Zone.GetZoneByID(zone.Spec.ID); err != nil {
		return multierror.Append(retErr, errors.Wrapf(err, "could not get Zone by ID %s", zone.Spec.ID))
	} else if count != 1 {
		return multierror.Append(retErr, errors.Errorf(
			"expected 1 Zone with UUID %s, but got %d", zone.Spec.ID, count))
	} else {
		zone.Spec.Name = resp.Name
	}

	return nil
}
