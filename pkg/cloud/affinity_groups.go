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
	"errors"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1alpha3"
)

const (
	AntiAffinityGroupType = "host anti-affinity"
	AffinityGroupType     = "host affinity"
)

type AffinityGroup struct {
	Type string
	Name string
	Id   string
}

type AffinityGroupIFace interface {
	GetOrCreateAffinityGroup(*infrav1.CloudStackCluster, AffinityGroup) error
	DeleteAffinityGroup(*infrav1.CloudStackCluster, AffinityGroup) error
	AssociateAffinityGroup(*infrav1.CloudStackMachine, AffinityGroup) error
	DissassociateAffinityGroup(*infrav1.CloudStackMachine, AffinityGroup) error
}

func (c *client) FetchAffinityGroup(csCluster *infrav1.CloudStackCluster, group AffinityGroup) (reterr error) {
	if group.Id != "" {
		affinityGroup, count, err := c.cs.AffinityGroup.GetAffinityGroupByID(group.Id)
		if err != nil {
			// handle via multierr
			return err
		} else if count > 1 {
			// handle via creating a new error.
			return errors.New("Count bad")
		} else {
			group.Id = affinityGroup.Id
			group.Type = affinityGroup.Type
			return nil
		}
	}
	if group.Name != "" {
		affinityGroup, count, err := c.cs.AffinityGroup.GetAffinityGroupByName(group.Name)
		if err != nil {
			// handle via multierr
			return err
		} else if count > 1 {
			// handle via creating a new error.
			return errors.New("Count bad")
		} else {
			group.Id = affinityGroup.Id
			group.Type = affinityGroup.Type
			return nil
		}
	}
	return nil
}

func (c *client) GetOrCreateAffinityGroup(csCluster *infrav1.CloudStackCluster, group AffinityGroup) (retErr error) {
	if err := c.FetchAffinityGroup(csCluster, group); err != nil { // Group not found?
		p := c.cs.AffinityGroup.NewCreateAffinityGroupParams(group.Name, group.Type)
		setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
		setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)
		if resp, err := c.cs.AffinityGroup.CreateAffinityGroup(p); err != nil {
			return err
		} else {
			group.Id = resp.Id
		}
	}
	return nil
}

func (c *client) DeleteAffinityGroup(csCluster *infrav1.CloudStackCluster, group AffinityGroup) (retErr error) {
	p := c.cs.AffinityGroup.NewDeleteAffinityGroupParams()
	setIfNotEmpty(group.Id, p.SetId)
	setIfNotEmpty(group.Name, p.SetName)
	setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
	setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)
	_, retErr = c.cs.AffinityGroup.DeleteAffinityGroup(p)
	return retErr
}

func (c *client) AssociateAffinityGroup(csCluster *infrav1.CloudStackMachine, group AffinityGroup) (retErr error) {
	return nil
}

func (c *client) DissassociateAffinityGroup(csCluster *infrav1.CloudStackMachine, group AffinityGroup) (retErr error) {
	return nil
}
