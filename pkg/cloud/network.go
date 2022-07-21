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
	capcv1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
)

type NetworkIface interface {
	ResolveNetwork(*capcv1.Network) error
}

const (
	NetOffering         = "DefaultIsolatedNetworkOfferingWithSourceNatService"
	K8sDefaultAPIPort   = 6443
	NetworkTypeIsolated = "Isolated"
	NetworkTypeShared   = "Shared"
	NetworkProtocolTCP  = "tcp"
)

// NetworkExists checks that the network already exists based on the presence of all fields.
// Assumes that the a fetch has been done on network statuses prior.
func NetworkExists(net capcv1.Network) bool {
	if net.Name != "" && net.Type != "" && net.ID != "" {
		return true
	}
	return false
}

// ResolveNetwork fetches networks' ID, Name, and Type.
func (c *client) ResolveNetwork(net *capcv1.Network) (retErr error) {
	// TODO rebuild this to consider cases with networks in many zones.
	// Use ListNetworks instead.
	netName := net.Name
	netDetails, count, err := c.cs.Network.GetNetworkByName(netName)
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		retErr = multierror.Append(retErr, errors.Wrapf(err, "could not get Network ID from %s", netName))
	} else if count != 1 {
		retErr = multierror.Append(retErr, errors.Errorf(
			"expected 1 Network with name %s, but got %d", netName, count))
	} else { // Got netID from the network's name.
		net.ID = netDetails.Id
		net.Type = netDetails.Type
		return nil
	}

	// Now get network details.
	netDetails, count, err = c.cs.Network.GetNetworkByID(net.ID)
	if err != nil {
		return multierror.Append(retErr, errors.Wrapf(err, "could not get Network by ID %s", net.ID))
	} else if count != 1 {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return multierror.Append(retErr, errors.Errorf("expected 1 Network with UUID %s, but got %d", net.ID, count))
	}
	net.Name = netDetails.Name
	net.ID = netDetails.Id
	net.Type = netDetails.Type
	return nil
}

func generateNetworkTagName(csCluster *capcv1.CloudStackCluster) string {
	return ClusterTagNamePrefix + string(csCluster.UID)
}

// RemoveClusterTagFromNetwork the cluster in use tag from a network.
func (c *client) RemoveClusterTagFromNetwork(csCluster *capcv1.CloudStackCluster, net capcv1.Network) (retError error) {
	tags, err := c.GetTags(ResourceTypeNetwork, net.ID)
	if err != nil {
		return err
	}

	ClusterTagName := generateNetworkTagName(csCluster)
	if tagValue := tags[ClusterTagName]; tagValue != "" {
		if err = c.DeleteTags(ResourceTypeNetwork, net.ID, map[string]string{ClusterTagName: tagValue}); err != nil {
			return err
		}
	}

	return nil
}
