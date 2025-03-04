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
const ResourceTypeVPC = "VPC"

// VPCIface defines the interface for VPC operations.
type VPCIface interface {
	ResolveVPC(*infrav1.VPC) error
	CreateVPC(*infrav1.VPC) error
	GetOrCreateVPC(*infrav1.VPC) error
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
	return nil
}

// GetOrCreateVPC ensures a VPC exists for the given specification.
// If the VPC doesn't exist, it creates a new one.
func (c *client) GetOrCreateVPC(vpc *infrav1.VPC) error {
	if vpc == nil || (vpc.ID == "" && vpc.Name == "") {
		return nil
	}

	// Try to resolve the VPC
	err := c.ResolveVPC(vpc)
	if err != nil {
		// If it's a "not found" error and we have a name, create the VPC
		if strings.Contains(err.Error(), "no VPC found") && vpc.Name != "" {
			return c.CreateVPC(vpc)
		}
		return err
	}

	return nil
}

// CreateVPC creates a new VPC in CloudStack.
func (c *client) CreateVPC(vpc *infrav1.VPC) error {
	if vpc == nil || vpc.Name == "" {
		return errors.New("VPC name must be specified")
	}

	// Get VPC offering ID
	p := c.cs.VPC.NewListVPCOfferingsParams()
	resp, err := c.cs.VPC.ListVPCOfferings(p)
	if err != nil {
		c.customMetrics.EvaluateErrorAndIncrementAcsReconciliationErrorCounter(err)
		return errors.Wrap(err, "failed to list VPC offerings")
	}
	if resp.Count == 0 {
		return errors.New("no VPC offerings available")
	}

	// Since the SDK's VPC creation API might have compatibility issues with different CloudStack versions,
	// we'll need to handle this in the implementation of the network creation rather than here.
	// For now, we'll just return a "not implemented" error, and handle VPC creation in the isolated network creation.
	return errors.New("creating VPC not directly implemented; handled in isolated network creation")
}
