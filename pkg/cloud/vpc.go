/*
Copyright 2024 The Kubernetes Authors.

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

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/pkg/errors"
)

// ResourceTypeVPC is the type identifier for VPC resources.
const (
	ResourceTypeVPC = "Vpc"
	VPCOffering     = "Default VPC offering"
)

// VPCIface defines the interface for VPC operations.
type VPCIface interface {
	ResolveVPC(*infrav1.VPC) error
	CreateVPC(*infrav1.CloudStackFailureDomain, *infrav1.VPC) error
	RemoveClusterTagFromVPC(*infrav1.CloudStackCluster, infrav1.VPC) error
	DeleteVPCIfNotInUse(infrav1.VPC) (retError error)
}

// getVPCOfferingID fetches a vpc offering id.
func (c *client) getVPCOfferingID() (string, error) {
	offeringID, count, retErr := c.cs.VPC.GetVPCOfferingID(VPCOffering)
	if retErr != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(retErr)
		return "", retErr
	} else if count != 1 {
		return "", errors.New("found more than one vpc offering")
	}
	return offeringID, nil
}

// ResolveVPC checks if the specified VPC exists by ID or name.
// If it exists, it updates the VPC struct with the resolved ID or name.
func (c *client) ResolveVPC(vpc *infrav1.VPC) error {
	if vpc == nil || (vpc.ID == "" && vpc.Name == "") {
		return nil
	}

	// If VPC ID is provided, check if it exists
	if vpc.ID != "" {
		resp, count, err := c.cs.VPC.GetVPCByID(vpc.ID, cloudstack.WithProject(c.user.Project.ID))
		if err != nil {
			c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
			return errors.Wrapf(err, "failed to get VPC with ID %s", vpc.ID)
		}
		if count == 0 {
			return errors.Errorf("no VPC found with ID %s", vpc.ID)
		}
		vpc.Name = resp.Name
		vpc.CIDR = resp.Cidr
		return nil
	}

	// If VPC name is provided, check if it exists
	resp, count, err := c.cs.VPC.GetVPCByName(vpc.Name, cloudstack.WithProject(c.user.Project.ID))
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return errors.Wrapf(err, "failed to get VPC with name %s", vpc.Name)
	}
	if count == 0 {
		return errors.Errorf("no VPC found with name %s", vpc.Name)
	}
	vpc.ID = resp.Id
	vpc.CIDR = resp.Cidr
	return nil
}

// CreateVPC creates a new VPC in CloudStack.
func (c *client) CreateVPC(fd *infrav1.CloudStackFailureDomain, vpc *infrav1.VPC) error {
	if vpc == nil || vpc.Name == "" {
		return errors.New("VPC name must be specified")
	}

	offeringID, err := c.getVPCOfferingID()
	if err != nil {
		return err
	}

	p := c.cs.VPC.NewCreateVPCParams(vpc.CIDR, vpc.Name, vpc.Name, offeringID, fd.Spec.Zone.ID)
	setIfNotEmpty(c.user.Project.ID, p.SetProjectid)
	p.SetStart(true)
	resp, err := c.cs.VPC.CreateVPC(p)
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return errors.Wrapf(err, "creating VPC with name %s", vpc.Name)
	}
	vpc.ID = resp.Id
	return c.AddCreatedByCAPCTag(ResourceTypeVPC, vpc.ID)
}

// DeleteVPC deletes a VPC.
func (c *client) DeleteVPC(vpc infrav1.VPC) error {
	_, err := c.cs.VPC.DeleteVPC(c.cs.VPC.NewDeleteVPCParams(vpc.ID))
	c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
	return errors.Wrapf(err, "deleting vpc with id %s", vpc.ID)
}

// DeleteVPCIfNotInUse deletes a VPC if the VPC is no longer in use (indicated by in use tags).
func (c *client) DeleteVPCIfNotInUse(vpc infrav1.VPC) (retError error) {
	tags, err := c.GetTags(ResourceTypeVPC, vpc.ID)
	if err != nil {
		return err
	}

	var clusterTagCount int
	for tagName := range tags {
		if strings.HasPrefix(tagName, ClusterTagNamePrefix) {
			clusterTagCount++
		}
	}

	if clusterTagCount == 0 && tags[CreatedByCAPCTagName] != "" {
		return c.DeleteVPC(vpc)
	}

	return nil
}

func generateVPCTagName(csCluster *infrav1.CloudStackCluster) string {
	return ClusterTagNamePrefix + string(csCluster.UID)
}

// RemoveClusterTagFromVPC removes the cluster in use tag from a VPC.
func (c *client) RemoveClusterTagFromVPC(csCluster *infrav1.CloudStackCluster, vpc infrav1.VPC) (retError error) {
	tags, err := c.GetTags(ResourceTypeVPC, vpc.ID)
	if err != nil {
		return err
	}

	ClusterTagName := generateVPCTagName(csCluster)
	if tagValue := tags[ClusterTagName]; tagValue != "" {
		if err = c.DeleteTags(ResourceTypeVPC, vpc.ID, map[string]string{ClusterTagName: tagValue}); err != nil {
			return err
		}
	}

	return nil
}
