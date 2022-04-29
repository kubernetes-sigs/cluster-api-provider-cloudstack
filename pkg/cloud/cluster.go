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
	"strings"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const rootDomain = "ROOT"
const domainDelimiter = "/"

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

	if retErr := c.ResolveDomainAndAccount(csCluster); retErr != nil {
		return retErr
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
		tokens := strings.Split(csCluster.Spec.Domain, domainDelimiter)
		domainName := tokens[len(tokens)-1]
		p.SetListall(true)
		p.SetName(domainName)

		resp, retErr := c.cs.Domain.ListDomains(p)
		if retErr != nil {
			return retErr
		}

		var domainPath string
		if csCluster.Spec.Domain == rootDomain {
			domainPath = rootDomain
		} else {
			domainPath = strings.Join([]string{rootDomain, csCluster.Spec.Domain}, domainDelimiter)
		}
		for _, domain := range resp.Domains {
			if domain.Path == domainPath {
				csCluster.Status.DomainID = domain.Id
				break
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
	return nil
}
