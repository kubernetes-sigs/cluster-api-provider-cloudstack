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
	"github.com/pkg/errors"
)

const (
	AntiAffinityGroupType = "host anti-affinity"
	AffinityGroupType     = "host affinity"
)

type AffinityGroup struct {
	Type string
	Name string
	ID   string
}

type AffinityGroupIface interface {
	FetchAffinityGroup(*infrav1.CloudStackCluster, *AffinityGroup) error
	GetOrCreateAffinityGroup(*infrav1.CloudStackCluster, *AffinityGroup) error
	DeleteAffinityGroup(*AffinityGroup) error
	AssociateAffinityGroup(*infrav1.CloudStackMachine, AffinityGroup) error
	DisassociateAffinityGroup(*infrav1.CloudStackMachine, AffinityGroup) error
}

func (c *client) FetchAffinityGroup(csCluster *infrav1.CloudStackCluster, group *AffinityGroup) (reterr error) {
	p := c.cs.AffinityGroup.NewListAffinityGroupsParams()
	setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
	setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)

	if group.ID != "" {
		p.SetId(group.ID)
	} else if group.Name != "" {
		p.SetName(group.Name)
	}

	resp, err := c.cs.AffinityGroup.ListAffinityGroups(p)
	if err != nil {
		return err
	} else if resp.Count > 1 {
		return errors.Errorf("found more than one affinity group by name %s or id %s", group.Name, group.ID)
	} else if resp.Count == 0 {
		return errors.Errorf("affinity group not found by name %s or id %s", group.Name, group.ID)
	}
	group.Name = resp.AffinityGroups[0].Name
	group.Type = resp.AffinityGroups[0].Type
	return nil
}

func (c *client) GetOrCreateAffinityGroup(csCluster *infrav1.CloudStackCluster, group *AffinityGroup) (retErr error) {
	if err := c.FetchAffinityGroup(csCluster, group); err != nil { // Group not found?
		p := c.cs.AffinityGroup.NewCreateAffinityGroupParams(group.Name, group.Type)
		setIfNotEmpty(csCluster.Spec.Account, p.SetAccount)
		setIfNotEmpty(csCluster.Status.DomainID, p.SetDomainid)
		resp, err := c.cs.AffinityGroup.CreateAffinityGroup(p)
		if err != nil {
			return err
		}
		group.ID = resp.Id
	}
	return nil
}

func (c *client) DeleteAffinityGroup(group *AffinityGroup) (retErr error) {
	p := c.cs.AffinityGroup.NewDeleteAffinityGroupParams()
	setIfNotEmpty(group.ID, p.SetId)
	setIfNotEmpty(group.Name, p.SetName)
	_, retErr = c.cs.AffinityGroup.DeleteAffinityGroup(p)
	return retErr
}

type affinityGroups []AffinityGroup

func (c *client) getCurrentAffinityGroups(csMachine *infrav1.CloudStackMachine) (affinityGroups, error) {
	// Start by fetching VM details which includes an array of currently associated affinity groups.
	if virtM, count, err := c.cs.VirtualMachine.GetVirtualMachineByID(*csMachine.Spec.InstanceID); err != nil {
		return nil, err
	} else if count > 1 {
		return nil, errors.Errorf("found more than one VM for ID: %s", *csMachine.Spec.InstanceID)
	} else {
		groups := make([]AffinityGroup, 0, len(virtM.Affinitygroup))
		for _, v := range virtM.Affinitygroup {
			groups = append(groups, AffinityGroup{Name: v.Name, Type: v.Type, ID: v.Id})
		}
		return groups, nil
	}
}

func (ags *affinityGroups) toArrayOfIDs() []string {
	groupIDs := make([]string, 0, len(*ags))
	for _, group := range *ags {
		groupIDs = append(groupIDs, group.ID)
	}
	return groupIDs
}

func (ags *affinityGroups) addGroup(addGroup AffinityGroup) {
	// This is essentially adding to a set followed by array conversion.
	groupSet := map[string]AffinityGroup{addGroup.ID: addGroup}
	for _, group := range *ags {
		groupSet[group.ID] = group
	}
	*ags = make([]AffinityGroup, 0, len(groupSet))
	for _, group := range groupSet {
		*ags = append(*ags, group)
	}
}

func (ags *affinityGroups) removeGroup(removeGroup AffinityGroup) {
	// This is essentially subtracting from a set followed by array conversion.
	groupSet := map[string]AffinityGroup{}
	for _, group := range *ags {
		groupSet[group.ID] = group
	}
	delete(groupSet, removeGroup.ID)
	*ags = make([]AffinityGroup, 0, len(groupSet))
	for _, group := range groupSet {
		*ags = append(*ags, group)
	}
}

func (c *client) stopAndModifyAffinityGroups(csMachine *infrav1.CloudStackMachine, groups affinityGroups) (retErr error) {
	agp := c.cs.AffinityGroup.NewUpdateVMAffinityGroupParams(*csMachine.Spec.InstanceID)
	agp.SetAffinitygroupids(groups.toArrayOfIDs())

	p1 := c.cs.VirtualMachine.NewStopVirtualMachineParams(string(*csMachine.Spec.InstanceID))
	if _, err := c.cs.VirtualMachine.StopVirtualMachine(p1); err != nil {
		return err
	}

	if _, err := c.cs.AffinityGroup.UpdateVMAffinityGroup(agp); err != nil {
		return err
	}

	p2 := c.cs.VirtualMachine.NewStartVirtualMachineParams(string(*csMachine.Spec.InstanceID))
	_, err := c.cs.VirtualMachine.StartVirtualMachine(p2)
	return err
}

func (c *client) AssociateAffinityGroup(csMachine *infrav1.CloudStackMachine, group AffinityGroup) (retErr error) {
	groups, err := c.getCurrentAffinityGroups(csMachine)
	if err != nil {
		return err
	}
	groups.addGroup(group)
	return c.stopAndModifyAffinityGroups(csMachine, groups)
}

func (c *client) DisassociateAffinityGroup(csMachine *infrav1.CloudStackMachine, group AffinityGroup) (retErr error) {
	groups, err := c.getCurrentAffinityGroups(csMachine)
	if err != nil {
		return err
	}
	groups.removeGroup(group)
	return c.stopAndModifyAffinityGroups(csMachine, groups)
}
