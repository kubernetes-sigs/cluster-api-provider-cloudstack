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
	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
)

type ZoneIFace interface {
	ResolveZone(*infrav1.CloudStackZoneSpec) error
	ResolveNetworkForZone(*infrav1.CloudStackZoneSpec) error
}

func (c *client) ResolveZone(zSpec *infrav1.CloudStackZoneSpec) (retErr error) {
	if zoneID, count, err := c.cs.Zone.GetZoneID(zSpec.Name); err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		retErr = multierror.Append(retErr, errors.Wrapf(err, "could not get Zone ID from %v", zSpec.Name))
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"expected 1 Zone with name %s, but got %d", zSpec.Name, count))
	} else {
		zSpec.ID = zoneID
	}

	resp, count, err := c.cs.Zone.GetZoneByID(zSpec.ID)
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return multierror.Append(retErr, errors.Wrapf(err, "could not get Zone by ID %v", zSpec.ID))
	} else if count != 1 {
		return multierror.Append(retErr, errors.Errorf(
			"expected 1 Zone with UUID %s, but got %d", zSpec.ID, count))
	}
	zSpec.Name = resp.Name

	return nil
}

// ResolveNetworkForZone fetches details on Zone's specified network.
func (c *client) ResolveNetworkForZone(zSpec *infrav1.CloudStackZoneSpec) (retErr error) {
	netName := zSpec.Network.Name
	netDetails, count, err := c.cs.Network.GetNetworkByName(netName, cloudstack.WithProject(c.user.Project.ID))
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		retErr = multierror.Append(retErr, errors.Wrapf(err, "could not get Network ID from %v", netName))
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"expected 1 Network with name %s, but got %d", netName, count))
	} else { // Got netID from the network's name.
		zSpec.Network.ID = netDetails.Id
		zSpec.Network.Type = netDetails.Type
		return nil
	}

	// Now get network details.
	netDetails, count, err = c.cs.Network.GetNetworkByID(zSpec.Network.ID, cloudstack.WithProject(c.user.Project.ID))
	if err != nil {
		return multierror.Append(retErr, errors.Wrapf(err, "could not get Network by ID %s", zSpec.Network.ID))
	} else if count != 1 {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return multierror.Append(retErr, errors.Errorf("expected 1 Network with UUID %v, but got %d", zSpec.Network.ID, count))
	}
	zSpec.Network.Name = netDetails.Name
	zSpec.Network.ID = netDetails.Id
	zSpec.Network.Type = netDetails.Type
	return nil
}
