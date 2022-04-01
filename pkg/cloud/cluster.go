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
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const domainPrefix = "ROOT/"

type ClusterIface interface {
	GetOrCreateCluster(*infrav1.CloudStackCluster) error
	DisposeClusterResources(cluster *infrav1.CloudStackCluster) error
	ResolveZones(*infrav1.CloudStackCluster) error
}

func (c *client) ResolveZones(csCluster *infrav1.CloudStackCluster) (retErr error) {
	for _, specZone := range csCluster.Spec.Zones {
		if zoneID, count, err := c.cs.Zone.GetZoneID(specZone.Name); err != nil {
			retErr = multierror.Append(retErr, errors.Wrapf(err, "could not get Zone ID from %s", specZone))
		} else if count != 1 {
			retErr = multierror.Append(retErr, errors.Errorf(
				"expected 1 Zone with name %s, but got %d", specZone.Name, count))
		} else {
			specZone.ID = zoneID
		}

		if resp, count, err := c.cs.Zone.GetZoneByID(specZone.ID); err != nil {
			return multierror.Append(retErr, errors.Wrapf(err, "could not get Zone by ID %s", specZone.ID))
		} else if count != 1 {
			return multierror.Append(retErr, errors.Errorf(
				"expected 1 Zone with UUID %s, but got %d", specZone.ID, count))
		} else {
			csCluster.Status.Zones[resp.Id] = infrav1.Zone{
				Name: resp.Name, ID: resp.Id, Network: specZone.Network}
		}
	}

	return nil
}

func (c *client) GetOrCreateCluster(csCluster *infrav1.CloudStackCluster) (retErr error) {
	if csCluster.Status.Zones == nil {
		csCluster.Status.Zones = make(map[string]infrav1.Zone)
	}
	if retErr = c.ResolveZones(csCluster); retErr != nil {
		return errors.Wrapf(retErr, "error resolving Zone details for Cluster %s", csCluster.Name)
	}

	csCluster.Status.FailureDomains = capiv1.FailureDomains{}
	for _, zone := range csCluster.Status.Zones {
		csCluster.Status.FailureDomains[zone.ID] = capiv1.FailureDomainSpec{ControlPlane: true}
	}

	if retErr := c.ResolveDomainAndAccount(csCluster); retErr != nil {
		return retErr
	}

	// Get current network statuses.
	// CAPC only modifies networks in the single isolated network case.
	if retErr = c.ResolveNetworkStatuses(csCluster); retErr != nil {
		return retErr
	}

	if UsesIsolatedNetwork(csCluster) {
		return c.GetOrCreateIsolatedNetwork(csCluster)
	}

	return nil
}

func (c *client) ResolveDomainAndAccount(csCluster *infrav1.CloudStackCluster) error {
	if (csCluster.Spec.Domain != "" && csCluster.Spec.Account == "") ||
		(csCluster.Spec.Domain == "" && csCluster.Spec.Account != "") {
		return errors.Errorf("Both domain and account must be specified or none of them must be specified")
	}

	if csCluster.Spec.Domain != "" && csCluster.Spec.Account != "" {
		p := c.cs.Domain.NewListDomainsParams()
		p.SetListall(true)
		resp, retErr := c.cs.Domain.ListDomains(p)
		if retErr != nil {
			return retErr
		} else {
			for _, domain := range resp.Domains {
				if domain.Path == domainPrefix+csCluster.Spec.Domain {
					csCluster.Status.DomainID = domain.Id
					break
				}
			}
		}
		if csCluster.Status.DomainID == "" {
			return errors.Errorf("domain not found for domain path %s", csCluster.Spec.Domain)
		}

		listAccountParams := c.cs.Account.NewListAccountsParams()
		listAccountParams.SetDomainid(csCluster.Status.DomainID)
		listAccountParams.SetName(csCluster.Spec.Account)
		listAccountResp, retErr := c.cs.Account.ListAccounts(listAccountParams)
		if retErr != nil {
			return retErr
		} else if listAccountResp.Count != 1 {
			return errors.Errorf("expected 1 Account with account name %s in domain ID %s, but got %d",
				csCluster.Spec.Account, csCluster.Status.DomainID, resp.Count)
		}
	}
	return nil
}

func (c *client) DisposeClusterResources(csCluster *infrav1.CloudStackCluster) (retError error) {
	if csCluster.Status.PublicIPID != "" {
		if err := c.DeleteClusterTag(ResourceTypeIPAddress, csCluster.Status.PublicIPID, csCluster); err != nil {
			return err
		}
		if err := c.DisassociatePublicIPAddressIfNotInUse(csCluster); err != nil {
			return err
		}
	}
	for _, zone := range csCluster.Status.Zones {
		if err := c.RemoveClusterTagFromNetwork(csCluster, zone.Network); err != nil {
			return err
		}
		if err := c.DeleteNetworkIfNotInUse(csCluster, zone.Network); err != nil {
			return err
		}
	}
	return nil
}
